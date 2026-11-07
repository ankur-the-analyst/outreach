package sync

import (
	"context"
	"net/smtp"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
)

type fakeSource struct{ items []Item }

func (f fakeSource) List(context.Context) ([]Item, error) { return f.items, nil }

func TestInReplyToJoinsThread(t *testing.T) {
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
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteMessagesForWorkspace(ctx, wsID)
			_ = q.DeleteThreadsForWorkspace(ctx, wsID)
			_ = q.DeleteSyncStateForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m41"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "pit-sync@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := &fakeSource{items: []Item{
		{RFCMessageID: "<a@x>", Subject: "Hello", From: "other@x.test", To: []string{"pit-sync@example.test"}, Text: "hi", At: time.Now().UTC()},
		{RFCMessageID: "<b@x>", InReplyTo: "<a@x>", Subject: "Re: Hello", From: "other@x.test", To: []string{"pit-sync@example.test"}, Text: "reply", At: time.Now().UTC()},
	}}
	store := NewPostgres(pool, src)
	n, err := store.SyncMailbox(ctx, wsID, mb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("ingested %d", n)
	}
	threads, err := store.ListThreads(ctx, wsID, mb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || len(threads[0].Messages) != 2 {
		t.Fatalf("%+v", threads)
	}
	if threads[0].Messages[0].Direction != "in" {
		t.Fatalf("%+v", threads[0].Messages)
	}
	n, err = store.SyncMailbox(ctx, wsID, mb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("second sync %d", n)
	}
	threads, err = store.ListThreads(ctx, wsID, mb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || len(threads[0].Messages) != 2 {
		t.Fatalf("dedup %+v", threads)
	}
}

func TestMailpitReplyAppearsAsThread(t *testing.T) {
	api := os.Getenv("MAILPIT_API")
	smtpAddr := os.Getenv("MAILPIT_SMTP")
	url := os.Getenv("DATABASE_APP_URL")
	if api == "" || smtpAddr == "" || url == "" {
		t.Skip("MAILPIT_API, MAILPIT_SMTP, or DATABASE_APP_URL not set")
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
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteMessagesForWorkspace(ctx, wsID)
			_ = q.DeleteThreadsForWorkspace(ctx, wsID)
			_ = q.DeleteSyncStateForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "mailpit-sync"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	mbEmail := "sync-" + id.String()[:8] + "@example.test"
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: mbEmail, AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	first := "<first-" + id.String() + "@outreach.test>"
	reply := "<reply-" + id.String() + "@outreach.test>"
	if err := smtp.SendMail(smtpAddr, nil, "person@example.test", []string{mbEmail}, []byte(
		"From: person@example.test\r\nTo: "+mbEmail+"\r\nSubject: Sync hello\r\nMessage-ID: "+first+"\r\n\r\nhi\r\n",
	)); err != nil {
		t.Fatal(err)
	}
	if err := smtp.SendMail(smtpAddr, nil, "person@example.test", []string{mbEmail}, []byte(
		"From: person@example.test\r\nTo: "+mbEmail+"\r\nSubject: Re: Sync hello\r\nMessage-ID: "+reply+"\r\nIn-Reply-To: "+first+"\r\n\r\nok\r\n",
	)); err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool, Mailpit{Base: api})
	deadline := time.Now().Add(5 * time.Second)
	var threads []Thread
	for {
		if _, err := store.SyncMailbox(ctx, wsID, mb.ID); err != nil {
			t.Fatal(err)
		}
		threads, err = store.ListThreads(ctx, wsID, mb.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(threads) == 1 && len(threads[0].Messages) >= 2 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("want one thread with reply, got %+v", threads)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestNormalizeSubject(t *testing.T) {
	t.Parallel()
	if got := normalizeSubject("Re: Re: Hello"); got != "hello" {
		t.Fatalf("%q", got)
	}
}
