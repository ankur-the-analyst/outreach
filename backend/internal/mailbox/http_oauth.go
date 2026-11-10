package mailbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"outreach/internal/auth"
)

const (
	oauthStateCookie = "oauth_state"
	oauthStateTTL    = 10 * time.Minute
)

var googleScopes = []string{
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/userinfo.email",
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	StateKey     []byte
	Now          func() time.Time
	HTTP         *http.Client
	SuccessPath  string
}

type tokenExchanger interface {
	Exchange(ctx context.Context, code string) (refresh, email string, err error)
}

type googleExchange struct {
	cfg  *oauth2.Config
	http *http.Client
}

func (g googleExchange) Exchange(ctx context.Context, code string) (string, string, error) {
	tok, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return "", "", fmt.Errorf("google token: %w", err)
	}
	if tok.RefreshToken == "" {
		return "", "", fmt.Errorf("google did not return a refresh token; remove access for this app in Google account and try again")
	}
	email, err := fetchGoogleEmail(ctx, g.http, tok.AccessToken)
	if err != nil {
		return "", "", err
	}
	return tok.RefreshToken, email, nil
}

func fetchGoogleEmail(ctx context.Context, client *http.Client, access string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("google userinfo %s", res.Status)
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return "", err
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" {
		return "", fmt.Errorf("google userinfo: empty email")
	}
	return email, nil
}

type SyncQueue interface {
	EnqueueSync(ctx context.Context, workspaceID, mailboxID uuid.UUID) error
}

func (a *API) WithOAuth(cfg OAuthConfig) *API {
	a.oauth = cfg
	if a.oauth.Now == nil {
		a.oauth.Now = time.Now
	}
	if a.oauth.SuccessPath == "" {
		a.oauth.SuccessPath = "/mailboxes"
	}
	return a
}

func (a *API) WithDialer(d interface {
	Dial(ctx context.Context, workspaceID, mailboxID uuid.UUID) (Transport, string, error)
}) *API {
	a.dial = d
	return a
}

func (a *API) WithSync(q SyncQueue) *API {
	a.syncq = q
	return a
}

func (a *API) googleOAuth() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.oauth.ClientID,
		ClientSecret: a.oauth.ClientSecret,
		RedirectURL:  a.oauth.RedirectURI,
		Scopes:       googleScopes,
		Endpoint:     google.Endpoint,
	}
}

func (a *API) OAuthEnabled(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"google": a.oauth.ClientID != ""})
}

func (a *API) OAuthStart(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	if a.oauth.ClientID == "" || len(a.oauth.StateKey) == 0 {
		writeError(w, http.StatusBadRequest, "not_configured", "Google sign-in is not configured")
		return
	}
	tz := strings.TrimSpace(r.URL.Query().Get("timezone"))
	if tz == "" {
		tz = "UTC"
	}
	now := a.oauth.Now()
	tok, err := signOAuthState(a.oauth.StateKey, oauthState{
		WorkspaceID: s.WorkspaceID,
		UserID:      s.UserID,
		Timezone:    tz,
		Exp:         now.Add(oauthStateTTL).Unix(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not start Google sign-in")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    tok,
		Path:     "/",
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	u := a.googleOAuth().AuthCodeURL(tok, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	http.Redirect(w, r, u, http.StatusFound)
}

func (a *API) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Google hits this URL on the API host. Auth is the HMAC state from
	// /start (session cookie stays on the app origin).
	if a.oauth.ClientID == "" {
		a.redirectMailboxes(w, r, "google_not_configured")
		return
	}
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		a.redirectMailboxes(w, r, "google_denied")
		return
	}
	state := r.URL.Query().Get("state")
	if c, err := r.Cookie(oauthStateCookie); err == nil && c.Value != "" && c.Value != state {
		a.redirectMailboxes(w, r, "google_state")
		return
	}
	st, err := verifyOAuthState(a.oauth.StateKey, state, a.oauth.Now())
	if err != nil {
		a.redirectMailboxes(w, r, "google_state")
		return
	}
	s := auth.Session{UserID: st.UserID, WorkspaceID: st.WorkspaceID}
	if err := a.requireEdit(r.Context(), s); err != nil {
		a.redirectMailboxes(w, r, "google_state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		a.redirectMailboxes(w, r, "google_code")
		return
	}
	ex := a.exchanger
	if ex == nil {
		ex = googleExchange{cfg: a.googleOAuth(), http: a.oauth.HTTP}
	}
	refresh, email, err := ex.Exchange(r.Context(), code)
	if err != nil {
		a.redirectMailboxes(w, r, "google_token")
		return
	}
	tz := st.Timezone
	if tz == "" {
		tz = "UTC"
	}
	local, _, _ := strings.Cut(email, "@")
	mb, err := a.store.Connect(r.Context(), s.WorkspaceID, ConnectInput{
		Email: email, DisplayName: local, AuthKind: "oauth", Provider: "gmail",
		Password: refresh, Timezone: tz,
	})
	if err != nil {
		a.redirectMailboxes(w, r, url.QueryEscape(err.Error()))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: oauthStateCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	if a.syncq != nil {
		_ = a.syncq.EnqueueSync(r.Context(), s.WorkspaceID, mb.ID)
	}
	a.redirectMailboxes(w, r, "")
}

func (a *API) redirectMailboxes(w http.ResponseWriter, r *http.Request, errCode string) {
	path := a.oauth.SuccessPath
	if path == "" {
		path = "/mailboxes"
	}
	if errCode != "" {
		path += "?error=" + errCode
	} else {
		path += "?connected=1"
	}
	http.Redirect(w, r, path, http.StatusFound)
}
