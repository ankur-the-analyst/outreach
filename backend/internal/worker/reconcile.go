package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type ReconcileArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	LedgerID    uuid.UUID `json:"ledger_id"`
	To          []string  `json:"to"`
	Subject     string    `json:"subject"`
	Text        string    `json:"text"`
	HTML        string    `json:"html"`
}

func (ReconcileArgs) Kind() string { return "ledger.reconcile" }

func (ReconcileArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

type ReconcileWorker struct {
	river.WorkerDefaults[ReconcileArgs]
	Run func(ctx context.Context, args ReconcileArgs) error
}

func (w ReconcileWorker) Work(ctx context.Context, job *river.Job[ReconcileArgs]) error {
	if w.Run == nil {
		return fmt.Errorf("worker: reconcile run missing")
	}
	return w.Run(ctx, job.Args)
}

func AddReconcile(workers *river.Workers, run func(ctx context.Context, args ReconcileArgs) error) {
	river.AddWorker(workers, ReconcileWorker{Run: run})
}

func EnqueueReconcile(ctx context.Context, client *river.Client[pgx.Tx], args ReconcileArgs) error {
	_, err := client.Insert(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("worker: insert reconcile: %w", err)
	}
	return nil
}
