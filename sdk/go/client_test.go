package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	protocol "github.com/Pg-Helix/helix-protocol/gen/go"
)

func TestCreateCollection_SendsIdempotencyKey(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(protocol.Collection{Id: "col-1", Tenant: "acme", Name: "docs"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	col, err := c.CreateCollection(context.Background(), "acme", "docs", "idem-key-123")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if gotKey != "idem-key-123" {
		t.Fatalf("Idempotency-Key header = %q, want %q", gotKey, "idem-key-123")
	}
	if col.Id != "col-1" {
		t.Fatalf("Id = %q, want col-1", col.Id)
	}
}

func TestListCollections_Pagination(t *testing.T) {
	var gotCursors []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		gotCursors = append(gotCursors, cursor)
		if cursor == "" {
			_ = json.NewEncoder(w).Encode(CollectionPage{
				Collections: []protocol.Collection{{Id: "col-1"}},
				NextCursor:  "page-2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(CollectionPage{
			Collections: []protocol.Collection{{Id: "col-2"}},
			NextCursor:  "",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	page1, err := c.ListCollections(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if page1.NextCursor != "page-2" {
		t.Fatalf("page1.NextCursor = %q, want page-2", page1.NextCursor)
	}
	page2, err := c.ListCollections(context.Background(), page1.NextCursor, 10)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if page2.NextCursor != "" {
		t.Fatalf("page2.NextCursor = %q, want empty (no further pages)", page2.NextCursor)
	}
	if len(gotCursors) != 2 || gotCursors[0] != "" || gotCursors[1] != "page-2" {
		t.Fatalf("server saw cursors %v, want [\"\", \"page-2\"]", gotCursors)
	}
}

func TestSearch_TypedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(protocol.ErrorResponse{
			Code: "tenant_mismatch", Message: "tenant does not own this collection", TraceId: "trace-x",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.Search(context.Background(), protocol.SearchJSONBody{Tenant: "acme", CollectionId: "col-1", Query: "refunds"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is not *APIError: %v", err)
	}
	if apiErr.Code != "tenant_mismatch" {
		t.Fatalf("Code = %q, want tenant_mismatch", apiErr.Code)
	}
	if apiErr.TraceID != "trace-x" {
		t.Fatalf("TraceID = %q, want trace-x", apiErr.TraceID)
	}
}

func TestGetAnswer_ContextDeadlineExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	c := NewClient(srv.URL)
	_, err := c.GetAnswer(ctx, protocol.GetAnswerJSONBody{Tenant: "acme", CollectionId: "col-1", Query: "refunds"})
	if err == nil {
		t.Fatal("expected a deadline error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error does not wrap context.DeadlineExceeded: %v", err)
	}
}
