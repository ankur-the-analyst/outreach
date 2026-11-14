package fields

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
	List(ctx context.Context, workspaceID uuid.UUID, object string) ([]Def, error)
	Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Def, error)
	Mine(ctx context.Context, workspaceID, userID uuid.UUID) (map[string]string, error)
	SetMine(ctx context.Context, workspaceID, userID uuid.UUID, values map[string]string) error
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	object := strings.TrimSpace(r.URL.Query().Get("object"))
	if object != "" && !ValidObject(object) {
		writeError(w, http.StatusBadRequest, "invalid_input", "unknown object")
		return
	}
	defs, err := a.store.List(r.Context(), s.WorkspaceID, object)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if defs == nil {
		defs = []Def{}
	}
	if r.URL.Query().Get("catalog") == "1" && object != "" {
		defs = append(Catalog(object), defs...)
	}
	writeJSON(w, http.StatusOK, map[string]any{"fields": defs})
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in struct {
		Object   string   `json:"object"`
		Key      string   `json:"key"`
		Label    string   `json:"label"`
		Kind     string   `json:"kind"`
		Required bool     `json:"required"`
		Options  []string `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	d, err := a.store.Create(r.Context(), s.WorkspaceID, CreateInput{
		Object: in.Object, Key: in.Key, Label: in.Label, Kind: in.Kind, Required: in.Required, Options: in.Options,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (a *API) Mine(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	values, err := a.store.Mine(r.Context(), s.WorkspaceID, s.UserID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"values": values})
}

func (a *API) SetMine(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in struct {
		Values map[string]string `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	if err := a.store.SetMine(r.Context(), s.WorkspaceID, s.UserID, in.Values); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) member(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return auth.Session{}, false
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStoreError(w, err)
		return auth.Session{}, false
	}
	return s, true
}

func (a *API) edit(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	s, ok := a.member(w, r)
	if !ok {
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
	case errors.Is(err, ErrDuplicate):
		writeError(w, http.StatusConflict, "conflict", "That field already exists")
	case errors.Is(err, ErrBuiltin):
		writeError(w, http.StatusBadRequest, "invalid_input", "That name is already a built-in field")
	case errors.Is(err, ErrKey):
		writeError(w, http.StatusBadRequest, "invalid_input", "Use lowercase letters, numbers, and underscores")
	case errors.Is(err, ErrOptions):
		writeError(w, http.StatusBadRequest, "invalid_input", "Add at least one choice")
	case errors.Is(err, ErrObject), errors.Is(err, ErrKind):
		writeError(w, http.StatusBadRequest, "invalid_input", "That field type is not supported")
	case errors.Is(err, ErrLabel):
		writeError(w, http.StatusBadRequest, "invalid_input", "Name this field")
	case errors.Is(err, ErrLimit):
		writeError(w, http.StatusBadRequest, "invalid_input", "Too many extra fields")
	default:
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
