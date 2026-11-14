package sync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	SyncMailbox(ctx context.Context, workspaceID, mailboxID uuid.UUID) (int, error)
	ListThreads(ctx context.Context, workspaceID, mailboxID uuid.UUID) ([]Thread, error)
	GetThread(ctx context.Context, workspaceID, threadID uuid.UUID) (Thread, error)
	MarkRead(ctx context.Context, workspaceID, threadID uuid.UUID) error
}

type API struct {
	store Store
}

func New(store Store) *API {
	return &API{store: store}
}

func (a *API) Sync(w http.ResponseWriter, r *http.Request) {
	s, mailboxID, ok := a.mailbox(w, r)
	if !ok {
		return
	}
	if err := a.requireEdit(r.Context(), s); err != nil {
		writeError(w, http.StatusForbidden, "forbidden", "owner or member role required")
		return
	}
	n, err := a.sync(r, s.WorkspaceID, mailboxID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		if strings.Contains(err.Error(), "date") {
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ingested": n})
}

type ranged interface {
	SyncRange(ctx context.Context, workspaceID, mailboxID uuid.UUID, from, to time.Time) (int, error)
}

func (a *API) sync(r *http.Request, workspaceID, mailboxID uuid.UUID) (int, error) {
	from, to, err := dateQuery(r)
	if err != nil {
		return 0, err
	}
	if from.IsZero() && to.IsZero() {
		return a.store.SyncMailbox(r.Context(), workspaceID, mailboxID)
	}
	rng, ok := a.store.(ranged)
	if !ok {
		return 0, errors.New("date range is not available")
	}
	return rng.SyncRange(r.Context(), workspaceID, mailboxID, from, to)
}

func dateQuery(r *http.Request) (time.Time, time.Time, error) {
	from, err := day(r.URL.Query().Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := day(r.URL.Query().Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		return time.Time{}, time.Time{}, errors.New("end date is before the start date")
	}
	return from, to, nil
}

func day(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return time.Time{}, errors.New("use a date like 2026-11-13")
	}
	return t.UTC(), nil
}

func (a *API) Threads(w http.ResponseWriter, r *http.Request) {
	s, mailboxID, ok := a.mailbox(w, r)
	if !ok {
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeError(w, http.StatusForbidden, "forbidden", "not a member")
		return
	}
	rows, err := a.store.ListThreads(r.Context(), s.WorkspaceID, mailboxID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if rows == nil {
		rows = []Thread{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"threads": rows})
}

func (a *API) Thread(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeError(w, http.StatusForbidden, "forbidden", "not a member")
		return
	}
	row, err := a.store.GetThread(r.Context(), s.WorkspaceID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (a *API) MarkRead(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeError(w, http.StatusForbidden, "forbidden", "not a member")
		return
	}
	if err := a.store.MarkRead(r.Context(), s.WorkspaceID, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) mailbox(w http.ResponseWriter, r *http.Request) (auth.Session, uuid.UUID, bool) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return auth.Session{}, uuid.Nil, false
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid id")
		return auth.Session{}, uuid.Nil, false
	}
	return s, id, true
}

func (a *API) requireEdit(ctx context.Context, s auth.Session) error {
	role, err := a.store.Membership(ctx, s.WorkspaceID, s.UserID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "member" {
		return errForbidden
	}
	return nil
}

var errForbidden = errors.New("forbidden")

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
