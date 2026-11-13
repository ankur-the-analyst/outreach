package compose

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/ai"
	"outreach/internal/auth"
)

type fakeStore struct {
	role string
	d    Draft
	err  error
	in   ComposeInput
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, f.err
}
func (f *fakeStore) Create(context.Context, uuid.UUID, CreateInput) (Draft, error) {
	return f.d, f.err
}
func (f *fakeStore) Patch(context.Context, uuid.UUID, uuid.UUID, PatchInput) (Draft, error) {
	return f.d, f.err
}
func (f *fakeStore) Compose(_ context.Context, _ uuid.UUID, in ComposeInput) (ComposeResult, error) {
	f.in = in
	return ComposeResult{Draft: f.d, Failures: []Failure{}}, f.err
}
func (f *fakeStore) List(context.Context, uuid.UUID, *uuid.UUID, int) (Page, error) {
	return Page{Drafts: []Draft{f.d}}, f.err
}
func (f *fakeStore) Approve(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Draft, error) {
	return f.d, f.err
}
func (f *fakeStore) Undo(context.Context, uuid.UUID, uuid.UUID) (Draft, error) {
	return f.d, f.err
}
func (f *fakeStore) Reject(context.Context, uuid.UUID, uuid.UUID) (Draft, error) {
	return f.d, f.err
}
func (f *fakeStore) Regenerate(context.Context, uuid.UUID, uuid.UUID) (ComposeResult, error) {
	return ComposeResult{Draft: f.d, Failures: []Failure{}}, f.err
}

func TestListDraftsRejectsBadCursor(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/drafts?cursor=nope", nil)
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

func TestPatchDraftHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", d: Draft{ID: id, Subject: "edited", BodyText: "new", Status: "pending"}})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/drafts/"+id.String(), bytes.NewBufferString(`{"subject":"edited","text":"new"}`))
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Patch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestRejectDraftHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", d: Draft{ID: id, Status: "rejected"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts/"+id.String()+"/reject", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Reject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestComposeHTTPProfileID(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	profile := uuid.MustParse("01700000-0000-7000-8000-00000000000a")
	store := &fakeStore{role: "owner", d: Draft{ID: id, Status: "pending"}}
	api := New(store)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts/compose", bytes.NewBufferString(
		`{"mailbox_id":"01700000-0000-7000-8000-000000000004","to":"a@b.co","subject":"hi","profile_id":"`+profile.String()+`"}`,
	))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Compose(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if store.in.ProfileID == nil || *store.in.ProfileID != profile {
		t.Fatalf("profile_id = %v", store.in.ProfileID)
	}
}

func TestComposeHTTPNeedsSettings(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", err: ai.ErrNoKey})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts/compose", bytes.NewBufferString(
		`{"mailbox_id":"01700000-0000-7000-8000-000000000004","to":"a@b.co","subject":"hi"}`,
	))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Compose(rec, req)
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("ai_not_configured")) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestApproveLeftoverHTTP(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", err: ErrLeftover})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts/"+id.String()+"/approve", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Approve(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "leftover") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestComposeHTTPNoClaims(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", err: ErrNoClaims})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts/compose", bytes.NewBufferString(
		`{"mailbox_id":"01700000-0000-7000-8000-000000000004","to":"a@b.co","subject":"hi"}`,
	))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Compose(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
