package compose

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/ai"
	"outreach/internal/contacts"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
	"outreach/internal/sender"
	"outreach/internal/tracking"
)

func TestApproveHoldAndUndoCancels(t *testing.T) {
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
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m37"}); err != nil {
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
	led := sender.NewPostgres(pool, time.Now, time.Minute)
	store := NewPostgres(pool, led)
	d, err := store.Create(ctx, wsID, CreateInput{
		MailboxID: mb.ID, To: "sink@outreach.test", Subject: "hello", Text: "body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "pending" || d.RFCMessageID == "" {
		t.Fatalf("%+v", d)
	}
	held, err := store.Approve(ctx, wsID, userID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if held.Status != "approved" || held.LedgerState != pacing.StateHeld || held.LedgerID == nil {
		t.Fatalf("%+v", held)
	}
	used, err := led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 1 {
		t.Fatalf("usage %d %v", used, err)
	}
	undone, err := store.Undo(ctx, wsID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if undone.Status != "cancelled" {
		t.Fatalf("%s", undone.Status)
	}
	used, err = led.Usage(ctx, wsID, mb.ID)
	if err != nil || used != 0 {
		t.Fatalf("usage after undo %d %v", used, err)
	}
	_, err = led.Advance(ctx, wsID, *held.LedgerID, pacing.StateReserved)
	if !errors.Is(err, sender.ErrConflict) {
		t.Fatalf("send path after undo: %v", err)
	}
}

func TestRejectAndRegeneratePending(t *testing.T) {
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
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m37r"}); err != nil {
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
	store := NewPostgres(pool, sender.NewPostgres(pool, time.Now, time.Minute)).
		WithWriter(&ai.Fake{Text: "Open to a short chat?"})
	d, err := store.Create(ctx, wsID, CreateInput{
		MailboxID: mb.ID, To: "sink@outreach.test", Subject: "hello", Text: "old body",
	})
	if err != nil {
		t.Fatal(err)
	}
	patched, err := store.Patch(ctx, wsID, d.ID, PatchInput{Subject: "hello edited", Text: "operator body"})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Subject != "hello edited" || patched.BodyText != "operator body" {
		t.Fatalf("%+v", patched)
	}
	rewritten, err := store.Regenerate(ctx, wsID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rewritten.Draft.BodyText != "Open to a short chat?" || rewritten.Draft.Status != "pending" {
		t.Fatalf("%+v", rewritten)
	}
	dropped, err := store.Reject(ctx, wsID, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dropped.Status != "rejected" {
		t.Fatalf("%+v", dropped)
	}
	listed, err := store.List(ctx, wsID, nil, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Drafts) != 0 {
		t.Fatalf("rejected still listed: %+v", listed)
	}
	_, err = store.Reject(ctx, wsID, d.ID)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("second reject: %v", err)
	}
}

func TestApproveRejectsLeftoverAndPatchKeepsHTML(t *testing.T) {
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
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m103-html"}); err != nil {
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
	store := NewPostgres(pool, sender.NewPostgres(pool, time.Now, time.Minute))
	d, err := store.Create(ctx, wsID, CreateInput{
		MailboxID: mb.ID, To: "sink@outreach.test", Subject: "hello",
		Text: "Hi Ada", HTML: "<p>Hi {{contact.first_name}}</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Approve(ctx, wsID, userID, d.ID); !errors.Is(err, ErrLeftover) {
		t.Fatalf("want leftover %v", err)
	}
	patched, err := store.Patch(ctx, wsID, d.ID, PatchInput{Subject: "hello", Text: "Hi Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if patched.BodyHTML != "<p>Hi {{contact.first_name}}</p>" {
		t.Fatalf("html wiped %+v", patched)
	}
	fixed, err := store.Patch(ctx, wsID, d.ID, PatchInput{
		Subject: "hello", Text: "Hi Ada", HTML: "<p>Hi Ada</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	held, err := store.Approve(ctx, wsID, userID, fixed.ID)
	if err != nil || held.Status != "approved" {
		t.Fatalf("%+v %v", held, err)
	}
}

func TestListShowsOpenedClicked(t *testing.T) {
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
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
			_ = q.DeleteEnrolmentsForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m11-review"}); err != nil {
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
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.InsertCampaign(ctx, db.InsertCampaignParams{
			WorkspaceID: wsID, ID: campID, Name: "Cold", Intent: "cold",
			SenderStrategy: "round_robin", Status: "draft", NewContactsPerDay: 5,
			UndoHoldS: 300, Attach: "none", TrackOpens: true, TrackClicks: true,
		}); err != nil {
			return err
		}
		if _, err := q.InsertEnrolment(ctx, db.InsertEnrolmentParams{
			WorkspaceID: wsID, ID: enID, CampaignID: campID, ContactID: ct.ID,
			Intent: "cold", Context: []byte(`{}`), Status: "waiting_review",
		}); err != nil {
			return err
		}
		_, err := q.InsertDraft(ctx, db.InsertDraftParams{
			WorkspaceID: wsID, ID: draftID, EnrolmentID: &enID, MailboxID: mb.ID,
			ToAddr: ct.Email, RfcMessageID: "<m11-review@outreach.test>", Kind: "outbound",
			Subject: "Hi", BodyHtml: "<p>Hi</p>", BodyText: "Hi", FactsUsed: []byte("[]"),
			Status: "pending", CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 11, 13, 13, 0, 0, 0, time.UTC)
	tracks := tracking.NewPostgres(pool, func() time.Time { return stamp })
	if err := tracks.Record(ctx, wsID, tracking.Event{
		EnrolmentID: enID, DraftID: draftID, Kind: tracking.KindOpen,
	}); err != nil {
		t.Fatal(err)
	}
	if err := tracks.Record(ctx, wsID, tracking.Event{
		EnrolmentID: enID, DraftID: draftID, Kind: tracking.KindClick, URL: "https://jobs.example/ada",
	}); err != nil {
		t.Fatal(err)
	}
	listed, err := NewPostgres(pool, sender.NewPostgres(pool, time.Now, time.Minute)).List(ctx, wsID, nil, 50)
	if err != nil || len(listed.Drafts) != 1 {
		t.Fatalf("list %+v %v", listed, err)
	}
	got := listed.Drafts[0]
	if got.OpenedAt == nil || !got.OpenedAt.Equal(stamp) || got.ClickedAt == nil || !got.ClickedAt.Equal(stamp) {
		t.Fatalf("opened/clicked %+v", got)
	}
}
