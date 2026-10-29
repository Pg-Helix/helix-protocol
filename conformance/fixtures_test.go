// Package conformance loads the fixtures in fixtures/ and checks them against
// the generated types (gen/go/types.go). It is the drift check between
// openapi/openapi.yaml and any hand-maintained consumer: every component
// consumed across repos round-trips through exactly this fixture set.
package conformance

import (
	"embed"
	"encoding/json"
	"reflect"
	"testing"

	protocol "github.com/Pg-Helix/helix-protocol/gen/go"
)

//go:embed fixtures/*.json
var fixturesFS embed.FS

func load(t *testing.T, name string) []byte {
	t.Helper()
	b, err := fixturesFS.ReadFile("fixtures/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestJobContractValid(t *testing.T) {
	var job protocol.JobContract
	if err := json.Unmarshal(load(t, "job_valid.json"), &job); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if job.Tenant == "" || job.JobId.String() == "00000000-0000-0000-0000-000000000000" ||
		job.SourceId == "" || len(job.SourceKey) == 0 || job.SourceDigest == "" ||
		job.SourceGeneration == 0 || job.PipelineVersion == "" || job.TargetModel == "" ||
		job.TraceId == "" {
		t.Fatalf("required field came back zero-valued: %+v", job)
	}
	if job.Attempts != 1 || job.LeaseEpoch != 3 {
		t.Fatalf("unexpected attempts/lease_epoch: %+v", job)
	}
	if err := job.Validate(); err != nil {
		t.Fatalf("Validate() on a genuinely valid fixture: %v", err)
	}

	// Round-trip: marshal back and re-parse must yield an identical struct,
	// or the generated type silently drops/renames a field relative to the
	// schema that produced the fixture.
	b, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTripped protocol.JobContract
	if err := json.Unmarshal(b, &roundTripped); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if !reflect.DeepEqual(job, roundTripped) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", roundTripped, job)
	}
}

// TestJobContractMissingRequiredField documents a real gap: encoding/json
// does not enforce "required" on its own, so a missing trace_id silently
// decodes as "". That is exactly why JobContract.Validate() (gen/go/validate.go)
// exists — the next test proves it actually rejects this same fixture, so a
// caller isn't left trusting silent zero-value decoding as validation.
func TestJobContractMissingRequiredField(t *testing.T) {
	var job protocol.JobContract
	if err := json.Unmarshal(load(t, "job_missing_trace_id.json"), &job); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if job.TraceId != "" {
		t.Fatalf("expected trace_id to decode as zero-value, got %q", job.TraceId)
	}
}

// TestJobContractValidateRejectsMissingField is the real required-field
// enforcement the review asked for: the same fixture decodes "successfully"
// (previous test) but Validate() must still reject it.
func TestJobContractValidateRejectsMissingField(t *testing.T) {
	var job protocol.JobContract
	if err := json.Unmarshal(load(t, "job_missing_trace_id.json"), &job); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := job.Validate(); err == nil {
		t.Fatal("Validate() must reject a job missing trace_id, got nil error")
	}
}

func TestPublishRejectionStaleSource(t *testing.T) {
	var e protocol.ErrorResponse
	if err := json.Unmarshal(load(t, "publish_rejection_stale_source.json"), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if protocol.PublishRejectionCode(e.Code) != protocol.StaleSource {
		t.Fatalf("code = %q, want %q", e.Code, protocol.StaleSource)
	}
	if e.TraceId == "" {
		t.Fatal("trace_id must not be empty")
	}
}

func TestChunkIdentityValid(t *testing.T) {
	var chunk protocol.ChunkIdentity
	if err := json.Unmarshal(load(t, "chunk_identity_valid.json"), &chunk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if chunk.Status != protocol.Ready {
		t.Fatalf("status = %q, want %q", chunk.Status, protocol.Ready)
	}
}

func TestAnswerResponseValid(t *testing.T) {
	var ans protocol.AnswerResponse
	if err := json.Unmarshal(load(t, "answer_response_valid.json"), &ans); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ans.Answer == "" {
		t.Fatal("answer must not be empty")
	}
	if ans.AnswerMethod != protocol.Extractive {
		t.Fatalf("answer_method = %q, want %q", ans.AnswerMethod, protocol.Extractive)
	}
	if len(ans.Citations) != 1 || ans.Citations[0].ChunkId == "" {
		t.Fatalf("expected one well-formed citation, got %+v", ans.Citations)
	}
}

// TestDocumentValid and the three below it are ADR-0007/Workflow-2 schemas:
// Document, Permission and EvidenceResponse. Same round-trip pattern as the
// RFC-0003 fixtures above.
func TestDocumentValid(t *testing.T) {
	var doc protocol.Document
	if err := json.Unmarshal(load(t, "document_valid.json"), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("Validate() on a genuinely valid fixture: %v", err)
	}
	if doc.Status != protocol.Ready {
		t.Fatalf("status = %q, want %q", doc.Status, protocol.Ready)
	}
	if doc.ChunkCount == nil || *doc.ChunkCount != 5 {
		t.Fatalf("chunk_count = %v, want 5", doc.ChunkCount)
	}

	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTripped protocol.Document
	if err := json.Unmarshal(b, &roundTripped); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if !reflect.DeepEqual(doc, roundTripped) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", roundTripped, doc)
	}
}

func TestPermissionOwnerValid(t *testing.T) {
	var p protocol.Permission
	if err := json.Unmarshal(load(t, "permission_owner_valid.json"), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() on a genuinely valid fixture: %v", err)
	}
	if p.Role != protocol.Owner {
		t.Fatalf("role = %q, want %q", p.Role, protocol.Owner)
	}
}

func TestPermissionValidateRejectsBadRole(t *testing.T) {
	p := protocol.Permission{Principal: "mallory", Role: "superuser"}
	if err := p.Validate(); err == nil {
		t.Fatal("Validate() must reject an unrecognized role, got nil error")
	}
}

func TestEvidenceResponseValid(t *testing.T) {
	var e protocol.EvidenceResponse
	if err := json.Unmarshal(load(t, "evidence_valid.json"), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("Validate() on a genuinely valid fixture: %v", err)
	}
	if e.Text == "" {
		t.Fatal("text must not be empty")
	}
}
