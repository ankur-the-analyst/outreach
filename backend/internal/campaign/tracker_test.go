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
	"outreach/internal/tracking"
)

func TestTrackerStage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		enrol, draft, ledger, want string
	}{
		{"pending", "", "", "queued"},
		{"waiting_review", "pending", "", "drafted"},
		{"waiting_review", "pending", "held", "drafted"},
		{"active", "sent", "", "sent"},
		{"active", "approved", "sent", "sent"},
		{"replied", "sent", "sent", "replied"},
		{"stopped", "pending", "", "stopped"},
	}
	for _, tc := range cases {
		if got := TrackerStage(tc.enrol, tc.draft, tc.ledger); got != tc.want {
			t.Fatalf("%s %s %s: got %s want %s", tc.enrol, tc.draft, tc.ledger, got, tc.want)
		}
	}
	if got := PreferApplicationStage("replied", "interview"); got != "interview" {
		t.Fatalf("prefer %s", got)
	}
	if got := PreferApplicationStage("sent", "drafted"); got != "sent" {
		t.Fatalf("derived %s", got)
	}
}

func TestListTrackerMovesDraftedSentReplied(t *testing.T) {
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
			_ = q.DeleteMailEventsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m64-tracker"}); err != nil {
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
		Email: "ada@acme.com", FirstName: "Ada", Company: "Acme", Source: "typed", AddedBy: userID,
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
	camp, err := store.Create(ctx, wsID, CreateInput{Name: "AE intro", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}, ProfileID: &pid})
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
	draft, err := store.DraftDueStep(ctx, wsID, en.ID, 0, now)
	if err != nil {
		t.Fatal(err)
	}
	assertTracker(t, store, ctx, wsID, "drafted", "Ada", "Acme", "AE intro")

	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		_, err := q.UpdateDraftStatus(ctx, db.UpdateDraftStatusParams{
			WorkspaceID: wsID, ID: draft.ID, Status: "sent",
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	assertTracker(t, store, ctx, wsID, "sent", "Ada", "Acme", "AE intro")

	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		_, err := q.UpdateEnrolmentStatus(ctx, db.UpdateEnrolmentStatusParams{
			WorkspaceID: wsID, ID: en.ID, Status: "replied",
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	assertTracker(t, store, ctx, wsID, "replied", "Ada", "Acme", "AE intro")

	rows, err := store.ListTracker(ctx, wsID)
	if err != nil || len(rows) != 1 || rows[0].ApplicationID == nil {
		t.Fatalf("application %+v %v", rows, err)
	}
	if _, err := store.SetStage(ctx, wsID, *rows[0].ApplicationID, "interview"); err != nil {
		t.Fatal(err)
	}
	assertTracker(t, store, ctx, wsID, "interview", "Ada", "Acme", "AE intro")

	stamp := time.Date(2026, 11, 13, 12, 4, 0, 0, time.UTC)
	tracks := tracking.NewPostgres(pool, func() time.Time { return stamp })
	if err := tracks.Record(ctx, wsID, tracking.Event{
		EnrolmentID: en.ID, DraftID: draft.ID, Kind: tracking.KindOpen,
	}); err != nil {
		t.Fatal(err)
	}
	if err := tracks.Record(ctx, wsID, tracking.Event{
		EnrolmentID: en.ID, DraftID: draft.ID, Kind: tracking.KindClick, URL: "https://jobs.example/ada",
	}); err != nil {
		t.Fatal(err)
	}
	rows, err = store.ListTracker(ctx, wsID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("opened rows %+v %v", rows, err)
	}
	got := rows[0]
	if got.OpenedAt == nil || !got.OpenedAt.Equal(stamp) || got.ClickedAt == nil || !got.ClickedAt.Equal(stamp) {
		t.Fatalf("opened/clicked %+v", got)
	}
}

func assertTracker(t *testing.T, store *Postgres, ctx context.Context, wsID uuid.UUID, stage, contact, company, campaign string) {
	t.Helper()
	rows, err := store.ListTracker(ctx, wsID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows %+v %v", rows, err)
	}
	got := rows[0]
	if got.Stage != stage || got.Contact != contact || got.Company != company || got.Campaign != campaign {
		t.Fatalf("%+v want %s %s %s %s", got, stage, contact, company, campaign)
	}
}
