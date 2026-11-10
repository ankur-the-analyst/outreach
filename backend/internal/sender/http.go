package sender

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"outreach/internal/auth"
	"outreach/internal/mime"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	Hold(ctx context.Context, workspaceID uuid.UUID, in HoldInput) (Entry, error)
	Cancel(ctx context.Context, workspaceID, id uuid.UUID) (Entry, error)
	Usage(ctx context.Context, workspaceID, mailboxID uuid.UUID) (int64, error)
	Reserve(ctx context.Context, workspaceID uuid.UUID, in ReserveInput) (Entry, error)
	Send(ctx context.Context, workspaceID, ledgerID uuid.UUID, in SendInput) (Entry, error)
	Reconcile(ctx context.Context, workspaceID, ledgerID uuid.UUID, in SendInput) (Entry, error)
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) Hold(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	mailboxID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	var in struct {
		RFCMessageID string `json:"rfc_message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	msgid := strings.TrimSpace(in.RFCMessageID)
	if msgid == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "rfc_message_id required")
		return
	}
	ent, err := a.store.Hold(r.Context(), s.WorkspaceID, HoldInput{MailboxID: mailboxID, RFCMessageID: msgid})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ent)
}

func (a *API) Cancel(w http.ResponseWriter, r *http.Request) {
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
	ent, err := a.store.Cancel(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ent)
}

func (a *API) Reserve(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		MailboxIDs   []string `json:"mailbox_ids"`
		Strategy     string   `json:"strategy"`
		RFCMessageID string   `json:"rfc_message_id"`
		To           []string `json:"to"`
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
	msgid := strings.TrimSpace(in.RFCMessageID)
	if msgid == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "rfc_message_id required")
		return
	}
	ent, err := a.store.Reserve(r.Context(), s.WorkspaceID, ReserveInput{
		MailboxIDs: ids, Strategy: in.Strategy, RFCMessageID: msgid, To: in.To,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ent)
}

func (a *API) Send(w http.ResponseWriter, r *http.Request) {
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
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
		HTML    string   `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	ent, err := a.store.Send(r.Context(), s.WorkspaceID, id, SendInput{
		To: in.To, Subject: in.Subject, Text: in.Text, HTML: in.HTML,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ent)
}

func (a *API) Reconcile(w http.ResponseWriter, r *http.Request) {
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
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
		HTML    string   `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	ent, err := a.store.Reconcile(r.Context(), s.WorkspaceID, id, SendInput{
		To: in.To, Subject: in.Subject, Text: in.Text, HTML: in.HTML,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ent)
}

func (a *API) Usage(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStoreError(w, err)
		return
	}
	mailboxID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	n, err := a.store.Usage(r.Context(), s.WorkspaceID, mailboxID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"used": n})
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
	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, ErrNoneEligible):
		writeError(w, http.StatusConflict, "none_eligible", err.Error())
	case errors.Is(err, ErrPaused):
		writeError(w, http.StatusConflict, "sends_paused", err.Error())
	case errors.Is(err, ErrSuppressed):
		writeError(w, http.StatusConflict, "suppressed", err.Error())
	case errors.Is(err, mime.ErrTooLarge):
		writeError(w, http.StatusBadRequest, "invalid_input", "this mail is too large to send")
	default:
		if err != nil {
			switch err.Error() {
			case "add a CV to this profile", "this file's From does not match the mailbox":
				writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
				return
			}
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
