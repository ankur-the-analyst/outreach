package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	session Session
	exp     time.Time
	getErr  error
}

func (f fakeStore) CreateAccount(context.Context, uuid.UUID, uuid.UUID, string, string, string, string, time.Time) error {
	return nil
}
func (f fakeStore) UserByEmail(context.Context, string) (uuid.UUID, string, string, error) {
	return uuid.Nil, "", "", nil
}
func (f fakeStore) Memberships(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (f fakeStore) CreateSession(context.Context, Session, time.Time) error { return nil }
func (f fakeStore) SessionByID(context.Context, string) (Session, time.Time, error) {
	return f.session, f.exp, f.getErr
}
func (f fakeStore) DeleteSession(context.Context, string) error { return nil }

func TestRegisterRejectsShortPassword(t *testing.T) {
	t.Parallel()
	api := New(fakeStore{})
	body := bytes.NewBufferString(`{"email":"a@b.co","password":"short"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	rec := httptest.NewRecorder()
	api.Register(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestRequireUnauthorizedWithoutCookie(t *testing.T) {
	t.Parallel()
	api := New(fakeStore{})
	inner := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not run")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	api.Require(inner).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestRequireAllowsValidSession(t *testing.T) {
	t.Parallel()
	sid := "abc"
	uid := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	ws := uuid.MustParse("01700000-0000-7000-8000-000000000002")
	api := New(fakeStore{
		session: Session{ID: sid, UserID: uid, WorkspaceID: ws},
		exp:     time.Now().Add(time.Hour),
	})
	var ran bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran = true
		s, ok := SessionFrom(r.Context())
		if !ok || s.UserID != uid {
			t.Fatalf("session %+v ok %v", s, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: sid})
	rec := httptest.NewRecorder()
	api.Require(inner).ServeHTTP(rec, req)
	if !ran || rec.Code != http.StatusNoContent {
		t.Fatalf("ran %v code %d", ran, rec.Code)
	}
}

func TestWriteErrorShape(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusUnauthorized, "unauthenticated", "sign in required")
	var got map[string]map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["error"]["code"] != "unauthenticated" {
		t.Fatalf("%v", got)
	}
}
