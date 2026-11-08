package compose

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"outreach/internal/auth"
	"outreach/internal/evidence"
	"outreach/internal/sender"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Draft, error)
	Compose(ctx context.Context, workspaceID uuid.UUID, in ComposeInput) (ComposeResult, error)
	Patch(ctx context.Context, workspaceID, draftID uuid.UUID, in PatchInput) (Draft, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]Draft, error)
	Approve(ctx context.Context, workspaceID, userID, draftID uuid.UUID) (Draft, error)
	Reject(ctx context.Context, workspaceID, draftID uuid.UUID) (Draft, error)
	Regenerate(ctx context.Context, workspaceID, draftID uuid.UUID) (ComposeResult, error)
	Undo(ctx context.Context, workspaceID, draftID uuid.UUID) (Draft, error)
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		MailboxID string `json:"mailbox_id"`
		To        string `json:"to"`
		Subject   string `json:"subject"`
		Text      string `json:"text"`
		HTML      string `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	mailboxID, err := uuid.Parse(in.MailboxID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid mailbox id")
		return
	}
	d, err := a.store.Create(r.Context(), s.WorkspaceID, CreateInput{
		MailboxID: mailboxID, To: in.To, Subject: in.Subject, Text: in.Text, HTML: in.HTML,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (a *API) Compose(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		MailboxID string `json:"mailbox_id"`
		To        string `json:"to"`
		Subject   string `json:"subject"`
		Facts     []Fact `json:"facts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	mailboxID, err := uuid.Parse(in.MailboxID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid mailbox id")
		return
	}
	out, err := a.store.Compose(r.Context(), s.WorkspaceID, ComposeInput{
		MailboxID: mailboxID, To: in.To, Subject: in.Subject, Facts: in.Facts,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) Patch(w http.ResponseWriter, r *http.Request) {
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
	var in struct {
		Subject string `json:"subject"`
		Text    string `json:"text"`
		HTML    string `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	d, err := a.store.Patch(r.Context(), s.WorkspaceID, id, PatchInput{
		Subject: in.Subject, Text: in.Text, HTML: in.HTML,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
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
	rows, err := a.store.List(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if rows == nil {
		rows = []Draft{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"drafts": rows})
}

func (a *API) Approve(w http.ResponseWriter, r *http.Request) {
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
	d, err := a.store.Approve(r.Context(), s.WorkspaceID, s.UserID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (a *API) Reject(w http.ResponseWriter, r *http.Request) {
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
	d, err := a.store.Reject(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (a *API) Regenerate(w http.ResponseWriter, r *http.Request) {
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
	out, err := a.store.Regenerate(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) Undo(w http.ResponseWriter, r *http.Request) {
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
	d, err := a.store.Undo(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
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
	case errors.Is(err, ErrConflict), errors.Is(err, sender.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, evidence.ErrUnconfirmed):
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
