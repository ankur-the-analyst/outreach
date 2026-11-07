package sender

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
)

func TestHoldCountsCancelFrees(t *testing.T) {
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
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "ledger-m32"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	led := NewPostgres(pool, func() time.Time { return time.Now().UTC() }, 5*time.Minute)
	used, err := led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 0 {
		t.Fatalf("usage before %d %v", used, err)
	}
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mb.ID, RFCMessageID: "<m32@outreach.test>"})
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	if held.State != pacing.StateHeld || held.ReleaseAt == nil {
		t.Fatalf("%+v", held)
	}
	used, err = led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 1 {
		t.Fatalf("hold should count: %d %v", used, err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateSent); err == nil {
		t.Fatal("held → sent should conflict")
	}
	cancelled, err := led.Cancel(ctx, wsID, held.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.State != pacing.StateCancelled {
		t.Fatalf("%s", cancelled.State)
	}
	used, err = led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 0 {
		t.Fatalf("cancel should free: %d %v", used, err)
	}

	held2, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mb.ID, RFCMessageID: "<m32b@outreach.test>"})
	if err != nil {
		t.Fatalf("hold2: %v", err)
	}
	if _, err := led.Advance(ctx, wsID, held2.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held2.ID, pacing.StateSending); err != nil {
		t.Fatal(err)
	}
	sent, err := led.Advance(ctx, wsID, held2.ID, pacing.StateSent)
	if err != nil {
		t.Fatal(err)
	}
	if sent.State != pacing.StateSent {
		t.Fatalf("%s", sent.State)
	}
	used, err = led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 1 {
		t.Fatalf("sent should count: %d %v", used, err)
	}
}
