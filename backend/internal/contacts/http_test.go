package contacts

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
	c    Contact
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) Create(_ context.Context, _ uuid.UUID, in CreateInput) (Contact, error) {
	if in.Source == "" {
		return Contact{}, ErrProvenance
	}
	if f.err != nil {
		return Contact{}, f.err
	}
	return f.c, nil
}
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Contact, error) {
	return []Contact{f.c}, nil
}
func (f *fakeStore) Get(context.Context, uuid.UUID, uuid.UUID) (Contact, error) {
	return f.c, nil
}
func (f *fakeStore) Patch(context.Context, uuid.UUID, uuid.UUID, PatchInput) (Contact, error) {
	return f.c, nil
}
func (f *fakeStore) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func TestCreateRejectsEmptyProvenance(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewBufferString(`{"email":"ada@acme.com"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("provenance required")) {
		t.Fatalf("%s", rec.Body.String())
	}
}

func TestPreviewHTTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/preview", bytes.NewBufferString(`{"csv":"email\nada@acme.com"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Preview(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
