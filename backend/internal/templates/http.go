package templates

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
	List(ctx context.Context, workspaceID uuid.UUID) ([]Template, error)
	Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Template, error)
	Update(ctx context.Context, workspaceID, id uuid.UUID, in CreateInput) (Template, error)
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
	rows, err := a.store.List(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rows == nil {
		rows = []Template{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": rows})
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	in, ok := decodeSave(w, r)
	if !ok {
		return
	}
	t, err := a.store.Create(r.Context(), s.WorkspaceID, in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (a *API) Put(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	in, ok := decodeSave(w, r)
	if !ok {
		return
	}
	t, err := a.store.Update(r.Context(), s.WorkspaceID, id, in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func decodeSave(w http.ResponseWriter, r *http.Request) (CreateInput, bool) {
	var in struct {
		Name     string          `json:"name"`
		Subject  string          `json:"subject"`
		BodyText string          `json:"body_text"`
		BodyHTML string          `json:"body_html"`
		Design   json.RawMessage `json:"design_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return CreateInput{}, false
	}
	return CreateInput{
		Name: in.Name, Subject: in.Subject, BodyText: in.BodyText, BodyHTML: in.BodyHTML, Design: in.Design,
	}, true
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
	case errors.Is(err, ErrUnknown):
		writeError(w, http.StatusBadRequest, "invalid_input", ErrUnknown.Error())
	case errors.Is(err, ErrUnclosed), errors.Is(err, ErrName), errors.Is(err, ErrSubject), errors.Is(err, ErrBody), errors.Is(err, ErrDesign):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
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
