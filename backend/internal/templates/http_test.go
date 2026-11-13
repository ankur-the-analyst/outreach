package templates

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
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Template, error) {
	return nil, nil
}
func (f *fakeStore) Create(_ context.Context, _ uuid.UUID, in CreateInput) (Template, error) {
	if err := Validate(in.Subject); err != nil {
		return Template{}, err
	}
	if err := Validate(in.BodyText); err != nil {
		return Template{}, err
	}
	if err := ValidateJSON(in.Design); err != nil {
		return Template{}, err
	}
	if f.err != nil {
		return Template{}, f.err
	}
	return Template{Name: in.Name, Subject: in.Subject, BodyText: in.BodyText, Design: in.Design}, nil
}

func (f *fakeStore) Update(_ context.Context, _ uuid.UUID, _ uuid.UUID, in CreateInput) (Template, error) {
	return f.Create(context.Background(), uuid.Nil, in)
}

func TestCreateRejectsUnknownVar(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewBufferString(
		`{"name":"intro","subject":"Hi","body_text":"Hello {{nope}}"}`,
	))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("placeholder")) {
		t.Fatalf("%s", rec.Body.String())
	}
}

func TestCreateRejectsUnknownVarInDesign(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewBufferString(
		`{"name":"intro","subject":"Hi","body_text":"Hello","design_json":{"elements":[{"type":"button","href":"https://x.test?n={{contact.nope}}"}]}}`,
	))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("placeholder")) {
		t.Fatalf("%s", rec.Body.String())
	}
}
