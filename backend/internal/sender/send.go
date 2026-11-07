package sender

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/mime"
	"outreach/internal/pacing"
	"outreach/internal/suppression"
)

type Dialer interface {
	Dial(ctx context.Context, workspaceID, mailboxID uuid.UUID) (mailbox.Transport, string, error)
}

type SendInput struct {
	To      []string
	Subject string
	Text    string
	HTML    string
}

func (p *Postgres) WithDialer(d Dialer) *Postgres {
	p.dial = d
	return p
}

func (p *Postgres) Send(ctx context.Context, workspaceID, ledgerID uuid.UUID, in SendInput) (Entry, error) {
	if p.dial == nil {
		return Entry{}, fmt.Errorf("sender: dialer required")
	}
	if len(in.To) == 0 {
		return Entry{}, fmt.Errorf("sender: to required")
	}
	if p.block != nil {
		if err := p.block.Blocked(ctx, workspaceID, in.To); err != nil {
			if errors.Is(err, suppression.ErrSuppressed) {
				return Entry{}, ErrSuppressed
			}
			return Entry{}, err
		}
	}
	ent, err := p.Advance(ctx, workspaceID, ledgerID, pacing.StateSending)
	if err != nil {
		return Entry{}, err
	}
	tr, from, err := p.dial.Dial(ctx, workspaceID, ent.MailboxID)
	if err != nil {
		_, _ = p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
		return Entry{}, err
	}
	built, err := mime.Build(mime.Message{
		From: from, To: in.To, Subject: in.Subject, Text: in.Text, HTML: in.HTML,
		MessageID: ent.RFCMessageID,
	})
	if err != nil {
		_, _ = p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
		return Entry{}, err
	}
	if built.MessageID != ent.RFCMessageID {
		_, _ = p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
		return Entry{}, fmt.Errorf("sender: message-id changed")
	}
	if _, err := tr.Send(ctx, built); err != nil {
		if isSendTimeout(err) {
			_ = p.recordAttempt(ctx, workspaceID, ledgerID, "timeout", err)
			ent, advErr := p.Advance(ctx, workspaceID, ledgerID, pacing.StateUnknown)
			if advErr != nil {
				return Entry{}, advErr
			}
			if p.queue != nil {
				_ = p.queue.EnqueueReconcile(ctx, workspaceID, ledgerID, in)
			}
			return ent, err
		}
		_, _ = p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
		return Entry{}, err
	}
	appendErr := tr.AppendSent(ctx, built)
	outcome := "sent"
	if appendErr != nil {
		outcome = "append_failed"
	}
	_ = p.recordAttempt(ctx, workspaceID, ledgerID, outcome, appendErr)
	return p.Advance(ctx, workspaceID, ledgerID, pacing.StateSent)
}

func isSendTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var nerr net.Error
	return errors.As(err, &nerr) && nerr.Timeout()
}

func (p *Postgres) recordAttempt(ctx context.Context, workspaceID, ledgerID uuid.UUID, outcome string, detail error) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	resp := pgtype.Text{}
	if detail != nil {
		resp = pgtype.Text{String: detail.Error(), Valid: true}
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		n, err := q.CountAttempts(ctx, db.CountAttemptsParams{WorkspaceID: workspaceID, LedgerID: ledgerID})
		if err != nil {
			return err
		}
		return q.InsertSendAttempt(ctx, db.InsertSendAttemptParams{
			WorkspaceID:      workspaceID,
			ID:               id,
			LedgerID:         ledgerID,
			Attempt:          int32(n + 1),
			Outcome:          outcome,
			ProviderResponse: resp,
		})
	})
}
