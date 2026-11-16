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
func (f *fakeStore) Import(context.Context, uuid.UUID, uuid.UUID, string, string, ColumnMap) (ImportResult, error) {
	if f.err != nil {
		return ImportResult{}, f.err
	}
	return ImportResult{Created: 1}, nil
}
func (f *fakeStore) List(context.Context, uuid.UUID, *uuid.UUID, int) (Page, error) {
	return Page{Contacts: []Contact{f.c}}, nil
}
func (f *fakeStore) Get(context.Context, uuid.UUID, uuid.UUID) (Contact, error) {
	return f.c, nil
}
func (f *fakeStore) Patch(context.Context, uuid.UUID, uuid.UUID, PatchInput) (Contact, error) {
	return f.c, nil
}
func (f *fakeStore) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeStore) Erase(context.Context, uuid.UUID, uuid.UUID) error  { return f.err }
func (f *fakeStore) Dedup(context.Context, uuid.UUID) (Dedup, error) {
	return Dedup{Action: "prevent"}, f.err
}
func (f *fakeStore) SetDedup(context.Context, uuid.UUID, Dedup) (Dedup, error) {
	return Dedup{Action: "prevent"}, f.err
}

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

func TestListRejectsBadCursor(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?cursor=nope", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.List(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestEraseHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+id.String()+"/erase", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Erase(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
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

func TestImportHTTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/import", bytes.NewBufferString(`{"csv":"email\nada@acme.com","provenance":{"source":"csv"}}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Import(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
