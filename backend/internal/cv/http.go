package cv

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	Save(ctx context.Context, workspaceID, profileID uuid.UUID, spec Spec) (Document, error)
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) Save(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	var spec Spec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	doc, err := a.store.Save(r.Context(), s.WorkspaceID, id, spec)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (a *API) edit(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return auth.Session{}, false
	}
	role, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID)
	if err != nil {
		writeStoreError(w, err)
		return auth.Session{}, false
	}
	if role != "owner" && role != "member" {
		writeStoreError(w, ErrForbidden)
		return auth.Session{}, false
	}
	return s, true
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "owner or member role required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	default:
		if strings.Contains(err.Error(), "required") {
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
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
