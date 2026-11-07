package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	mailsync "outreach/internal/sync"
)

type SyncArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	MailboxID   uuid.UUID `json:"mailbox_id"`
}

func (SyncArgs) Kind() string { return "mailbox.sync" }

func (SyncArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

type SyncWorker struct {
	river.WorkerDefaults[SyncArgs]
	Run func(ctx context.Context, args SyncArgs) error
}

func (w SyncWorker) Work(ctx context.Context, job *river.Job[SyncArgs]) error {
	if w.Run == nil {
		return fmt.Errorf("worker: sync run missing")
	}
	return w.Run(ctx, job.Args)
}

func (SyncWorker) NextRetry(job *river.Job[SyncArgs]) time.Time {
	return time.Now().Add(mailsync.Backoff(job.Attempt))
}

func AddSync(workers *river.Workers, run func(ctx context.Context, args SyncArgs) error) {
	river.AddWorker(workers, SyncWorker{Run: run})
}

func EnqueueSync(ctx context.Context, client *river.Client[pgx.Tx], workspaceID, mailboxID uuid.UUID) error {
	_, err := client.Insert(ctx, SyncArgs{WorkspaceID: workspaceID, MailboxID: mailboxID}, nil)
	if err != nil {
		return fmt.Errorf("worker: insert sync: %w", err)
	}
	return nil
}
