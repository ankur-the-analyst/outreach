package mailbox

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
	List(ctx context.Context, workspaceID uuid.UUID) ([]Mailbox, error)
	Connect(ctx context.Context, workspaceID uuid.UUID, in ConnectInput) (Mailbox, error)
	Pause(ctx context.Context, workspaceID, id uuid.UUID) error
	Reconnect(ctx context.Context, workspaceID, id uuid.UUID) error
	AssignWarmup(ctx context.Context, workspaceID, id uuid.UUID, kind string) error
	SetReplyMode(ctx context.Context, workspaceID, id uuid.UUID, mode string) error
	CreateSA(ctx context.Context, workspaceID uuid.UUID, in SAInput) (ServiceAccount, error)
	ListSA(ctx context.Context, workspaceID uuid.UUID) ([]ServiceAccount, error)
	SMTPEndpoint(ctx context.Context, workspaceID, id uuid.UUID) (host string, port int, err error)
}

type SMTPTester interface {
	Test(ctx context.Context, host string, port int) error
}

type API struct {
	store Store
	smtp  SMTPTester
}

func New(store Store, smtp SMTPTester) *API {
	return &API{store: store, smtp: smtp}
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
	items, err := a.store.List(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mailboxes": items})
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
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		AuthKind    string `json:"auth_kind"`
		Password    string `json:"password"`
		Timezone    string `json:"timezone"`
		SMTPHost    string `json:"smtp_host"`
		SMTPPort    int    `json:"smtp_port"`
		SMTPTLS     string `json:"smtp_tls"`
		IMAPHost    string `json:"imap_host"`
		IMAPPort    int    `json:"imap_port"`
		WarmupKind  string `json:"warmup_kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(in.Email))
	kind := strings.TrimSpace(in.AuthKind)
	if kind == "" {
		kind = "app_password"
	}
	if email == "" || strings.TrimSpace(in.Timezone) == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "email and timezone required")
		return
	}
	if kind == "oauth" {
		writeStoreError(w, ErrNotConfigured)
		return
	}
	if kind == "app_password" && in.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "password required")
		return
	}
	if err := RejectGmailServiceAccount(email, kind); err != nil {
		writeStoreError(w, err)
		return
	}
	mb, err := a.store.Connect(r.Context(), s.WorkspaceID, ConnectInput{
		Email: email, DisplayName: in.DisplayName, AuthKind: kind, Password: in.Password,
		Timezone: in.Timezone, SMTPHost: in.SMTPHost, SMTPPort: in.SMTPPort, SMTPTLS: in.SMTPTLS,
		IMAPHost: in.IMAPHost, IMAPPort: in.IMAPPort, WarmupKind: in.WarmupKind,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, mb)
}

func (a *API) CreateSA(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		Domain  string          `json:"domain"`
		JSONKey json.RawMessage `json:"json_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	sa, err := a.store.CreateSA(r.Context(), s.WorkspaceID, SAInput{Domain: in.Domain, JSON: in.JSONKey})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sa)
}

func (a *API) ListSA(w http.ResponseWriter, r *http.Request) {
	s, ok := a.session(w, r)
	if !ok {
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStoreError(w, err)
		return
	}
	items, err := a.store.ListSA(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"service_accounts": items})
}

func (a *API) Pause(w http.ResponseWriter, r *http.Request) {
	a.mutateStatus(w, r, a.store.Pause)
}

func (a *API) Reconnect(w http.ResponseWriter, r *http.Request) {
	a.mutateStatus(w, r, a.store.Reconnect)
}

func (a *API) Test(w http.ResponseWriter, r *http.Request) {
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
	host, port, err := a.store.SMTPEndpoint(r.Context(), s.WorkspaceID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := a.smtp.Test(r.Context(), host, port); err != nil {
		writeError(w, http.StatusBadGateway, "test_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
		WarmupKind string `json:"warmup_kind"`
		ReplyMode  string `json:"reply_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	kind := strings.TrimSpace(in.WarmupKind)
	mode := strings.TrimSpace(in.ReplyMode)
	if kind == "" && mode == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "warmup_kind or reply_mode required")
		return
	}
	if mode != "" {
		if err := a.store.SetReplyMode(r.Context(), s.WorkspaceID, id, mode); err != nil {
			writeStoreError(w, err)
			return
		}
	}
	if kind != "" {
		if err := a.store.AssignWarmup(r.Context(), s.WorkspaceID, id, kind); err != nil {
			writeStoreError(w, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) mutateStatus(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID, uuid.UUID) error) {
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
	if err := fn(r.Context(), s.WorkspaceID, id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	if !CanEdit(role) {
		return ErrForbidden
	}
	return nil
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrGmailServiceAccount):
		writeError(w, http.StatusBadRequest, "gmail_sa", err.Error())
	case errors.Is(err, ErrNotConfigured):
		writeError(w, http.StatusBadRequest, "not_configured", err.Error())
	case errors.Is(err, ErrNotSupported):
		writeError(w, http.StatusBadRequest, "not_supported", err.Error())
	case errors.Is(err, ErrSADomainTaken):
		writeError(w, http.StatusConflict, "domain_taken", err.Error())
	case errors.Is(err, ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", err.Error())
	case errors.Is(err, ErrNoSMTP):
		writeError(w, http.StatusBadRequest, "no_smtp", err.Error())
	case errors.Is(err, ErrReplyMode):
		writeError(w, http.StatusBadRequest, "reply_mode", err.Error())
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "owner or member role required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
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
