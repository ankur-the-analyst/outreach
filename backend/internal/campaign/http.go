package campaign

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
	Packs() []Pack
	PutIdentity(ctx context.Context, workspaceID uuid.UUID, in Identity) (Identity, error)
	GetIdentity(ctx context.Context, workspaceID uuid.UUID) (Identity, error)
	Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Campaign, error)
	List(ctx context.Context, workspaceID uuid.UUID, after *uuid.UUID, limit int) (Page, error)
	Get(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error)
	Start(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error)
	Pause(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error)
	Stop(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error)
	Enrol(ctx context.Context, workspaceID, campaignID uuid.UUID, in EnrolInput) (Enrolment, error)
	ListEnrolments(ctx context.Context, workspaceID, campaignID uuid.UUID) ([]Enrolment, error)
	ListTracker(ctx context.Context, workspaceID uuid.UUID) ([]TrackerRow, error)
	SetStage(ctx context.Context, workspaceID, id uuid.UUID, stage string) (Application, error)
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) ListPacks(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.member(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packs": a.store.Packs()})
}

func (a *API) PutIdentity(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in Identity
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	out, err := a.store.PutIdentity(r.Context(), s.WorkspaceID, in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) GetIdentity(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	out, err := a.store.GetIdentity(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
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
	page, err := a.store.List(r.Context(), s.WorkspaceID, after, limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if page.Campaigns == nil {
		page.Campaigns = []Campaign{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in struct {
		Name              string   `json:"name"`
		Intent            string   `json:"intent"`
		SenderStrategy    string   `json:"sender_strategy"`
		MailboxIDs        []string `json:"mailbox_ids"`
		PoolID            string   `json:"pool_id"`
		NewContactsPerDay int32    `json:"new_contacts_per_day"`
		AIInstructions    string   `json:"ai_instructions"`
		ProfileID         string   `json:"profile_id"`
		TemplateID        string   `json:"template_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	ids := make([]uuid.UUID, 0, len(in.MailboxIDs))
	for _, raw := range in.MailboxIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid mailbox id")
			return
		}
		ids = append(ids, id)
	}
	var poolID *uuid.UUID
	if in.PoolID != "" {
		p, err := uuid.Parse(in.PoolID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid group id")
			return
		}
		poolID = &p
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
	var templateID *uuid.UUID
	if in.TemplateID != "" {
		t, err := uuid.Parse(in.TemplateID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid saved email")
			return
		}
		templateID = &t
	}
	c, err := a.store.Create(r.Context(), s.WorkspaceID, CreateInput{
		Name: in.Name, Intent: in.Intent, SenderStrategy: in.SenderStrategy,
		MailboxIDs: ids, PoolID: poolID, NewContactsPerDay: in.NewContactsPerDay, AIInstructions: in.AIInstructions,
		ProfileID: profileID, TemplateID: templateID,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *API) Get(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	c, err := a.store.Get(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) Start(w http.ResponseWriter, r *http.Request) {
	a.transition(w, r, a.store.Start)
}

func (a *API) Pause(w http.ResponseWriter, r *http.Request) {
	a.transition(w, r, a.store.Pause)
}

func (a *API) Stop(w http.ResponseWriter, r *http.Request) {
	a.transition(w, r, a.store.Stop)
}

func (a *API) transition(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID, uuid.UUID) (Campaign, error)) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	c, err := fn(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) Enrol(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in struct {
		ContactID string `json:"contact_id"`
		JobID     string `json:"job_id"`
		ProfileID string `json:"profile_id"`
		Notes     string `json:"notes"`
		Context   struct {
			Notes string `json:"notes"`
		} `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	contactID, err := uuid.Parse(in.ContactID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid contact id")
		return
	}
	notes := in.Notes
	if notes == "" {
		notes = in.Context.Notes
	}
	var jobID, profileID *uuid.UUID
	if in.JobID != "" {
		j, err := uuid.Parse(in.JobID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid job id")
			return
		}
		jobID = &j
	}
	if in.ProfileID != "" {
		p, err := uuid.Parse(in.ProfileID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid profile id")
			return
		}
		profileID = &p
	}
	en, err := a.store.Enrol(r.Context(), s.WorkspaceID, id, EnrolInput{
		ContactID: contactID, JobID: jobID, ProfileID: profileID, Notes: notes,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, en)
}

func (a *API) ListEnrolments(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	rows, err := a.store.ListEnrolments(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rows == nil {
		rows = []Enrolment{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"enrolments": rows})
}

func (a *API) ListTracker(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	rows, err := a.store.ListTracker(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rows == nil {
		rows = []TrackerRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

func (a *API) SetStage(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in struct {
		Stage string `json:"stage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	out, err := a.store.SetStage(r.Context(), s.WorkspaceID, id, strings.TrimSpace(in.Stage))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return uuid.UUID{}, false
	}
	return id, true
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
	case errors.Is(err, ErrIdentity), errors.Is(err, ErrContext), errors.Is(err, ErrPack), errors.Is(err, ErrName), errors.Is(err, ErrMailbox), errors.Is(err, ErrFrom), errors.Is(err, ErrStatus), errors.Is(err, ErrStage), errors.Is(err, ErrTemplate), errors.Is(err, ErrResume):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrDuplicate):
		writeError(w, http.StatusConflict, "conflict", err.Error())
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
