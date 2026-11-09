package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	Propose(ctx context.Context, workspaceID uuid.UUID, in ProposeInput) (Item, error)
	List(ctx context.Context, workspaceID uuid.UUID, after *uuid.UUID, limit int) (Page, error)
	Retrieve(ctx context.Context, workspaceID uuid.UUID, profileID *uuid.UUID, query string) ([]Item, error)
	Confirm(ctx context.Context, workspaceID, id uuid.UUID) (Item, error)
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStoreError(w, err)
		return
	}
	var after *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("cursor")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid cursor")
			return
		}
		after = &id
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := a.store.List(r.Context(), s.WorkspaceID, after, limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if page.Evidence == nil {
		page.Evidence = []Item{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) Usable(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStoreError(w, err)
		return
	}
	var profileID *uuid.UUID
	if raw := r.URL.Query().Get("profile_id"); raw != "" {
		p, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid profile id")
			return
		}
		profileID = &p
	}
	rows, err := a.store.Retrieve(r.Context(), s.WorkspaceID, profileID, r.URL.Query().Get("q"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rows == nil {
		rows = []Item{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"evidence": rows})
}

func (a *API) Propose(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		Text      string   `json:"text"`
		SourceRef string   `json:"source_ref"`
		Tags      []string `json:"tags"`
		ProfileID string   `json:"profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	var profileID *uuid.UUID
	if in.ProfileID != "" {
		p, err := uuid.Parse(in.ProfileID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid profile id")
			return
		}
		profileID = &p
	}
	item, err := a.store.Propose(r.Context(), s.WorkspaceID, ProposeInput{
		Text: in.Text, SourceRef: in.SourceRef, Tags: in.Tags, ProfileID: profileID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) Confirm(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	item, err := a.store.Confirm(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) session(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
	}
	return s, ok
}

func (a *API) requireEdit(ctx context.Context, s auth.Session) error {
	role, err := a.store.Membership(ctx, s.WorkspaceID, s.UserID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "member" {
		return ErrForbidden
	}
	return nil
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "owner or member role required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, ErrUnconfirmed):
		writeError(w, http.StatusBadRequest, "unconfirmed", err.Error())
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
