# SQL API (canonical signatures)

This is the canonical contract for the `helix_ai` SQL functions. helix-protocol owns the signatures;
helix-kernel owns the implementation (RFC-0001, RFC-0003) and must not diverge from this doc without a
PR here first. Schema/role names per `helix-atlas` `decisions/0002-schema-and-role-names.md`.

## Registration

```sql
helix_ai.register_source(
    tenant        text,
    relation      regclass,
    key_columns   name[],      -- composite key supported; tables without a stable key are rejected
    model_binding jsonb,       -- {model_id, revision, dims?, metric?}
    pipeline_version text
) RETURNS helix_ai.source_id
```

A plain `SELECT` against the resulting enriched view reads the stored sidecar value and status — it never
triggers processing (Part 2 §2).

## Status

```sql
helix_ai.status(source_id) RETURNS TABLE (
    source_key   text,
    status       helix_ai.pseudo_column_status,  -- pending|ready|stale|failed|deleted
    generated_at timestamptz
)
```

## Refresh (enqueue)

```sql
helix_ai.request_refresh(source_id, source_key) RETURNS helix_ai.job_id
```
Writes an outbox row; see RFC-0003 for the job contract fields and ADR-0003 for how the row gets there
(trigger vs polling, mode-dependent).

## Claim (worker role only)

```sql
helix_ai.claim_job(worker_role, capabilities jsonb)
    RETURNS helix_ai.job  -- or no row if nothing claimable
```
Increments `lease_epoch`, sets `deadline = now() + lease_duration`. See RFC-0003 §Claim.

## Publish (worker role only, compare-and-set)

```sql
helix_ai.publish_result(
    job_id                  uuid,
    lease_epoch             bigint,
    expected_source_digest  text,
    expected_model          text,
    expected_pipeline       text,
    value                   jsonb
) RETURNS helix_ai.publish_outcome  -- ok | stale_lease | stale_source | model_mismatch
                                    --      | pipeline_mismatch | tombstoned
```
One short conditional transaction. See RFC-0003 §Publish and `errors/codes.md` for the outcome codes,
which are the same codes surfaced over REST.

## Privilege boundary

Only the narrow worker role (RFC-0001/ADR-0002) may `EXECUTE` `claim_job`/`publish_result`, and that
role's only write path into `helix_ai` is through these two functions — it cannot `UPDATE`/`DELETE`
sidecar or job rows directly, and has no privilege on application tables at all.
