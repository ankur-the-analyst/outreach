package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"outreach/internal/auth"
	"outreach/internal/db"
	"outreach/internal/workspace"
)

type Memberships interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
}

type API struct {
	pool   *db.Pool
	store  Memberships
	client *river.Client[pgx.Tx]
}

func New(pool *db.Pool, store Memberships, client *river.Client[pgx.Tx]) *API {
	return &API{pool: pool, store: store, client: client}
}

func (a *API) Ping(w http.ResponseWriter, r *http.Request) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	role, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID)
	if err != nil {
		writeError(w, http.StatusForbidden, "forbidden", "not a member")
		return
	}
	if role != workspace.RoleOwner && role != workspace.RoleMember {
		writeError(w, http.StatusForbidden, "forbidden", "owner or member role required")
		return
	}
	var jobID int64
	var skipped bool
	err = a.pool.ExecTx(r.Context(), db.GUC{WorkspaceID: s.WorkspaceID, UserID: s.UserID}, func(ctx context.Context, q *db.Queries, tx pgx.Tx) error {
		res, err := EnqueuePing(ctx, q, tx, a.client, s.WorkspaceID)
		if err != nil {
			return err
		}
		jobID = res.Job.ID
		skipped = res.UniqueSkippedAsDuplicate
		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id":         jobID,
		"unique_skipped": skipped,
	})
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
