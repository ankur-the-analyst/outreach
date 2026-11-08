package contacts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/suppression"
)

func TestEraseRemovesContactKeepsHashedSuppression(t *testing.T) {
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
	userID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteSuppressionForWorkspace(ctx, wsID)
			_ = q.DeleteContactsForWorkspace(ctx, wsID)
			_ = q.DeleteCompaniesForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m64-erase"})
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool).WithHMAC(keys)
	c, err := store.Create(ctx, wsID, CreateInput{
		Email: "Ada@acme.com", FirstName: "Ada", Company: "Acme", Source: "typed", AddedBy: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Erase(ctx, wsID, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, wsID, c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("contact still there %v", err)
	}
	sum, _, err := suppression.Hash(keys, suppression.KindEmail, "ada@acme.com")
	if err != nil {
		t.Fatal(err)
	}
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetSuppression(ctx, db.GetSuppressionParams{
			WorkspaceID: wsID, Kind: suppression.KindEmail, ValueHash: sum,
		})
		if err != nil {
			return err
		}
		if row.ValueCitext.Valid {
			return fmt.Errorf("plaintext still %q", row.ValueCitext.String)
		}
		if row.Reason != "erasure" {
			return fmt.Errorf("reason %s", row.Reason)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Erase(ctx, wsID, c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second erase %v", err)
	}
}
