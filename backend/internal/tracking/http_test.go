package tracking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/crypto"
)

type fakeStore struct {
	kind string
	n    int
}

func (f *fakeStore) RecordFromDraft(context.Context, uuid.UUID, uuid.UUID, string, string) error {
	f.n++
	return nil
}

func TestOpenAndClickHTTP(t *testing.T) {
	t.Parallel()
	k, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ws := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	draft := uuid.MustParse("01700000-0000-7000-8000-000000000002")
	st := &fakeStore{}
	api := New(st, k)
	openTok, err := Token(k, TokenData{WorkspaceID: ws, DraftID: draft, Kind: KindOpen})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/t/o/"+openTok, nil)
	req.SetPathValue("token", openTok)
	rec := httptest.NewRecorder()
	api.Open(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/gif" || st.n != 1 {
		t.Fatalf("open %d %s n=%d", rec.Code, rec.Body.String(), st.n)
	}
	bad := httptest.NewRecorder()
	breq := httptest.NewRequest(http.MethodGet, "/t/o/nope", nil)
	breq.SetPathValue("token", "nope")
	api.Open(bad, breq)
	if bad.Code != http.StatusNoContent || st.n != 1 {
		t.Fatalf("invalid open %d n=%d", bad.Code, st.n)
	}
	clickTok, err := Token(k, TokenData{WorkspaceID: ws, DraftID: draft, Kind: KindClick, URL: "https://jobs.example/ada"})
	if err != nil {
		t.Fatal(err)
	}
	creq := httptest.NewRequest(http.MethodGet, "/t/c/"+clickTok, nil)
	creq.SetPathValue("token", clickTok)
	crec := httptest.NewRecorder()
	api.Click(crec, creq)
	if crec.Code != http.StatusFound || crec.Header().Get("Location") != "https://jobs.example/ada" || st.n != 2 {
		t.Fatalf("click %d %s n=%d", crec.Code, crec.Header().Get("Location"), st.n)
	}
	nf := httptest.NewRecorder()
	nreq := httptest.NewRequest(http.MethodGet, "/t/c/nope", nil)
	nreq.SetPathValue("token", "nope")
	api.Click(nf, nreq)
	if nf.Code != http.StatusNotFound {
		t.Fatalf("invalid click %d", nf.Code)
	}
}
