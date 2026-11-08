package contacts

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
	Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Contact, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]Contact, error)
	Get(ctx context.Context, workspaceID, id uuid.UUID) (Contact, error)
	Patch(ctx context.Context, workspaceID, id uuid.UUID, in PatchInput) (Contact, error)
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error
	Erase(ctx context.Context, workspaceID, id uuid.UUID) error
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
		rows = []Contact{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"contacts": rows})
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in struct {
		Email       string `json:"email"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Title       string `json:"title"`
		Company     string `json:"company"`
		LinkedInURL string `json:"linkedin_url"`
		Timezone    string `json:"timezone"`
		Region      string `json:"region"`
		Provenance  struct {
			Source string `json:"source"`
		} `json:"provenance"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	source := in.Provenance.Source
	if source == "" {
		source = in.Source
	}
	c, err := a.store.Create(r.Context(), s.WorkspaceID, CreateInput{
		Email: in.Email, FirstName: in.FirstName, LastName: in.LastName, Title: in.Title,
		Company: in.Company, LinkedInURL: in.LinkedInURL, Timezone: in.Timezone, Region: in.Region,
		Source: source, AddedBy: s.UserID,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *API) Preview(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.member(w, r); !ok {
		return
	}
	var in struct {
		CSV  string `json:"csv"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	text := in.CSV
	if text == "" {
		text = in.Text
	}
	rows, err := PreviewCSV(text)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

func (a *API) Get(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	c, err := a.store.Get(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) Patch(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	var in PatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	c, err := a.store.Patch(r.Context(), s.WorkspaceID, id, in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) Delete(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	if err := a.store.Delete(r.Context(), s.WorkspaceID, id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) Erase(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	if err := a.store.Erase(r.Context(), s.WorkspaceID, id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	case errors.Is(err, ErrProvenance), errors.Is(err, ErrEmail), errors.Is(err, ErrCSV):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrDuplicate):
		writeError(w, http.StatusConflict, "conflict", err.Error())
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
