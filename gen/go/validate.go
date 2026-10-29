package protocol

import (
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Validate checks JobContract's required fields/enums/bounds (openapi.yaml's
// `required` list plus the bounds on attempts/lease_epoch/source_generation).
// encoding/json does not enforce "required" or minimums on its own — see
// conformance/fixtures_test.go's TestJobContractMissingRequiredField for that
// documented gap — so a caller that needs real enforcement must call this
// explicitly after unmarshaling.
func (j JobContract) Validate() error {
	switch {
	case j.Tenant == "":
		return fmt.Errorf("job contract: tenant is required")
	case j.JobId == (openapi_types.UUID{}):
		return fmt.Errorf("job contract: job_id is required")
	case j.SourceId == "":
		return fmt.Errorf("job contract: source_id is required")
	case len(j.SourceKey) == 0:
		return fmt.Errorf("job contract: source_key is required")
	case j.SourceDigest == "":
		return fmt.Errorf("job contract: source_digest is required")
	case j.SourceGeneration < 1:
		return fmt.Errorf("job contract: source_generation must be >= 1, got %d", j.SourceGeneration)
	case j.PipelineVersion == "":
		return fmt.Errorf("job contract: pipeline_version is required")
	case j.TargetModel == "":
		return fmt.Errorf("job contract: target_model is required")
	case j.Attempts < 0:
		return fmt.Errorf("job contract: attempts must be >= 0, got %d", j.Attempts)
	case j.LeaseEpoch < 0:
		return fmt.Errorf("job contract: lease_epoch must be >= 0, got %d", j.LeaseEpoch)
	case j.TraceId == "":
		return fmt.Errorf("job contract: trace_id is required")
	}
	return nil
}

// Validate checks IngestDocumentJSONBody's required fields (ADR-0007/C2).
// tenant is intentionally still checked here even though the authenticated
// caller's tenant is the only one ever trusted server-side — a request that
// supplies a conflicting tenant is rejected as tenant_mismatch by the
// server, not by this function, which only checks shape. content_type's
// enum is deliberately NOT checked here: errors/codes.md gives an
// unsupported content type its own stable code (unsupported_content_type),
// and the server's store layer is what produces it -- duplicating the
// check here would instead surface it as a generic invalid_argument.
func (b IngestDocumentJSONBody) Validate() error {
	switch {
	case b.Tenant == "":
		return fmt.Errorf("ingest document: tenant is required")
	case b.Name == "":
		return fmt.Errorf("ingest document: name is required")
	case b.Content == "":
		return fmt.Errorf("ingest document: content is required")
	}
	return nil
}

// Validate checks Permission's required fields/enum.
func (p Permission) Validate() error {
	switch {
	case p.Principal == "":
		return fmt.Errorf("permission: principal is required")
	case p.Role != Owner && p.Role != Reader:
		return fmt.Errorf("permission: role must be owner or reader, got %q", p.Role)
	}
	return nil
}

// Validate checks Document's required fields/enums.
func (d Document) Validate() error {
	switch {
	case d.Id == "":
		return fmt.Errorf("document: id is required")
	case d.CollectionId == "":
		return fmt.Errorf("document: collection_id is required")
	case d.Tenant == "":
		return fmt.Errorf("document: tenant is required")
	case d.Digest == "":
		return fmt.Errorf("document: digest is required")
	case d.ChunkerVersion == "":
		return fmt.Errorf("document: chunker_version is required")
	}
	return nil
}

// Validate checks EvidenceResponse's required fields.
func (e EvidenceResponse) Validate() error {
	switch {
	case e.ChunkId == "":
		return fmt.Errorf("evidence response: chunk_id is required")
	case e.DocumentId == "":
		return fmt.Errorf("evidence response: document_id is required")
	case e.DocumentVersionDigest == "":
		return fmt.Errorf("evidence response: document_version_digest is required")
	}
	return nil
}
