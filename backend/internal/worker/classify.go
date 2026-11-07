package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type ClassifyArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	MessageID   uuid.UUID `json:"message_id"`
}

func (ClassifyArgs) Kind() string { return "inbound.classify" }

func (ClassifyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

type ClassifyWorker struct {
	river.WorkerDefaults[ClassifyArgs]
	Run func(ctx context.Context, args ClassifyArgs) error
}

func (w ClassifyWorker) Work(ctx context.Context, job *river.Job[ClassifyArgs]) error {
	if w.Run == nil {
		return fmt.Errorf("worker: classify run missing")
	}
	return w.Run(ctx, job.Args)
}

func AddClassify(workers *river.Workers, run func(ctx context.Context, args ClassifyArgs) error) {
	river.AddWorker(workers, ClassifyWorker{Run: run})
}

func EnqueueClassify(ctx context.Context, client *river.Client[pgx.Tx], workspaceID, messageID uuid.UUID) error {
	_, err := client.Insert(ctx, ClassifyArgs{WorkspaceID: workspaceID, MessageID: messageID}, nil)
	if err != nil {
		return fmt.Errorf("worker: insert classify: %w", err)
	}
	return nil
}
