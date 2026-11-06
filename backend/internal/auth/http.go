package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	cookieName   = "session"
	sessionTTL   = 7 * 24 * time.Hour
	minPassword  = 8
	roleOwner    = "owner"
	ctxSession   = ctxKey("session")
)

type ctxKey string

type Session struct {
	ID          string
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

type Store interface {
	CreateAccount(ctx context.Context, userID, workspaceID uuid.UUID, email, name, passwordHash, sessionID string, expires time.Time) error
	UserByEmail(ctx context.Context, email string) (userID uuid.UUID, passwordHash string, name string, err error)
	Memberships(ctx context.Context, userID uuid.UUID) (workspaceID uuid.UUID, err error)
	CreateSession(ctx context.Context, s Session, expires time.Time) error
	SessionByID(ctx context.Context, id string) (Session, time.Time, error)
	DeleteSession(ctx context.Context, id string) error
}

type API struct {
	store Store
	now   func() time.Time
}

func New(store Store) *API {
	return &API{store: store, now: time.Now}
}

func SessionFrom(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(ctxSession).(Session)
	return s, ok
}

func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, ctxSession, s)
}

func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" || len(in.Password) < minPassword {
		writeError(w, http.StatusBadRequest, "invalid_input", "email and password (min 8) required")
		return
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not hash password")
		return
	}
	userID, err := uuid.NewV7()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not mint id")
		return
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not mint id")
		return
	}
	sid, err := newSessionID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not mint session")
		return
	}
	name := strings.TrimSpace(in.Name)
	expires := a.now().Add(sessionTTL)
	err = a.store.CreateAccount(r.Context(), userID, wsID, email, name, hash, sid, expires)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email_taken", "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}
	setSessionCookie(w, sid, expires)
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":      map[string]any{"id": userID, "email": email, "name": name},
		"workspace": map[string]any{"id": wsID, "name": workspaceName(name, email)},
	})
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(in.Email))
	userID, hash, name, err := a.store.UserByEmail(r.Context(), email)
	if err != nil || !VerifyPassword(hash, in.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	wsID, err := a.store.Memberships(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not load workspace")
		return
	}
	sid, err := newSessionID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not mint session")
		return
	}
	expires := a.now().Add(sessionTTL)
	if err := a.store.CreateSession(r.Context(), Session{ID: sid, UserID: userID, WorkspaceID: wsID}, expires); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not create session")
		return
	}
	setSessionCookie(w, sid, expires)
	writeJSON(w, http.StatusOK, map[string]any{
		"user":      map[string]any{"id": userID, "email": email, "name": name},
		"workspace": map[string]any{"id": wsID},
	})
}

func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	s, ok := SessionFrom(r.Context())
	if ok {
		_ = a.store.DeleteSession(r.Context(), s.ID)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil || c.Value == "" {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
			return
		}
		sess, exp, err := a.store.SessionByID(r.Context(), c.Value)
		if err != nil || a.now().After(exp) {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxSession, sess)))
	})
}

func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	s, ok := SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":      s.UserID,
		"workspace_id": s.WorkspaceID,
	})
}

func workspaceName(name, email string) string {
	if name != "" {
		return name
	}
	local, _, _ := strings.Cut(email, "@")
	return local
}

func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func setSessionCookie(w http.ResponseWriter, id string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

