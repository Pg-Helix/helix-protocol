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

Source of truth: `helix-atlas` `rfcs/0003-pseudo-column-lifecycle-and-worker-protocol.md`.
