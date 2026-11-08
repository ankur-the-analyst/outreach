package suppression

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	Add(ctx context.Context, workspaceID uuid.UUID, kind, value, reason string) (Entry, error)
}

type API struct {
	store Store
	hmac  Hasher
}

func New(store Store, hmac Hasher) *API {
	return &API{store: store, hmac: hmac}
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	role, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID)
	if err != nil {
		writeError(w, http.StatusForbidden, "forbidden", "not a member")
		return
	}
	if role != "owner" && role != "member" {
		writeError(w, http.StatusForbidden, "forbidden", "owner or member role required")
		return
	}
	var in struct {
		Kind   string `json:"kind"`
		Value  string `json:"value"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	ent, err := a.store.Add(r.Context(), s.WorkspaceID, in.Kind, in.Value, in.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ent)
}

func (a *API) OneClick(w http.ResponseWriter, r *http.Request) {
	wsID, email, err := ParseToken(a.hmac, r.PathValue("token"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if _, err := a.store.Add(r.Context(), wsID, KindEmail, email, "opt-out"); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("You've been unsubscribed.\n"))
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
