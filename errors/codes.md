# Error codes

Every non-2xx response from the REST API uses `ErrorResponse` (`openapi/openapi.yaml`):
`{code, message, trace_id}`. `code` is stable and machine-matched by SDK consumers; `message`
is for humans and may change. Two families:

## Request-level codes
Returned by any endpoint.

| Code | Meaning | Retryable |
|---|---|---|
| `invalid_argument` | Request failed validation (missing/malformed field) | no |
| `not_found` | Referenced tenant, collection, model or chunk does not exist | no |
| `unauthorized` | Missing or invalid credential | no |
| `forbidden` | Credential valid, but lacks tenant/object access | no |
| `tenant_mismatch` | Request's tenant does not match the caller's scope | no |
| `rate_limited` | Caller exceeded a quota or concurrency limit | yes, after backoff |
| `internal` | Unexpected server-side failure | yes |

## Publish rejection codes (RFC-0003)
Returned by `publish_result`'s compare-and-set and surfaced through the SDK as typed errors, not as
`internal`. These are expected, routine outcomes — a late worker or a changed source row is normal
operation, not a bug.

| Code | Meaning | Retryable |
|---|---|---|
| `stale_lease` | `lease_epoch` presented does not match the job's current epoch (re-leased after deadline) | no — the job was already re-leased; re-claim if work is still needed |
| `stale_source` | Source row's digest changed since the job was read | no — re-enqueue against the new digest |
| `model_mismatch` | `target_model` no longer matches the job | no |
| `pipeline_mismatch` | `pipeline_version` no longer matches the job | no |
| `tombstoned` | Source key was deleted; publish is rejected to prevent resurrection | no |
| `dead_lettered` | Job already exhausted its retry budget (`dead_letter = true`); a worker that held the lease up to that point cannot publish successfully even at a still-current epoch | no |

Source of truth: `helix-atlas` `rfcs/0003-pseudo-column-lifecycle-and-worker-protocol.md`, `decisions/0006-k2-k3-security-and-contract-corrections.md`.

## Bounded-input codes (ADR-0006, helix-atlas)

Returned by `helix_ai.get_job_input`, the function that gives a worker its authorized model input
for a claimed job (RFC-0001/0002 never specified this contract; ADR-0006 decision 7 fills the gap).

| Code | Meaning | Retryable |
|---|---|---|
| `lease_invalid` | `job_id`/`lease_epoch` do not identify a job the caller currently holds the lease on (wrong epoch, completed, or dead-lettered) | no |
| `input_too_large` | Current source row exceeds the documented size cap (1 MiB via `pg_column_size`); rejected rather than silently truncated | no |

## Workflow 2 ingestion/retrieval codes (ADR-0007, helix-atlas)

Returned by the ingest, delete, permission and evidence-lookup operations.

| Code | Meaning | Retryable |
|---|---|---|
| `unsupported_content_type` | Document `content_type` is not `text/plain` or `text/markdown` (remote URL, HTML, PDF/OCR are out of scope for this milestone) | no |
| `content_too_large` | Document bytes, chunk count, or an individual chunk exceed the documented bound | no |
| `chunk_limit_exceeded` | Chunking the document would exceed the documented maximum chunk count | no |
| `citation_invalid` | A citation's chunk/document/generation/digest no longer matches current state (superseded, deleted, revoked, or fabricated) — rejected without disclosing any private text or metadata | no |

Source of truth: `helix-atlas` `decisions/0007-workflow2-ingestion-retrieval-identity-and-authorization.md`.
