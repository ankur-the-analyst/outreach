package campaign

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/contacts"
	"outreach/internal/crypto"
	"outreach/internal/cv"
	"outreach/internal/db"
	"outreach/internal/jobsjd"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
	"outreach/internal/sender"
	"outreach/internal/storage"
)

func TestFilesOriginalResume(t *testing.T) {
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
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
			_ = q.DeleteEnrolmentsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignSendersForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignStepsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignsForWorkspace(ctx, wsID)
			_ = q.DeleteResumesForWorkspace(ctx, wsID)
			_ = q.DeleteProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteContactsForWorkspace(ctx, wsID)
			_ = q.DeleteCompaniesForWorkspace(ctx, wsID)
			_ = q.DeleteMailboxesForWorkspace(ctx, wsID)
			_ = q.DeleteSecretsForWorkspace(ctx, wsID)
			_ = q.DeleteWarmupProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteSendingDomainsForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m93-attach"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	blobs := storage.NewMemory()
	pdf := []byte("%PDF-1.4 original\n%%EOF\n")
	jj := jobsjd.NewPostgres(pool, blobs, nil)
	prof, err := jj.CreateProfile(ctx, wsID, "Ada", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jj.UploadResume(ctx, wsID, prof.ID, "Ada.pdf", "application/pdf", "Ada Lovelace", pdf); err != nil {
		t.Fatal(err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	ct, err := contacts.NewPostgres(pool).Create(ctx, wsID, contacts.CreateInput{
		Email: "ada@acme.com", Source: "typed", AddedBy: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	packs, err := LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool, packs)
	camp, err := store.Create(ctx, wsID, CreateInput{
		Name: "AE", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}, ProfileID: &prof.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	en, err := store.Enrol(ctx, wsID, camp.ID, EnrolInput{ContactID: ct.ID, Notes: "met at a meetup"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Start(ctx, wsID, camp.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	draft, err := store.DraftDueStep(ctx, wsID, en.ID, 0, now)
	if err != nil {
		t.Fatal(err)
	}
	var msgid string
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetDraft(ctx, db.GetDraftParams{WorkspaceID: wsID, ID: draft.ID})
		if err != nil {
			return err
		}
		msgid = row.RfcMessageID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	held, err := sender.NewPostgres(pool, time.Now, time.Minute).Hold(ctx, wsID, sender.HoldInput{
		MailboxID: mb.ID, RFCMessageID: msgid, DraftID: draft.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := cv.NewPostgres(pool, blobs).Files(ctx, wsID, held.ID, mb.ID, mb.Email)
	if err != nil || len(files) != 1 || files[0].Name != "Ada.pdf" || !bytes.Equal(files[0].Bytes, pdf) {
		t.Fatalf("%+v %v", files, err)
	}
	missing, err := cv.NewPostgres(pool, storage.NewMemory()).Files(ctx, wsID, held.ID, mb.ID, mb.Email)
	if err == nil || missing != nil {
		t.Fatalf("missing blob %+v %v", missing, err)
	}
}
