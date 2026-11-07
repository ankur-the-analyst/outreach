package worker

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"outreach/internal/db"
)

func TestEnqueuePingSameTxAsWrite(t *testing.T) {
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
		_, _ = pool.Pgx().Exec(c, "delete from river_job where args->>'workspace_id' = $1", wsID.String())
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			return q.DeleteWorkspace(ctx, wsID)
		})
		pool.Close()
	})
	err = pool.InTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "river-ping"})
	})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	client, err := NewInsertClient(pool)
	if err != nil {
		t.Fatal(err)
	}

	err = pool.ExecTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries, tx pgx.Tx) error {
		if _, err := EnqueuePing(ctx, q, tx, client, wsID); err != nil {
			return err
		}
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected boom")
	}
	var n int
	if err := pool.Pgx().QueryRow(ctx, "select count(*) from river_job where args->>'workspace_id' = $1", wsID.String()).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("rolled back job still present: %d", n)
	}

	res, err := enqueueCommitted(ctx, pool, client, wsID)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.UniqueSkippedAsDuplicate {
		t.Fatal("first insert skipped")
	}
	dup, err := enqueueCommitted(ctx, pool, client, wsID)
	if err != nil {
		t.Fatalf("dup: %v", err)
	}
	if !dup.UniqueSkippedAsDuplicate {
		t.Fatal("expected unique skip")
	}
	if err := pool.Pgx().QueryRow(ctx, "select count(*) from river_job where args->>'workspace_id' = $1", wsID.String()).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("unique jobs %d want 1", n)
	}
}

func enqueueCommitted(ctx context.Context, pool *db.Pool, client *river.Client[pgx.Tx], wsID uuid.UUID) (*rivertype.JobInsertResult, error) {
	var res *rivertype.JobInsertResult
	err := pool.ExecTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries, tx pgx.Tx) error {
		var err error
		res, err = EnqueuePing(ctx, q, tx, client, wsID)
		return err
	})
	return res, err
}
