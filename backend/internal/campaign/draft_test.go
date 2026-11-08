package campaign

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/contacts"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
)

func TestDraftDueStepWritesReviewDraftOnce(t *testing.T) {
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
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
			_ = q.DeleteEnrolmentsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignSendersForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignStepsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m63-draft"}); err != nil {
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
	camp, err := store.Create(ctx, wsID, CreateInput{Name: "AE intro", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}})
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
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	first, err := store.DraftDueStep(ctx, wsID, en.ID, 0, now)
	if err != nil || first.ID == uuid.Nil || first.To != "ada@acme.com" || first.Subject != "AE intro" || first.Body != "met at a meetup" || first.Status != "pending" {
		t.Fatalf("%+v %v", first, err)
	}
	again, err := store.DraftDueStep(ctx, wsID, en.ID, 0, now)
	if err != nil || again.ID != first.ID {
		t.Fatalf("idempotent %+v %v", again, err)
	}
	var n int
	var status string
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetEnrolment(ctx, db.GetEnrolmentParams{WorkspaceID: wsID, ID: en.ID})
		if err != nil {
			return err
		}
		status = row.Status
		drafts, err := q.ListReviewDrafts(ctx, wsID)
		if err != nil {
			return err
		}
		n = len(drafts)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if status != "waiting_review" || n != 1 {
		t.Fatalf("status %s drafts %d", status, n)
	}
}

func TestDraftDueStepSkipsOutsideWindow(t *testing.T) {
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
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
			_ = q.DeleteEnrolmentsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignSendersForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignStepsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m63-window"}); err != nil {
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
	ct, err := contacts.NewPostgres(pool).Create(ctx, wsID, contacts.CreateInput{
		Email: "ada@acme.com", Source: "typed", AddedBy: userID, Timezone: "Pacific/Honolulu",
	})
	if err != nil {
		t.Fatal(err)
	}
	packs, err := LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool, packs)
	camp, err := store.Create(ctx, wsID, CreateInput{Name: "AE intro", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}})
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
	sunday := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	d, err := store.DraftDueStep(ctx, wsID, en.ID, 0, sunday)
	if !errors.Is(err, ErrOutsideWindow) || d.ID != uuid.Nil {
		t.Fatalf("sunday %+v %v", d, err)
	}
	utcEvening := time.Date(2026, 11, 9, 18, 0, 0, 0, time.UTC)
	d, err = store.DraftDueStep(ctx, wsID, en.ID, 0, utcEvening)
	if !errors.Is(err, ErrOutsideWindow) {
		t.Fatalf("recipient evening %v", err)
	}
	open := time.Date(2026, 11, 9, 20, 0, 0, 0, time.UTC)
	d, err = store.DraftDueStep(ctx, wsID, en.ID, 0, open)
	if err != nil || d.ID == uuid.Nil {
		t.Fatalf("honolulu morning %+v %v", d, err)
	}
}
