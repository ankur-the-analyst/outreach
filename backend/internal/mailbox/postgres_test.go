package mailbox

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/pacing"
)

func TestPostgresConnectPauseReconnect(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
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
			_ = q.DeleteMailboxesForWorkspace(ctx, wsID)
			_ = q.DeleteSecretsForWorkspace(ctx, wsID)
			_ = q.DeleteWarmupProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteSendingDomainsForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.InTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "mb-m23"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	store := NewPostgres(pool, keys)
	mb, err := store.Connect(ctx, wsID, ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if mb.Status != pacing.StatusWarming || mb.EffectiveDailyLimit != 2 {
		t.Fatalf("%+v", mb)
	}
	if err := store.Pause(ctx, wsID, mb.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Reconnect(ctx, wsID, mb.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.AssignWarmup(ctx, wsID, mb.ID, pacing.KindConservative); err != nil {
		t.Fatal(err)
	}
	items, err := store.List(ctx, wsID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list %v %d", err, len(items))
	}
	if items[0].Status != pacing.StatusWarming {
		t.Fatalf("status %s", items[0].Status)
	}

	_, err = store.Connect(ctx, wsID, ConnectInput{
		Email: "me@gmail.com", AuthKind: "service_account", Timezone: "UTC",
	})
	if err != ErrGmailServiceAccount {
		t.Fatalf("gmail sa: %v", err)
	}
}
