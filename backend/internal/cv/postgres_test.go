package cv

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/jobsjd"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
	"outreach/internal/storage"
)

func TestStampTwoMailboxesDifferentFrom(t *testing.T) {
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
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteCvRendersForWorkspace(ctx, wsID)
			_ = q.DeleteCvDocumentsForWorkspace(ctx, wsID)
			_ = q.DeleteMailboxesForWorkspace(ctx, wsID)
			_ = q.DeleteSecretsForWorkspace(ctx, wsID)
			_ = q.DeleteWarmupProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteSendingDomainsForWorkspace(ctx, wsID)
			_ = q.DeleteProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m82-cv"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	mbs := mailbox.NewPostgres(pool, keys)
	mb1, err := mbs.Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "a@outreach.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	mb2, err := mbs.Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "b@outreach.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	prof, err := jobsjd.NewPostgres(pool, storage.NewMemory(), nil).CreateProfile(ctx, wsID, "Ada", "")
	if err != nil {
		t.Fatal(err)
	}
	blobs := storage.NewMemory()
	store := NewPostgres(pool, blobs)
	if _, err := store.Save(ctx, wsID, prof.ID, Spec{Name: "Ada Lovelace", Headline: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	r1, err := store.StampFor(ctx, wsID, prof.ID, mb1.ID)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := store.StampFor(ctx, wsID, prof.ID, mb2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r1.ID == r2.ID {
		t.Fatal("want two renders")
	}
	if r1.FromEmail != "a@outreach.test" || r2.FromEmail != "b@outreach.test" {
		t.Fatalf("%q %q", r1.FromEmail, r2.FromEmail)
	}
	again, err := store.StampFor(ctx, wsID, prof.ID, mb1.ID)
	if err != nil || again.ID != r1.ID {
		t.Fatalf("%+v %v", again, err)
	}
	b1, err := blobs.Get(ctx, r1.FileKey)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := blobs.Get(ctx, r2.FileKey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b1, []byte("a@outreach.test")) || bytes.Contains(b1, []byte("b@outreach.test")) {
		t.Fatalf("pdf 1:\n%s", b1)
	}
	if !bytes.Contains(b2, []byte("b@outreach.test")) {
		t.Fatalf("pdf 2:\n%s", b2)
	}
}
