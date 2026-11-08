package suppression

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
)

func TestListAfterAdd(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
		c, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteSuppressionForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "supp-list"})
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool, keys)
	if _, err := store.Add(ctx, wsID, KindEmail, "ada@acme.com", "opt-out"); err != nil {
		t.Fatal(err)
	}
	rows, err := store.List(ctx, wsID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Value != "ada@acme.com" || rows[0].Reason != "opt-out" {
		t.Fatalf("%+v", rows)
	}
}
