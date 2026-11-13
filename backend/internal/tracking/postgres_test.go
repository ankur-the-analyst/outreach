package tracking

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/contacts"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
)

func TestRecordRespectsRLSAndKeepsFirstOpen(t *testing.T) {
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
	wsA, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	wsB, err := uuid.NewV7()
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
		for _, id := range []uuid.UUID{wsA, wsB} {
			_ = pool.WithGUC(c, db.GUC{WorkspaceID: id}, func(ctx context.Context, q *db.Queries) error {
				_ = q.DeleteMailEventsForWorkspace(ctx, id)
				_ = q.DeleteDraftsForWorkspace(ctx, id)
				_ = q.DeleteEnrolmentsForWorkspace(ctx, id)
				_ = q.DeleteCampaignsForWorkspace(ctx, id)
				_ = q.DeleteContactsForWorkspace(ctx, id)
				_ = q.DeleteMailboxesForWorkspace(ctx, id)
				_ = q.DeleteSecretsForWorkspace(ctx, id)
				_ = q.DeleteWarmupProfilesForWorkspace(ctx, id)
				_ = q.DeleteSendingDomainsForWorkspace(ctx, id)
				_ = q.DeleteWorkspace(ctx, id)
				return nil
			})
		}
		pool.Close()
	})
	for _, id := range []uuid.UUID{wsA, wsB} {
		name := "m111-a"
		if id == wsB {
			name = "m111-b"
		}
		err = pool.WithGUC(ctx, db.GUC{WorkspaceID: id}, func(ctx context.Context, q *db.Queries) error {
			if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: id, Name: name}); err != nil {
				return err
			}
			return pacing.SeedPresets(ctx, q, id)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsA, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	ct, err := contacts.NewPostgres(pool).Create(ctx, wsA, contacts.CreateInput{
		Email: "ada@acme.com", Source: "typed", AddedBy: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	campID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	enID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	draftID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsA}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.InsertCampaign(ctx, db.InsertCampaignParams{
			WorkspaceID: wsA, ID: campID, Name: "Cold", Intent: "cold",
			SenderStrategy: "round_robin", Status: "draft", NewContactsPerDay: 5,
			UndoHoldS: 300, Attach: "none", TrackOpens: true, TrackClicks: true,
		}); err != nil {
			return err
		}
		if _, err := q.InsertEnrolment(ctx, db.InsertEnrolmentParams{
			WorkspaceID: wsA, ID: enID, CampaignID: campID, ContactID: ct.ID,
			Intent: "cold", Context: []byte(`{}`), Status: "waiting_review",
		}); err != nil {
			return err
		}
		_, err := q.InsertDraft(ctx, db.InsertDraftParams{
			WorkspaceID: wsA, ID: draftID, EnrolmentID: &enID, MailboxID: mb.ID,
			ToAddr: ct.Email, RfcMessageID: "<m111@outreach.test>", Kind: "outbound",
			Subject: "Hi", BodyHtml: "<p>Hi</p>", BodyText: "Hi", FactsUsed: []byte("[]"),
			Status: "pending", CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	now := t0
	store := NewPostgres(pool, func() time.Time { return now })
	if err := store.Record(ctx, wsA, Event{EnrolmentID: enID, DraftID: draftID, Kind: KindOpen}); err != nil {
		t.Fatal(err)
	}
	now = t1
	if err := store.Record(ctx, wsA, Event{EnrolmentID: enID, DraftID: draftID, Kind: KindOpen}); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(ctx, wsA, Event{
		EnrolmentID: enID, DraftID: draftID, Kind: KindClick, URL: "https://jobs.example/ada",
	}); err != nil {
		t.Fatal(err)
	}
	fromA, err := store.List(ctx, wsA, enID)
	if err != nil || len(fromA) != 3 {
		t.Fatalf("wsA %+v %v", fromA, err)
	}
	fromB, err := store.List(ctx, wsB, enID)
	if err != nil || len(fromB) != 0 {
		t.Fatalf("wsB leaked %+v %v", fromB, err)
	}
	opened, clicked, err := store.Times(ctx, wsA, enID)
	if err != nil || opened == nil || clicked == nil || !opened.Equal(t0) || !clicked.Equal(t1) {
		t.Fatalf("times %v %v %v", opened, clicked, err)
	}
}
