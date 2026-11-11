package jobsjd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"outreach/internal/ai"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	CreateProfile(ctx context.Context, workspaceID uuid.UUID, name, about string) (Profile, error)
	ListProfiles(ctx context.Context, workspaceID uuid.UUID) ([]Profile, error)
	UploadResume(ctx context.Context, workspaceID, profileID uuid.UUID, fileName, mime, text string) (Resume, error)
	CreateJob(ctx context.Context, workspaceID uuid.UUID, title, sourceURL, raw string) (Job, error)
	ListJobs(ctx context.Context, workspaceID uuid.UUID, after *uuid.UUID, limit int) (JobPage, error)
	Fit(ctx context.Context, workspaceID, jobID, profileID uuid.UUID) (Fit, error)
	ListFits(ctx context.Context, workspaceID uuid.UUID) ([]Fit, error)
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) CreateProfile(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in struct {
		Name  string `json:"name"`
		About string `json:"about"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	p, err := a.store.CreateProfile(r.Context(), s.WorkspaceID, in.Name, in.About)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *API) ListProfiles(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	rows, err := a.store.ListProfiles(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rows == nil {
		rows = []Profile{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": rows})
}

func (a *API) UploadResume(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	var in struct {
		FileName string `json:"file_name"`
		MIME     string `json:"mime"`
		Text     string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	res, err := a.store.UploadResume(r.Context(), s.WorkspaceID, id, in.FileName, in.MIME, in.Text)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (a *API) CreateJob(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in struct {
		Title     string `json:"title"`
		SourceURL string `json:"source_url"`
		Text      string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	job, err := a.store.CreateJob(r.Context(), s.WorkspaceID, in.Title, in.SourceURL, in.Text)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (a *API) ListJobs(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
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
	page, err := a.store.ListJobs(r.Context(), s.WorkspaceID, after, limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if page.Jobs == nil {
		page.Jobs = []Job{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) Fit(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	var in struct {
		ProfileID string `json:"profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	profileID, err := uuid.Parse(in.ProfileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid profile id")
		return
	}
	fit, err := a.store.Fit(r.Context(), s.WorkspaceID, jobID, profileID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, fit)
}

func (a *API) ListFits(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	rows, err := a.store.ListFits(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rows == nil {
		rows = []Fit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"fits": rows})
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
	case errors.Is(err, ai.ErrNoBinding), errors.Is(err, ai.ErrNoKey), errors.Is(err, ai.ErrNotFound):
		writeError(w, http.StatusBadRequest, "ai_not_configured", "Set a model and API key in Settings")
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
