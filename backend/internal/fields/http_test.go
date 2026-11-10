package fields

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
	defs []Def
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) List(context.Context, uuid.UUID, string) ([]Def, error) {
	return f.defs, f.err
}
func (f *fakeStore) Create(context.Context, uuid.UUID, CreateInput) (Def, error) {
	if f.err != nil {
		return Def{}, f.err
	}
	return Def{Key: "seniority", Label: "Seniority", Kind: "text", Object: "contact"}, nil
}

func TestCreateRejectsBuiltin(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", err: ErrBuiltin})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fields", bytes.NewBufferString(`{"object":"contact","key":"email","label":"Email"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("built-in")) {
		t.Fatalf("%s", rec.Body.String())
	}
}

func TestListEmpty(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fields?object=contact", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"fields":[]`)) {
		t.Fatalf("%s", rec.Body.String())
	}
}
