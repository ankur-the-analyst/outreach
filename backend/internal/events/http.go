package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	ReviewN(ctx context.Context, workspaceID uuid.UUID) (int64, error)
}

type API struct {
	store Store
	every time.Duration
}

func New(store Store) *API {
	return &API{store: store, every: 15 * time.Second}
}

func (a *API) Stream(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	tick := a.every
	if tick <= 0 {
		tick = 15 * time.Second
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	var last int64
	var sent bool
	if err := a.emit(w, fl, r.Context(), s.WorkspaceID, &last, &sent); err != nil {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-t.C:
			if err := a.emit(w, fl, r.Context(), s.WorkspaceID, &last, &sent); err != nil {
				return
			}
		}
	}
}

func (a *API) emit(w http.ResponseWriter, fl http.Flusher, ctx context.Context, workspaceID uuid.UUID, last *int64, sent *bool) error {
	n, err := a.store.ReviewN(ctx, workspaceID)
	if err != nil {
		return err
	}
	if *sent && *last == n {
		fmt.Fprintf(w, ": ping\n\n")
		fl.Flush()
		return nil
	}
	*sent = true
	*last = n
	fmt.Fprintf(w, "event: review\ndata: {\"n\":%d}\n\n", n)
	fl.Flush()
	return nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}
