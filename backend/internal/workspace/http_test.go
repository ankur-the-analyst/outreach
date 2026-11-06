package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type fakeStore struct {
	role        string
	ws          Workspace
	paused      bool
	deleted     bool
	members     []Member
	userID      uuid.UUID
	userErr     error
	addErr      error
	addedRole   string
	addedUserID uuid.UUID
}

func (f *fakeStore) Get(context.Context, uuid.UUID) (Workspace, error) { return f.ws, nil }
func (f *fakeStore) SetName(context.Context, uuid.UUID, string) error  { return nil }
func (f *fakeStore) SetPaused(_ context.Context, _ uuid.UUID, paused bool) error {
	f.paused = paused
	return nil
}
func (f *fakeStore) Delete(context.Context, uuid.UUID) error {
	f.deleted = true
	return nil
}
func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) ListMembers(context.Context, uuid.UUID) ([]Member, error) {
	return f.members, nil
}
func (f *fakeStore) UserByEmail(context.Context, string) (uuid.UUID, error) {
	if f.userErr != nil {
		return uuid.Nil, f.userErr
	}
	return f.userID, nil
}
func (f *fakeStore) AddMember(_ context.Context, _ uuid.UUID, userID uuid.UUID, role string) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.addedUserID = userID
	f.addedRole = role
	return nil
}

func sessionReq(method, path, body string, s auth.Session) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	}
	return r.WithContext(auth.WithSession(r.Context(), s))
}

func TestOwnerCanPause(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: RoleOwner}
	api := New(store)
	s := auth.Session{UserID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002")}
	req := sessionReq(http.MethodPut, "/api/v1/workspace/kill-switch", `{"paused":true}`, s)
	rec := httptest.NewRecorder()
	api.PutKillSwitch(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d body %s", rec.Code, rec.Body.Bytes())
	}
	if !store.paused {
		t.Fatal("expected pause")
	}
}

func TestMemberCannotPause(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: RoleMember}
	api := New(store)
	s := auth.Session{UserID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002")}
	req := sessionReq(http.MethodPut, "/api/v1/workspace/kill-switch", `{"paused":true}`, s)
	rec := httptest.NewRecorder()
	api.PutKillSwitch(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d", rec.Code)
	}
	if store.paused {
		t.Fatal("member must not pause")
	}
}

func TestMemberCannotDelete(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: RoleMember}
	api := New(store)
	s := auth.Session{UserID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002")}
	req := sessionReq(http.MethodDelete, "/api/v1/workspace", "", s)
	rec := httptest.NewRecorder()
	api.Delete(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d", rec.Code)
	}
	if store.deleted {
		t.Fatal("member must not delete")
	}
}

func TestOwnerCanDelete(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: RoleOwner}
	api := New(store)
	s := auth.Session{UserID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002")}
	req := sessionReq(http.MethodDelete, "/api/v1/workspace", "", s)
	rec := httptest.NewRecorder()
	api.Delete(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d", rec.Code)
	}
	if !store.deleted {
		t.Fatal("expected delete")
	}
}

func TestInviteDefaultsToMember(t *testing.T) {
	t.Parallel()
	invitee := uuid.MustParse("01700000-0000-7000-8000-000000000009")
	store := &fakeStore{role: RoleOwner, userID: invitee}
	api := New(store)
	s := auth.Session{UserID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002")}
	req := sessionReq(http.MethodPost, "/api/v1/workspace/members", `{"email":"m@x.co"}`, s)
	rec := httptest.NewRecorder()
	api.Invite(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d body %s", rec.Code, rec.Body.Bytes())
	}
	if store.addedRole != RoleMember || store.addedUserID != invitee {
		t.Fatalf("added role=%s user=%s", store.addedRole, store.addedUserID)
	}
	var got map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["role"] != RoleMember {
		t.Fatalf("%v", got)
	}
}
