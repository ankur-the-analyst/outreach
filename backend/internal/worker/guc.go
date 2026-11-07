package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"outreach/internal/db"
)

type ctxQueries struct{}

func QueriesFrom(ctx context.Context) *db.Queries {
	q, _ := ctx.Value(ctxQueries{}).(*db.Queries)
	return q
}

func WorkspaceGUC(pool *db.Pool) river.WorkerMiddlewareFunc {
	return func(ctx context.Context, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
		var payload struct {
			WorkspaceID uuid.UUID `json:"workspace_id"`
		}
		if err := json.Unmarshal(job.EncodedArgs, &payload); err != nil {
			return fmt.Errorf("worker: args: %w", err)
		}
		if payload.WorkspaceID == uuid.Nil {
			return fmt.Errorf("worker: workspace_id required")
		}
		return pool.WithGUC(ctx, db.GUC{WorkspaceID: payload.WorkspaceID}, func(ctx context.Context, q *db.Queries) error {
			return doInner(context.WithValue(ctx, ctxQueries{}, q))
		})
	}
}
