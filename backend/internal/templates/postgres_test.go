package templates

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/db"
)

func TestCreateRejectsUnknownVarInButtonHref(t *testing.T) {
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
		c, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteTemplatesForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m101"})
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool)
	_, err = store.Create(ctx, wsID, CreateInput{
		Name: "intro", Subject: "Hi", BodyText: "Hello",
		Design: json.RawMessage(`{"elements":[{"type":"button","href":"https://x.test?n={{contact.nope}}"}]}`),
	})
	if !errors.Is(err, ErrUnknown) {
		t.Fatalf("want unknown %v", err)
	}
	got, err := store.Create(ctx, wsID, CreateInput{
		Name: "intro", Subject: "Hi {{contact.first_name}}", BodyText: "Hello",
		BodyHTML: `<p>Hi</p>`,
		Design:   json.RawMessage(`{"elements":[{"type":"button","href":"https://x.test?n={{contact.first_name}}"}]}`),
	})
	if err != nil || got.ID == uuid.Nil || string(got.Design) == "" {
		t.Fatalf("%+v %v", got, err)
	}
}
