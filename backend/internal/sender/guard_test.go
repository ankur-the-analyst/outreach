package sender

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
	"outreach/internal/suppression"
)

func TestKillSwitchBlocksReserveNotSend(t *testing.T) {
	led, mbID, wsID, cleanup := setupGuard(t)
	defer cleanup()
	ctx := context.Background()
	err := led.pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.SetSendsPaused(ctx, db.SetSendsPausedParams{ID: wsID, SendsPaused: true})
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = led.Reserve(ctx, wsID, ReserveInput{
		MailboxIDs: []uuid.UUID{mbID}, RFCMessageID: "<ks@outreach.test>",
	})
	if !errors.Is(err, ErrPaused) {
		t.Fatalf("reserve %v", err)
	}
	err = led.pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.SetSendsPaused(ctx, db.SetSendsPausedParams{ID: wsID, SendsPaused: false})
	})
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := led.Reserve(ctx, wsID, ReserveInput{
		MailboxIDs: []uuid.UUID{mbID}, RFCMessageID: "<ks2@outreach.test>",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = led.pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.SetSendsPaused(ctx, db.SetSendsPausedParams{ID: wsID, SendsPaused: true})
	})
	if err != nil {
		t.Fatal(err)
	}
	sent, err := led.Send(ctx, wsID, reserved.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if sent.State != pacing.StateSent {
		t.Fatalf("%s", sent.State)
	}
}

func TestSuppressionBlocksReserve(t *testing.T) {
	led, mbID, wsID, cleanup := setupGuard(t)
	defer cleanup()
	ctx := context.Background()
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	sup := suppression.NewPostgres(led.pool, keys)
	if _, err := sup.Add(ctx, wsID, suppression.KindEmail, "sink@outreach.test", "opt-out"); err != nil {
		t.Fatal(err)
	}
	led.WithBlocker(sup)
	_, err = led.Reserve(ctx, wsID, ReserveInput{
		MailboxIDs: []uuid.UUID{mbID}, RFCMessageID: "<sup@outreach.test>",
		To: []string{"Sink@Outreach.test"},
	})
	if !errors.Is(err, ErrSuppressed) {
		t.Fatalf("reserve %v", err)
	}
}

func TestPausedMailboxNotReserved(t *testing.T) {
	led, mbID, wsID, cleanup := setupGuard(t)
	defer cleanup()
	ctx := context.Background()
	err := led.pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.SetMailboxStatus(ctx, db.SetMailboxStatusParams{
			WorkspaceID:  wsID,
			ID:           mbID,
			Status:       pacing.StatusPaused,
			StatusReason: pgtype.Text{String: "paused", Valid: true},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = led.Reserve(ctx, wsID, ReserveInput{
		MailboxIDs: []uuid.UUID{mbID}, RFCMessageID: "<paused@outreach.test>",
	})
	if !errors.Is(err, ErrNoneEligible) {
		t.Fatalf("reserve %v", err)
	}
}

func setupGuard(t *testing.T) (*Postgres, uuid.UUID, uuid.UUID, func()) {
	t.Helper()
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	pool, err := db.Open(ctx, url)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	err = pool.InTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m36"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	stub := &stubTransport{}
	led := NewPostgres(pool, time.Now, time.Minute).WithDialer(stubDialer{tr: stub})
	cleanup := func() {
		c, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteSuppressionForWorkspace(ctx, wsID)
			_ = q.DeleteMailboxesForWorkspace(ctx, wsID)
			_ = q.DeleteSecretsForWorkspace(ctx, wsID)
			_ = q.DeleteWarmupProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteSendingDomainsForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
		cancel()
	}
	return led, mb.ID, wsID, cleanup
}
