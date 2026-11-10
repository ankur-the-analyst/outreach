package mailbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type fakeEx struct {
	refresh, email string
	err            error
}

func (f fakeEx) Exchange(context.Context, string) (string, string, error) {
	return f.refresh, f.email, f.err
}

func oauthAPI(store Store, ex tokenExchanger) *API {
	api := New(store, fakeTester{})
	api.WithOAuth(OAuthConfig{
		ClientID:     "cid",
		ClientSecret: "sec",
		RedirectURI:  "http://localhost:3000/api/v1/mailboxes/oauth/callback",
		StateKey:     []byte("state-key"),
		Now:          func() time.Time { return time.Unix(1_700_000_000, 0) },
		SuccessPath:  "/mailboxes",
	})
	api.exchanger = ex
	return api
}

func TestOAuthStartRedirects(t *testing.T) {
	t.Parallel()
	api := oauthAPI(&fakeStore{role: "owner"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mailboxes/oauth/google/start", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.OAuthStart(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "accounts.google.com") || !strings.Contains(loc, "gmail.send") {
		t.Fatalf("location %s", loc)
	}
	if rec.Result().Cookies()[0].Name != oauthStateCookie {
		t.Fatalf("cookies %+v", rec.Result().Cookies())
	}
}

func TestOAuthCallbackConnects(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: "owner"}
	api := oauthAPI(store, fakeEx{refresh: "rt", email: "ada@gmail.com"})
	st, err := signOAuthState([]byte("state-key"), oauthState{
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		Timezone:    "UTC",
		Exp:         time.Unix(1_700_000_000, 0).Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mailboxes/oauth/callback?code=abc&state="+st, nil)
	rec := httptest.NewRecorder()
	api.OAuthCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "connected=1") {
		t.Fatalf("got %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if store.connect.AuthKind != "oauth" || store.connect.Email != "ada@gmail.com" || store.connect.Provider != "gmail" || store.connect.Password != "rt" {
		t.Fatalf("%+v", store.connect)
	}
}

func TestOAuthCallbackRejectsBadState(t *testing.T) {
	t.Parallel()
	api := oauthAPI(&fakeStore{role: "owner"}, fakeEx{refresh: "rt", email: "ada@gmail.com"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mailboxes/oauth/callback?code=abc&state=nope", nil)
	rec := httptest.NewRecorder()
	api.OAuthCallback(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "google_state") {
		t.Fatalf("got %d %s", rec.Code, rec.Header().Get("Location"))
	}
}
