package sender

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/campaign"
	"outreach/internal/contacts"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/mime"
	"outreach/internal/pacing"
	"outreach/internal/suppression"
)

func TestCommercialSendSetsRFC8058(t *testing.T) {
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
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteDraftsForWorkspace(ctx, wsID)
			_ = q.DeleteEnrolmentsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignSendersForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignStepsForWorkspace(ctx, wsID)
			_ = q.DeleteCampaignsForWorkspace(ctx, wsID)
			_ = q.DeleteSenderIdentityForWorkspace(ctx, wsID)
			_ = q.DeleteSuppressionForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m64-unsub"}); err != nil {
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
	packs, err := campaign.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	store := campaign.NewPostgres(pool, packs)
	if _, err := store.PutIdentity(ctx, wsID, campaign.Identity{
		LegalName: "Ada", ReplyAddress: "ada@acme.com", PostalAddress: "1 Main St",
	}); err != nil {
		t.Fatal(err)
	}
	camp, err := store.Create(ctx, wsID, campaign.CreateInput{
		Name: "Cold", Intent: "cold", MailboxIDs: []uuid.UUID{mb.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Start(ctx, wsID, camp.ID); err != nil {
		t.Fatal(err)
	}
	en, err := store.Enrol(ctx, wsID, camp.ID, campaign.EnrolInput{ContactID: ct.ID, Notes: "met at a meetup"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
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
	stub := &stubTransport{}
	led := NewPostgres(pool, time.Now, time.Minute).
		WithDialer(stubDialer{tr: stub}).
		WithUnsub("http://127.0.0.1:8080", keys)
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mb.ID, RFCMessageID: msgid, DraftID: draft.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	if _, err := led.Send(ctx, wsID, held.ID, SendInput{
		To: []string{ct.Email}, Subject: "Cold", Text: "hi",
	}); err != nil {
		t.Fatal(err)
	}
	if stub.last == nil || !bytes.Contains(stub.last.Bytes, []byte("List-Unsubscribe-Post: List-Unsubscribe=One-Click")) {
		t.Fatalf("%s", stub.last)
	}
	if !bytes.Contains(stub.last.Bytes, []byte("List-Unsubscribe: <http://127.0.0.1:8080/api/v1/unsubscribe/")) {
		t.Fatalf("%s", stub.last.Bytes)
	}
	want, err := suppression.OneClickURL("http://127.0.0.1:8080", keys, wsID, ct.Email)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(stub.last.Bytes, []byte("List-Unsubscribe: <"+want+">")) {
		t.Fatalf("token url missing:\n%s", stub.last.Bytes)
	}
	for _, part := range []string{"Ada", "1 Main St", "Unsubscribe: " + want} {
		if !bytes.Contains(stub.last.Bytes, []byte(part)) {
			t.Fatalf("footer missing %q:\n%s", part, stub.last.Bytes)
		}
	}
	if bytes.Contains(stub.last.Bytes, []byte(mime.OptOutLine)) {
		t.Fatal("commercial must not use the job opt-out line")
	}
}

func TestJobSendHasOptOutLine(t *testing.T) {
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
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m64-optout"}); err != nil {
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
	packs, err := campaign.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	store := campaign.NewPostgres(pool, packs)
	camp, err := store.Create(ctx, wsID, campaign.CreateInput{
		Name: "AE intro", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Start(ctx, wsID, camp.ID); err != nil {
		t.Fatal(err)
	}
	en, err := store.Enrol(ctx, wsID, camp.ID, campaign.EnrolInput{ContactID: ct.ID, Notes: "met at a meetup"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
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
	stub := &stubTransport{}
	led := NewPostgres(pool, time.Now, time.Minute).
		WithDialer(stubDialer{tr: stub}).
		WithUnsub("http://127.0.0.1:8080", keys)
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mb.ID, RFCMessageID: msgid, DraftID: draft.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	if _, err := led.Send(ctx, wsID, held.ID, SendInput{
		To: []string{ct.Email}, Subject: "AE intro", Text: "hi",
	}); err != nil {
		t.Fatal(err)
	}
	if stub.last == nil || !bytes.Contains(stub.last.Bytes, []byte(mime.OptOutLine)) {
		t.Fatalf("%s", stub.last)
	}
	if bytes.Contains(stub.last.Bytes, []byte("List-Unsubscribe:")) {
		t.Fatal("job mail must not set List-Unsubscribe")
	}
}
