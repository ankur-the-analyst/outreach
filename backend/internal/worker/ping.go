package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"outreach/internal/db"
)

type PingArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
}

func (PingArgs) Kind() string { return "workspace.ping" }

func (PingArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

type PingWorker struct {
	river.WorkerDefaults[PingArgs]
}

func (PingWorker) Work(ctx context.Context, job *river.Job[PingArgs]) error {
	q := QueriesFrom(ctx)
	if q == nil {
		return fmt.Errorf("worker: guc queries missing")
	}
	if _, err := q.GetWorkspace(ctx, job.Args.WorkspaceID); err != nil {
		return fmt.Errorf("worker: ping: %w", err)
	}
	return nil
}

func EnqueuePing(ctx context.Context, q *db.Queries, tx pgx.Tx, client *river.Client[pgx.Tx], workspaceID uuid.UUID) (*rivertype.JobInsertResult, error) {
	if err := q.TouchWorkspace(ctx, workspaceID); err != nil {
		return nil, fmt.Errorf("worker: touch: %w", err)
	}
	res, err := client.InsertTx(ctx, tx, PingArgs{WorkspaceID: workspaceID}, nil)
	if err != nil {
		return nil, fmt.Errorf("worker: insert ping: %w", err)
	}
	return res, nil
}
