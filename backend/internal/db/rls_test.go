package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWrongWorkspaceGUCReadsNothing(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	var wsA uuid.UUID
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if wsA != uuid.Nil {
			_ = pool.InTx(c, GUC{WorkspaceID: wsA}, func(ctx context.Context, q *Queries) error {
				return q.DeleteWorkspace(ctx, wsA)
			})
		}
		pool.Close()
	})

	wsA, err = uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	wsB, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}

	err = pool.InTx(ctx, GUC{WorkspaceID: wsA}, func(ctx context.Context, q *Queries) error {
		return q.InsertWorkspace(ctx, InsertWorkspaceParams{ID: wsA, Name: "rls-probe"})
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	nNone, err := pool.Queries().CountWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if nNone != 0 {
		t.Fatalf("no GUC: got %d want 0", nNone)
	}

	var nWrong int64
	err = pool.WithGUC(ctx, GUC{WorkspaceID: wsB}, func(ctx context.Context, q *Queries) error {
		nWrong, err = q.CountWorkspaces(ctx)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if nWrong != 0 {
		t.Fatalf("wrong GUC: got %d want 0", nWrong)
	}

	var nRight int64
	err = pool.WithGUC(ctx, GUC{WorkspaceID: wsA}, func(ctx context.Context, q *Queries) error {
		nRight, err = q.CountWorkspaces(ctx)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if nRight != 1 {
		t.Fatalf("matching GUC: got %d want 1", nRight)
	}

	nAfter, err := pool.Queries().CountWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if nAfter != 0 {
		t.Fatalf("after WithGUC reset: got %d want 0", nAfter)
	}
}
