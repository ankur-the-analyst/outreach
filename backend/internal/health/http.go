package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	Snapshot(ctx context.Context, workspaceID uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID) ([]Sample, error)
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeSampleError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStore(w, err)
		return
	}
	if err := a.store.Snapshot(r.Context(), s.WorkspaceID); err != nil {
		writeStore(w, err)
		return
	}
	items, err := a.store.List(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStore(w, err)
		return
	}
	if items == nil {
		items = []Sample{}
	}
	writeSampleJSON(w, http.StatusOK, map[string]any{"samples": items})
}

func writeStore(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeSampleError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	writeSampleError(w, http.StatusInternalServerError, "internal", "internal error")
}

func writeSampleJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSampleError(w http.ResponseWriter, status int, code, message string) {
	writeSampleJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}
