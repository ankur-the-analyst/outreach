package health

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type fakeStore struct {
	samples []Sample
	err     error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "owner", f.err
}
func (f *fakeStore) Snapshot(context.Context, uuid.UUID) error { return f.err }
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Sample, error) {
	return f.samples, f.err
}

func TestListSamplesHTTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{samples: []Sample{{Metric: "unknown", Value: 2, Window: "live"}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health-samples", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.List(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"unknown"`)) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}
