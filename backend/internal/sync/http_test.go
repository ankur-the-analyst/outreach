package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type fakeStore struct {
	role    string
	threads []Thread
	markErr error
	readID  uuid.UUID
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) SyncMailbox(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 0, nil
}
func (f *fakeStore) ListThreads(context.Context, uuid.UUID, uuid.UUID) ([]Thread, error) {
	return f.threads, nil
}
func (f *fakeStore) MarkRead(_ context.Context, _ uuid.UUID, threadID uuid.UUID) error {
	f.readID = threadID
	return f.markErr
}

func TestMarkReadHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000009")
	store := &fakeStore{role: "owner"}
	api := New(store)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/threads/"+id.String()+"/read", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.MarkRead(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if store.readID != id {
		t.Fatalf("read id %s", store.readID)
	}
}
