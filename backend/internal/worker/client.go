package worker

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"outreach/internal/db"
)

func NewWorkers() *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, PingWorker{})
	river.AddWorker(workers, StepWorker{})
	return workers
}

func NewInsertClient(pool *db.Pool) (*river.Client[pgx.Tx], error) {
	client, err := river.NewClient(riverpgxv5.New(pool.Pgx()), &river.Config{})
	if err != nil {
		return nil, fmt.Errorf("worker: insert client: %w", err)
	}
	return client, nil
}

func NewWorkerClient(pool *db.Pool, extra func(*river.Workers)) (*river.Client[pgx.Tx], error) {
	workers := NewWorkers()
	if extra != nil {
		extra(workers)
	}
	client, err := river.NewClient(riverpgxv5.New(pool.Pgx()), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 2},
		},
		Workers:    workers,
		Middleware: []rivertype.Middleware{WorkspaceGUC(pool)},
	})
	if err != nil {
		return nil, fmt.Errorf("worker: client: %w", err)
	}
	return client, nil
}
