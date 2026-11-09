package jobsjd

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
	p    Profile
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) CreateProfile(context.Context, uuid.UUID, string, string) (Profile, error) {
	return f.p, nil
}
func (f *fakeStore) ListProfiles(context.Context, uuid.UUID) ([]Profile, error) {
	return []Profile{f.p}, nil
}
func (f *fakeStore) UploadResume(context.Context, uuid.UUID, uuid.UUID, string, string, string) (Resume, error) {
	return Resume{}, nil
}
func (f *fakeStore) CreateJob(context.Context, uuid.UUID, string, string, string) (Job, error) {
	return Job{}, nil
}
func (f *fakeStore) ListJobs(context.Context, uuid.UUID, *uuid.UUID, int) (JobPage, error) {
	return JobPage{}, nil
}
func (f *fakeStore) Fit(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Fit, error) {
	return Fit{}, nil
}
func (f *fakeStore) ListFits(context.Context, uuid.UUID) ([]Fit, error) {
	return []Fit{{Score: ptr32(82)}}, nil
}

func ptr32(n int32) *int32 { return &n }

func TestListJobsRejectsBadCursor(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?cursor=nope", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.ListJobs(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestCreateProfileHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", p: Profile{ID: id, Name: "Ada"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewBufferString(`{"name":"Ada"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.CreateProfile(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestListFitsHTTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fits", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.ListFits(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"82"`)) && !bytes.Contains(rec.Body.Bytes(), []byte(`82`)) {
		t.Fatalf("%s", rec.Body.String())
	}
}
