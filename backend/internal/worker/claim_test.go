package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"outreach/internal/campaign"
	"outreach/internal/contacts"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/pacing"
)

func TestClaimDueUniqueEnrolmentStep(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
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
		_, _ = pool.Pgx().Exec(c, "delete from river_job where args->>'workspace_id' = $1", wsID.String())
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
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
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m63-claim"}); err != nil {
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
	later, err := contacts.NewPostgres(pool).Create(ctx, wsID, contacts.CreateInput{
		Email: "later@acme.com", Source: "typed", AddedBy: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	waiting, err := contacts.NewPostgres(pool).Create(ctx, wsID, contacts.CreateInput{
		Email: "wait@acme.com", Source: "typed", AddedBy: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	draftOnly, err := contacts.NewPostgres(pool).Create(ctx, wsID, contacts.CreateInput{
		Email: "draft@acme.com", Source: "typed", AddedBy: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	packs, err := campaign.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	store := campaign.NewPostgres(pool, packs)
	running, err := store.Create(ctx, wsID, campaign.CreateInput{Name: "AE", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}})
	if err != nil {
		t.Fatal(err)
	}
	due, err := store.Enrol(ctx, wsID, running.ID, campaign.EnrolInput{ContactID: ct.ID, Notes: "due now"})
	if err != nil {
		t.Fatal(err)
	}
	future, err := store.Enrol(ctx, wsID, running.ID, campaign.EnrolInput{ContactID: later.ID, Notes: "not yet"})
	if err != nil {
		t.Fatal(err)
	}
	waitEn, err := store.Enrol(ctx, wsID, running.ID, campaign.EnrolInput{ContactID: waiting.ID, Notes: "on wait"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Start(ctx, wsID, running.ID); err != nil {
		t.Fatal(err)
	}
	paused, err := store.Create(ctx, wsID, campaign.CreateInput{Name: "Draft", Intent: "job", MailboxIDs: []uuid.UUID{mb.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Enrol(ctx, wsID, paused.ID, campaign.EnrolInput{ContactID: draftOnly.ID, Notes: "still draft"}); err != nil {
		t.Fatal(err)
	}
	err = pool.ExecTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, _ *db.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `update enrolments set next_action_at = $1 where id = $2`,
			time.Now().UTC().Add(24*time.Hour), future.ID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `update enrolments set current_step = 1 where id = $1`, waitEn.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewInsertClient(pool)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	first, err := ClaimDue(ctx, pool, client, wsID, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Inserted != 1 || first.Skipped != 0 {
		t.Fatalf("first claim %+v", first)
	}
	again, err := ClaimDue(ctx, pool, client, wsID, now)
	if err != nil {
		t.Fatal(err)
	}
	if again.Inserted != 0 || again.Skipped != 1 {
		t.Fatalf("second claim %+v", again)
	}
	var n int
	if err := pool.Pgx().QueryRow(ctx, `
		select count(*) from river_job
		where kind = 'campaign.step' and args->>'workspace_id' = $1`, wsID.String()).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("jobs %d want 1", n)
	}
	var enrolmentID string
	var step int
	if err := pool.Pgx().QueryRow(ctx, `
		select args->>'enrolment_id', (args->>'step_position')::int
		from river_job
		where kind = 'campaign.step' and args->>'workspace_id' = $1`, wsID.String()).Scan(&enrolmentID, &step); err != nil {
		t.Fatal(err)
	}
	if enrolmentID != due.ID.String() || step != 0 {
		t.Fatalf("claimed %s step %d want %s 0", enrolmentID, step, due.ID)
	}

	if err := pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.SetSendsPaused(ctx, db.SetSendsPausedParams{ID: wsID, SendsPaused: true})
	}); err != nil {
		t.Fatal(err)
	}
	pausedClaim, err := ClaimDue(ctx, pool, client, wsID, now)
	if err != nil {
		t.Fatal(err)
	}
	if pausedClaim.Inserted != 0 || pausedClaim.Skipped != 0 {
		t.Fatalf("kill switch %+v", pausedClaim)
	}
}
