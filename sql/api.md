# SQL API (canonical signatures)

This is the canonical contract for the `helix_ai` SQL functions. helix-protocol owns the signatures;
helix-kernel owns the implementation (RFC-0001, RFC-0003, ADR-0006) and must not diverge from this doc
without a PR here first. Schema/role names per `helix-atlas` `decisions/0002-schema-and-role-names.md`
and `decisions/0006-k2-k3-security-and-contract-corrections.md`.

**Signature history note:** earlier drafts of this doc specified `key_columns`/`model_binding` as
caller-supplied parameters and a `status(source_id)` *function*. The implemented kernel instead
auto-detects the primary key (never caller-supplied, so registration can never claim a key that drifts
from the table's real one) and exposes `status` as a `security_invoker` *view* joining
`helix_ai.source`/`derived_value` directly. Both are intentional, reviewed choices (ADR-0006), not
implementation drift this doc is correcting the implementation toward — this doc is corrected to match
the implementation here instead.

## Registration

```sql
helix_ai.register_source(
    schema           name,
    table            name,
    model_id         text,
    model_revision   text,
    pipeline_version text,
    dims             integer DEFAULT NULL,  -- embedding bindings only
    metric           text    DEFAULT NULL  -- embedding bindings only
) RETURNS bigint  -- helix_ai.source.id
```
`tenant` is never a parameter — it is read from the session (`helix.tenant`), so a connection cannot
register on behalf of a tenant it has not been placed into. The row key is auto-detected from the
table's own `PRIMARY KEY` (composite keys supported); a table with no primary key is rejected.

PUBLIC may `EXECUTE` this (an ordinary application role can self-register without direct table grants),
but the caller must already hold `TRIGGER` privilege on the target relation itself (ADR-0006 decision 2)
— otherwise any role could use this `SECURITY DEFINER` function to attach a trigger, and a derived view,
to a table it has no privilege on. A plain `SELECT` against the resulting enriched view (`<table>_ai`)
reads the stored sidecar value and status — it never triggers processing.

## Status

```sql
helix_ai.status WITH (security_invoker = true) AS -- VIEW, not a function
  SELECT source_id, tenant, schema_name, table_name, source_key, status, source_digest, value, generated_at
```
`security_invoker` so a querying role's own RLS/grants on `helix_ai.source`/`derived_value` apply, not
this view's owner's.

## Refresh (enqueue)

```sql
helix_ai.request_refresh(source_id bigint, source_key jsonb) RETURNS uuid  -- job_id
```
Writes an outbox row by re-reading the row's *current* digest; see RFC-0003 for the job contract fields
and ADR-0003 for how the row gets there (trigger vs polling, mode-dependent). PUBLIC may `EXECUTE` this;
`helix_ai.source` is tenant-RLS'd and (ADR-0006 decision 1) the extension is owned by a `NOSUPERUSER
NOBYPASSRLS` role so that RLS actually constrains it — a caller cannot refresh another tenant's source
merely by knowing its `source_id`. Rejected once the key is tombstoned (`deleted`).

## Claim (worker role only)

```sql
helix_ai.claim_job(target_model text, lease_seconds integer DEFAULT 30, max_attempts integer DEFAULT 5)
    RETURNS helix_ai.job  -- or no row if nothing claimable
```
`target_model` (ADR-0006 decision 6) is required: a worker only ever claims jobs for the model it
declares itself able to serve (`WHERE target_model = target_model`). Increments `lease_epoch`, sets
`deadline = now() + lease_seconds`. A job whose `attempts` just exceeded `max_attempts` is dead-lettered
(`dead_letter = true`, sidecar marked `failed`) instead of re-leased. See RFC-0003 §Claim.

## Bounded authorized model input (worker role only)

```sql
helix_ai.get_job_input(job_id uuid, lease_epoch integer) RETURNS jsonb
```
ADR-0006 decision 7 — the contract RFC-0001/0002 left unspecified: how a worker gets data to run
inference on without any `SELECT` grant on application tables. Re-validates the caller actually holds
this exact job's current lease (not completed, not dead-lettered, `lease_epoch` matches), sets
`helix.tenant` from the job's own recorded tenant, and returns *only* the current source row at that
job's key as one jsonb object. Rejects rows over 1 MiB (`pg_column_size`) with `input_too_large` rather
than truncating. See `errors/codes.md` §Bounded-input codes.

## Publish (worker role only, compare-and-set)

```sql
helix_ai.publish_result(
    job_id              uuid,
    lease_epoch         integer,
    expected_digest     text,
    expected_model      text,
    expected_pipeline   text,
    value               jsonb
) RETURNS helix_ai.publish_outcome  -- ok | stale_lease | stale_source | model_mismatch
                                    --      | pipeline_mismatch | tombstoned | dead_lettered
```
One short conditional transaction, gated additionally (ADR-0006 decision 3) by
`generation = job's frozen source_generation` — the row-incarnation fence that stops a job issued
before a delete+identical-reinsert from CAS-publishing against the reinserted row just because its
digest happens to match again. `dead_lettered` (decision 5) rejects a job whose retry budget already
exhausted, even if the epoch it presents is still current. See RFC-0003 §Publish and `errors/codes.md`
for the outcome codes, which are the same codes surfaced over REST.

## Workflow 2 (ADR-0007): no SQL signature changes

ADR-0007 (helix-atlas) adds ingestion/retrieval/evidence-lookup REST endpoints
(`openapi/openapi.yaml`) and a new `helix_cortex_api` database role, but makes
**no change to any function above**. `cortex.chunk` is `register_source`'d
like any other application table; chunk embeddings flow through the exact
`claim_job` → `get_job_input` → `publish_result` sequence already documented
here, unmodified. See ADR-0007 for the new `cortex`-owned schema, the role
boundary between `helix_cortex_api` (HTTP-serving, no access to
`claim_job`/`publish_result`/`get_job_input`) and `helix_worker` (unchanged),
and the mapping between cortex's document/version identity and this schema's
`source_id`/`generation`.

## Privilege boundary

Only the narrow worker role, **`helix_worker`** (ADR-0006 decision 1 — the name RFC-0001/RFC-0003 cited
ADR-0002 as defining, finalized here), may `EXECUTE` `claim_job`/`publish_result`/`get_job_input`, and
that role's only write path into `helix_ai` is through these functions — it cannot `UPDATE`/`DELETE`
sidecar or job rows directly, and has no privilege on application tables at all. The extension itself is
owned by **`helix_owner`** (`NOLOGIN NOSUPERUSER NOBYPASSRLS`): no running process holds this role, and
its non-superuser status is what makes `FORCE ROW LEVEL SECURITY` on `helix_ai.source`/`derived_value`
actually bind for every `SECURITY DEFINER` function above.
