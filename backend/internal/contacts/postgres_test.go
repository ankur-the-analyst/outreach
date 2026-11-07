package contacts

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/db"
)

func TestCreateRequiresProvenanceAndDedupsEmail(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
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
			_ = q.DeleteContactsForWorkspace(ctx, wsID)
			_ = q.DeleteCompaniesForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m61"})
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool)
	if _, err := store.Create(ctx, wsID, CreateInput{Email: "ada@acme.com", AddedBy: userID}); err != ErrProvenance {
		t.Fatalf("want provenance %v", err)
	}
	c, err := store.Create(ctx, wsID, CreateInput{
		Email: "Ada@acme.com", FirstName: "Ada", Company: "Acme", Source: "typed", AddedBy: userID,
	})
	if err != nil || c.Email != "ada@acme.com" || c.Readiness != "ready" || c.Provenance.Source != "typed" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := store.Create(ctx, wsID, CreateInput{Email: "ada@acme.com", Source: "csv", AddedBy: userID}); err != ErrDuplicate {
		t.Fatalf("want duplicate %v", err)
	}
	risky, err := store.Create(ctx, wsID, CreateInput{Email: "careers@acme.com", Source: "typed", AddedBy: userID})
	if err != nil || risky.RoleClass != "generic_mailbox" || risky.Readiness != "risky" {
		t.Fatalf("%+v %v", risky, err)
	}
}
