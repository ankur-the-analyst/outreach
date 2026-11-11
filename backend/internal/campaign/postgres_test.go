package campaign

import (
	"context"
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

func TestCommercialStartRequiresIdentity(t *testing.T) {
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
			_ = q.DeleteEnrolmentsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignSendersForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignStepsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignsForWorkspace(ctx, wsID)
			_ = q.DeleteResumesForWorkspace(ctx, wsID)
			_ = q.DeleteProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteSenderIdentityForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m62"}); err != nil {
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
	pid := mustResume(t, ctx, pool, wsID)
	job, err := store.Create(ctx, wsID, CreateInput{Name: "AE", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}, ProfileID: &pid})
	if err != nil || len(job.Steps) != 3 || job.Commercial || job.Attach != "original" {
		t.Fatalf("%+v %v", job, err)
	}
	if _, err := store.Enrol(ctx, wsID, job.ID, EnrolInput{ContactID: ct.ID}); err != ErrContext {
		t.Fatalf("context %v", err)
	}
	en, err := store.Enrol(ctx, wsID, job.ID, EnrolInput{ContactID: ct.ID, Notes: "met at a meetup"})
	if err != nil || en.Status != "pending" {
		t.Fatalf("%+v %v", en, err)
	}
	if _, err := store.Start(ctx, wsID, job.ID); err != nil {
		t.Fatalf("job start %v", err)
	}
	bare, err := store.Create(ctx, wsID, CreateInput{Name: "Bare", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Start(ctx, wsID, bare.ID); err != ErrResume {
		t.Fatalf("want resume %v", err)
	}
	cold, err := store.Create(ctx, wsID, CreateInput{Name: "Cold", Intent: "cold", MailboxIDs: []uuid.UUID{mb.ID}})
	if err != nil || !cold.Commercial || !cold.ListUnsubscribe || cold.Attach != "none" {
		t.Fatalf("%+v %v", cold, err)
	}
	if _, err := store.Start(ctx, wsID, cold.ID); err != ErrIdentity {
		t.Fatalf("want identity %v", err)
	}
	if _, err := store.PutIdentity(ctx, wsID, Identity{
		LegalName: "Ada", ReplyAddress: "ada@acme.com", PostalAddress: "1 Main St",
	}); err != nil {
		t.Fatal(err)
	}
	started, err := store.Start(ctx, wsID, cold.ID)
	if err != nil || started.Status != "running" {
		t.Fatalf("%+v %v", started, err)
	}
}

func TestStartEmptyGroupCannotStartUntilMailboxAdded(t *testing.T) {
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
			_ = q.DeleteCampaignSendersForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignStepsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignsForWorkspace(ctx, wsID)
			_ = q.DeleteResumesForWorkspace(ctx, wsID)
			_ = q.DeleteProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteMailboxPoolMembersForWorkspace(ctx, wsID)
			_ = q.DeleteMailboxPoolsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m74"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	mbStore := mailbox.NewPostgres(pool, keys)
	empty, err := mbStore.CreatePool(ctx, wsID, "Empty group", nil)
	if err != nil {
		t.Fatal(err)
	}
	packs, err := LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool, packs)
	pid := mustResume(t, ctx, pool, wsID)
	camp, err := store.Create(ctx, wsID, CreateInput{Name: "Job", Intent: "job", PoolID: &empty.ID, ProfileID: &pid})
	if err != nil || len(camp.Senders) != 0 {
		t.Fatalf("%+v %v", camp, err)
	}
	if _, err := store.Start(ctx, wsID, camp.ID); err != ErrFrom {
		t.Fatalf("want from %v", err)
	}
	mb, err := mbStore.Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertMailboxPoolMember(ctx, db.InsertMailboxPoolMemberParams{
			WorkspaceID: wsID, PoolID: empty.ID, MailboxID: mb.ID,
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := store.Start(ctx, wsID, camp.ID)
	if err != nil || started.Status != "running" || len(started.Senders) != 1 {
		t.Fatalf("%+v %v", started, err)
	}
}
