package compose

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
	"outreach/internal/sender"
)

func TestApproveHoldAndUndoCancels(t *testing.T) {
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
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m37"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	led := sender.NewPostgres(pool, time.Now, time.Minute)
	store := NewPostgres(pool, led)
	d, err := store.Create(ctx, wsID, CreateInput{
		MailboxID: mb.ID, To: "sink@outreach.test", Subject: "hello", Text: "body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "pending" || d.RFCMessageID == "" {
		t.Fatalf("%+v", d)
	}
	held, err := store.Approve(ctx, wsID, userID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if held.Status != "approved" || held.LedgerState != pacing.StateHeld || held.LedgerID == nil {
		t.Fatalf("%+v", held)
	}
	used, err := led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 1 {
		t.Fatalf("usage %d %v", used, err)
	}
	undone, err := store.Undo(ctx, wsID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if undone.Status != "cancelled" {
		t.Fatalf("%s", undone.Status)
	}
	used, err = led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 0 {
		t.Fatalf("usage after undo %d %v", used, err)
	}
	_, err = led.Advance(ctx, wsID, *held.LedgerID, pacing.StateReserved)
	if !errors.Is(err, sender.ErrConflict) {
		t.Fatalf("send path after undo: %v", err)
	}
}
