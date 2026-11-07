package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/riverqueue/river/rivertype"
	"outreach/internal/db"
)

func TestPingWorkerSetsWorkspaceGUC(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			return q.DeleteWorkspace(ctx, wsID)
		})
		pool.Close()
	})
	err = pool.InTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "river-guc"})
	})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}

	tw := rivertest.NewWorker(t, riverpgxv5.New(pool.Pgx()), &river.Config{
		Middleware: []rivertype.Middleware{WorkspaceGUC(pool)},
	}, PingWorker{})
	tx, err := pool.Pgx().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tw.Work(ctx, t, tx, PingArgs{WorkspaceID: wsID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.EventKind != river.EventKindJobCompleted {
		t.Fatalf("kind %s", result.EventKind)
	}
}
