package suppression

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/auth"
	"outreach/internal/crypto"
)

type fakeStore struct {
	role string
	ent  Entry
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, f.err
}
func (f *fakeStore) Add(context.Context, uuid.UUID, string, string, string) (Entry, error) {
	return f.ent, f.err
}
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Entry, error) {
	return []Entry{f.ent}, f.err
}

func TestCreateSuppression(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", ent: Entry{Kind: KindEmail, Value: "a@b.co", Reason: "opt-out"}}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suppression", bytes.NewBufferString(`{"kind":"email","value":"a@b.co","reason":"opt-out"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), UserID: uuid.MustParse("01700000-0000-7000-8000-000000000002")}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestListSuppression(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", ent: Entry{Kind: KindEmail, Value: "a@b.co", Reason: "opt-out"}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/suppression", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"a@b.co"`)) {
		t.Fatalf("%s", rec.Body.String())
	}
}

func TestOneClick(t *testing.T) {
	t.Parallel()
	k, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ws := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	tok, err := Token(k, ws, "ada@acme.com")
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{ent: Entry{Kind: KindEmail, Value: "ada@acme.com", Reason: "opt-out"}}
	api := New(store, k)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unsubscribe/"+tok, bytes.NewBufferString("List-Unsubscribe=One-Click"))
	req.SetPathValue("token", tok)
	rec := httptest.NewRecorder()
	api.OneClick(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	bad := httptest.NewRequest(http.MethodPost, "/api/v1/unsubscribe/nope", nil)
	bad.SetPathValue("token", "nope")
	rec = httptest.NewRecorder()
	api.OneClick(rec, bad)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
