package sender

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
)

func TestParallelRRNeverDoublePicks(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "rr-lock"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	store := mailbox.NewPostgres(pool, keys)
	a, err := store.Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "a@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "b@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{a.ID, b.ID}
	led := NewPostgres(pool, time.Now, time.Minute)

	type result struct {
		mb  uuid.UUID
		err error
	}
	ch := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			msgid, err := uuid.NewV7()
			if err != nil {
				ch <- result{err: err}
				return
			}
			ent, err := led.Reserve(ctx, wsID, ReserveInput{
				MailboxIDs:   ids,
				Strategy:     "round_robin",
				RFCMessageID: "<rr-" + msgid.String() + "@outreach.test>",
			})
			if err != nil {
				ch <- result{err: err}
				return
			}
			ch <- result{mb: ent.MailboxID}
		}(i)
	}
	wg.Wait()
	close(ch)
	picked := map[uuid.UUID]int{}
	for r := range ch {
		if r.err != nil {
			t.Fatalf("reserve: %v", r.err)
		}
		picked[r.mb]++
	}
	if len(picked) != 2 || picked[a.ID] != 1 || picked[b.ID] != 1 {
		t.Fatalf("double-pick %v", picked)
	}
}
