package compose

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
	role string
	d    Draft
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, f.err
}
func (f *fakeStore) Create(context.Context, uuid.UUID, CreateInput) (Draft, error) {
	return f.d, f.err
}
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Draft, error) {
	return []Draft{f.d}, f.err
}
func (f *fakeStore) Approve(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Draft, error) {
	return f.d, f.err
}
func (f *fakeStore) Undo(context.Context, uuid.UUID, uuid.UUID) (Draft, error) {
	return f.d, f.err
}

func TestCreateDraftHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", d: Draft{ID: id, Status: "pending"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts", bytes.NewBufferString(`{"mailbox_id":"01700000-0000-7000-8000-000000000004","to":"a@b.co","subject":"hi","text":"body"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
