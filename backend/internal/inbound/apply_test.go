package inbound

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
	"outreach/internal/suppression"
	mailsync "outreach/internal/sync"
)

func TestUnsubscribeSuppressesAndDoesNotDraft(t *testing.T) {
	ctx, pool, wsID, mb, proc := setupInbound(t)
	src := fakeSource{items: []mailsync.Item{{
		RFCMessageID: "<unsub@x>", Subject: "stop", From: "lead@acme.test",
		To: []string{"box@example.test"}, Text: "Please unsubscribe me.", At: time.Now().UTC(),
	}}}
	store := mailsync.NewPostgres(pool, src).WithInbound(proc)
	if _, err := store.SyncMailbox(ctx, wsID, mb.ID); err != nil {
		t.Fatal(err)
	}
	if err := suppression.NewPostgres(pool, mustKeys(t)).Blocked(ctx, wsID, []string{"lead@acme.test"}); err == nil {
		t.Fatal("expected suppressed")
	}
	if n, err := countDrafts(ctx, pool, wsID); err != nil || n != 0 {
		t.Fatalf("drafts %d err %v", n, err)
	}
	row := mustMessage(t, ctx, pool, wsID, store, mb.ID)
	var class struct {
		Class      string `json:"class"`
		ReplyDraft bool   `json:"reply_draft"`
	}
	if err := json.Unmarshal(row.Classification, &class); err != nil {
		t.Fatal(err)
	}
	if class.Class != string(ClassUnsubscribe) || class.ReplyDraft {
		t.Fatalf("%+v", class)
	}
}

func TestInterestedCreatesPendingReplyDraft(t *testing.T) {
	ctx, pool, wsID, mb, proc := setupInbound(t)
	src := fakeSource{items: []mailsync.Item{{
		RFCMessageID: "<yes@x>", Subject: "re: role", From: "lead@acme.test",
		To: []string{"box@example.test"}, Text: "I'm interested in chatting.", At: time.Now().UTC(),
	}}}
	store := mailsync.NewPostgres(pool, src).WithInbound(proc)
	if _, err := store.SyncMailbox(ctx, wsID, mb.ID); err != nil {
		t.Fatal(err)
	}
	rows, err := listDrafts(ctx, pool, wsID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Kind != "reply" || rows[0].Status != "pending" || rows[0].ToAddr != "lead@acme.test" {
		t.Fatalf("%+v", rows)
	}
}

func TestReplyModeOffSkipsDraft(t *testing.T) {
	ctx, pool, wsID, mb, proc := setupInbound(t)
	err := mailbox.NewPostgres(pool, mustKeys(t)).SetReplyMode(ctx, wsID, mb.ID, "off")
	if err != nil {
		t.Fatal(err)
	}
	src := fakeSource{items: []mailsync.Item{{
		RFCMessageID: "<yes2@x>", Subject: "re: role", From: "lead@acme.test",
		To: []string{"box@example.test"}, Text: "I'm interested in chatting.", At: time.Now().UTC(),
	}}}
	store := mailsync.NewPostgres(pool, src).WithInbound(proc)
	if _, err := store.SyncMailbox(ctx, wsID, mb.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := countDrafts(ctx, pool, wsID); err != nil || n != 0 {
		t.Fatalf("drafts %d err %v", n, err)
	}
}

func setupInbound(t *testing.T) (context.Context, *db.Pool, uuid.UUID, mailbox.Mailbox, *Processor) {
	t.Helper()
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	keys := mustKeys(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
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
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
			_ = q.DeleteMessagesForWorkspace(ctx, wsID)
			_ = q.DeleteThreadsForWorkspace(ctx, wsID)
			_ = q.DeleteSyncStateForWorkspace(ctx, wsID)
			_ = q.DeleteSuppressionForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m43"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "box@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	proc := New(pool, suppression.NewPostgres(pool, keys), nil)
	return ctx, pool, wsID, mb, proc
}

func mustKeys(t *testing.T) *crypto.Keyring {
	t.Helper()
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

type fakeSource struct{ items []mailsync.Item }

func (f fakeSource) List(context.Context) ([]mailsync.Item, error) { return f.items, nil }

func countDrafts(ctx context.Context, pool *db.Pool, wsID uuid.UUID) (int64, error) {
	rows, err := listDrafts(ctx, pool, wsID)
	return int64(len(rows)), err
}

func listDrafts(ctx context.Context, pool *db.Pool, wsID uuid.UUID) ([]db.ListReviewDraftsRow, error) {
	var rows []db.ListReviewDraftsRow
	err := pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		var err error
		rows, err = q.ListReviewDrafts(ctx, wsID)
		return err
	})
	return rows, err
}

func mustMessage(t *testing.T, ctx context.Context, pool *db.Pool, wsID uuid.UUID, store *mailsync.Postgres, mailboxID uuid.UUID) db.GetMessageRow {
	t.Helper()
	threads, err := store.ListThreads(ctx, wsID, mailboxID)
	if err != nil || len(threads) != 1 {
		t.Fatalf("threads %+v err %v", threads, err)
	}
	var row db.GetMessageRow
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		m, err := q.GetMessage(ctx, db.GetMessageParams{WorkspaceID: wsID, ID: threads[0].Messages[0].ID})
		row = m
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return row
}
