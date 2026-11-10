package sender

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"outreach/internal/db"
	"outreach/internal/pacing"
)

func (p *Postgres) Reconcile(ctx context.Context, workspaceID, ledgerID uuid.UUID, in SendInput) (Entry, error) {
	if p.dial == nil {
		return Entry{}, fmt.Errorf("sender: dialer required")
	}
	var row db.SendLedger
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		cur, err := q.GetLedger(ctx, db.GetLedgerParams{WorkspaceID: workspaceID, ID: ledgerID})
		if err != nil {
			return err
		}
		row = cur
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	st, err := pacing.ParseState(row.State)
	if err != nil {
		return Entry{}, err
	}
	ent := entryFrom(row.ID, row.MailboxID, row.SendingDomainID, row.DraftID, row.RfcMessageID, row.State, row.ReleaseAt)
	if st == pacing.StateSent {
		return ent, nil
	}
	if st != pacing.StateUnknown {
		return Entry{}, ErrConflict
	}

	tr, from, err := p.dial.Dial(ctx, workspaceID, row.MailboxID)
	if err != nil {
		return Entry{}, err
	}
	found, err := tr.FindSent(ctx, row.RfcMessageID)
	if err != nil {
		return Entry{}, err
	}
	if found {
		return p.Advance(ctx, workspaceID, ledgerID, pacing.StateSent)
	}

	age := time.Duration(0)
	if row.CreatedAt.Valid {
		age = p.now().Sub(row.CreatedAt.Time)
	}
	if p.unknown > 0 && age < p.unknown {
		return ent, nil
	}

	var retries int64
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		n, err := q.CountAttemptsByOutcome(ctx, db.CountAttemptsByOutcomeParams{
			WorkspaceID: workspaceID, LedgerID: ledgerID, Outcome: "retry",
		})
		retries = n
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	if retries >= 1 {
		return p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
	}
	if len(in.To) == 0 {
		return p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
	}

	in, err = p.withFiles(ctx, workspaceID, ledgerID, row.MailboxID, from, in)
	if err != nil {
		_, _ = p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
		return Entry{}, err
	}
	built, err := p.buildMIME(ctx, workspaceID, ledgerID, from, row.RfcMessageID, in)
	if err != nil {
		_, _ = p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
		return Entry{}, err
	}
	if _, err := tr.Send(ctx, built); err != nil {
		_ = p.recordAttempt(ctx, workspaceID, ledgerID, "retry", err)
		return p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
	}
	appendErr := tr.AppendSent(ctx, built)
	_ = p.recordAttempt(ctx, workspaceID, ledgerID, "retry", appendErr)
	return p.Advance(ctx, workspaceID, ledgerID, pacing.StateSent)
}
