package evidence

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
	item Item
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, f.err
}
func (f *fakeStore) Propose(context.Context, uuid.UUID, ProposeInput) (Item, error) {
	return f.item, f.err
}
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Item, error) {
	return []Item{f.item}, f.err
}
func (f *fakeStore) Confirm(context.Context, uuid.UUID, uuid.UUID) (Item, error) {
	return f.item, f.err
}

func TestProposeHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", item: Item{ID: id, Text: "Shipped v1", Tags: []string{}}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence", bytes.NewBufferString(`{"text":"Shipped v1"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Propose(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
