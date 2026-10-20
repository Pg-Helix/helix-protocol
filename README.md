# helix-protocol

Versioned SQL/API/event schemas, generated clients, Go SDK and conformance fixtures.

| | |
|---|---|
| Ownership group | Product Experience and Documentation |
| Primary form | Schemas / Go |
| Visibility | PUBLIC |
| Status | **M1: first contract (Phase 1 P0).** REST contract, generated types and SDK exist; no service implements this contract yet. |

Design source of truth: the private `helix-atlas` repository (`architecture/repos/helix-protocol.md`,
`rfcs/0003-pseudo-column-lifecycle-and-worker-protocol.md`, `rfcs/0004-retrieval-citation-and-acl.md`).
Organization master prompt: PgHelix Organization Master Prompt, 16 Oct 2026.

## Layout

- `openapi/openapi.yaml` — the REST contract (collections, search, answers, model registry) plus the
  RFC-0003 job contract and RFC-0004 chunk-identity schemas, kept here so one tool generates every
  consumer's types.
- `gen/go/` — generated Go types. **Do not hand-edit `types.go`**; run `go generate ./...` to regenerate
  from `openapi/openapi.yaml`. Generation is deterministic: running it twice produces byte-identical
  output, checked by CI.
- `sdk/go/` — minimal hand-written Go client (context deadlines/cancellation via the caller's `context.Context`,
  typed `*APIError`, cursor pagination, idempotency-key passthrough).
- `errors/codes.md` — the stable error code list (request-level + RFC-0003 publish rejection codes).
- `sql/api.md` — canonical SQL function signatures (`register_source`, `claim_job`, `publish_result`);
  helix-kernel implements, this repo owns the signature.
- `conformance/` — fixtures consumed by every language/repo implementing this contract, plus a Go test
  that round-trips them through the generated types.

## Build and test

```
go generate ./...   # regenerate gen/go/types.go from openapi/openapi.yaml
go build ./...
go vet ./...
go test ./... -race
```

Requires Go 1.26+. No database or network access needed for any of the above.

## License

Not yet decided. The license is an ADR in `helix-atlas`; until it is accepted, no license is granted and all rights are reserved.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).
