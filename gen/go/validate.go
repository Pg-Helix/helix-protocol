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
