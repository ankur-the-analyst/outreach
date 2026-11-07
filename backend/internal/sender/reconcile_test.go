package sender

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
)

func TestFindSentTrueDoesNotResend(t *testing.T) {
	led, stub, wsID, mbID, cleanup := setupReconcile(t)
	defer cleanup()
	ctx := context.Background()
	stub.sendErr = context.DeadlineExceeded
	stub.finds = true
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mbID, RFCMessageID: "<stable-unknown@outreach.test>"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	unk, err := led.Send(ctx, wsID, held.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("send err %v", err)
	}
	if unk.State != pacing.StateUnknown {
		t.Fatalf("%s", unk.State)
	}
	if stub.sends != 1 {
		t.Fatalf("sends %d", stub.sends)
	}
	stub.sendErr = nil
	got, err := led.Reconcile(ctx, wsID, held.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != pacing.StateSent {
		t.Fatalf("%s", got.State)
	}
	if stub.sends != 1 {
		t.Fatalf("second delivery sends=%d", stub.sends)
	}
}

func TestUnknownRetrySameMessageID(t *testing.T) {
	led, stub, wsID, mbID, cleanup := setupReconcile(t)
	defer cleanup()
	led.WithUnknownTimeout(0)
	ctx := context.Background()
	stub.sendErr = context.DeadlineExceeded
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mbID, RFCMessageID: "<retry-same@outreach.test>"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	if _, err := led.Send(ctx, wsID, held.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("send err %v", err)
	}
	stub.sendErr = nil
	got, err := led.Reconcile(ctx, wsID, held.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != pacing.StateSent {
		t.Fatalf("%s", got.State)
	}
	if stub.sends != 2 {
		t.Fatalf("sends %d", stub.sends)
	}
	if stub.last == nil || stub.last.MessageID != "<retry-same@outreach.test>" {
		t.Fatalf("%+v", stub.last)
	}
}

func TestUnknownYoungSkipsRetry(t *testing.T) {
	led, stub, wsID, mbID, cleanup := setupReconcile(t)
	defer cleanup()
	led.WithUnknownTimeout(time.Hour)
	ctx := context.Background()
	stub.sendErr = context.DeadlineExceeded
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mbID, RFCMessageID: "<young@outreach.test>"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	if _, err := led.Send(ctx, wsID, held.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("send err %v", err)
	}
	got, err := led.Reconcile(ctx, wsID, held.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != pacing.StateUnknown {
		t.Fatalf("%s", got.State)
	}
	if stub.sends != 1 {
		t.Fatalf("sends %d", stub.sends)
	}
}

func setupReconcile(t *testing.T) (*Postgres, *stubTransport, uuid.UUID, uuid.UUID, func()) {
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "send-m35"}); err != nil {
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
	return led, stub, wsID, mb.ID, cleanup
}
