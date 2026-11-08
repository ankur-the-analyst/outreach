package workspace

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
	Get(ctx context.Context, workspaceID uuid.UUID) (Workspace, error)
	SetName(ctx context.Context, workspaceID uuid.UUID, name string) error
	SetPaused(ctx context.Context, workspaceID uuid.UUID, paused bool) error
	Delete(ctx context.Context, workspaceID uuid.UUID) error
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]Member, error)
	UserByEmail(ctx context.Context, email string) (uuid.UUID, error)
	CreateInvitedUser(ctx context.Context, email, passwordHash string) (uuid.UUID, bool, error)
	AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) Get(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	role, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	ws, err := a.store.Get(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           ws.ID,
		"name":         ws.Name,
		"sends_paused": ws.SendsPaused,
		"role":         role,
	})
}

func (a *API) Patch(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	if err := a.requireOwner(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "name required")
		return
	}
	if err := a.store.SetName(r.Context(), s.WorkspaceID, name); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) Delete(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	if err := a.requireOwner(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	if err := a.store.Delete(r.Context(), s.WorkspaceID); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) PutKillSwitch(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	if err := a.requireOwner(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		Paused *bool `json:"paused"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Paused == nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "paused required")
		return
	}
	if err := a.store.SetPaused(r.Context(), s.WorkspaceID, *in.Paused); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sends_paused": *in.Paused})
}

func (a *API) ListMembers(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStoreError(w, err)
		return
	}
	members, err := a.store.ListMembers(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (a *API) Invite(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	if err := a.requireOwner(r.Context(), s); err != nil {
		writeStoreError(w, err)
		return
	}
	var in struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	email := strings.TrimSpace(strings.ToLower(in.Email))
	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = RoleMember
	}
	if email == "" || !InviteRoleOK(role) {
		writeError(w, http.StatusBadRequest, "invalid_input", "email and role member|viewer required")
		return
	}
	userID, err := a.store.UserByEmail(r.Context(), email)
	password := ""
	if errors.Is(err, ErrNotFound) {
		plain, err := auth.RandomPassword()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not add person")
			return
		}
		hash, err := auth.HashPassword(plain)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "could not add person")
			return
		}
		created := false
		userID, created, err = a.store.CreateInvitedUser(r.Context(), email, hash)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if created {
			password = plain
		}
	} else if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := a.store.AddMember(r.Context(), s.WorkspaceID, userID, role); err != nil {
		writeStoreError(w, err)
		return
	}
	out := map[string]any{
		"user_id": userID,
		"email":   email,
		"role":    role,
	}
	if password != "" {
		out["password"] = password
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *API) requireOwner(ctx context.Context, s auth.Session) error {
	role, err := a.store.Membership(ctx, s.WorkspaceID, s.UserID)
	if err != nil {
		return err
	}
	if !CanManage(role) {
		return ErrForbidden
	}
	return nil
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "owner role required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, ErrAlreadyMember):
		writeError(w, http.StatusConflict, "already_member", "already a member")
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
