package sender

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/mime"
	"outreach/internal/pacing"
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
		_, _ = p.Advance(ctx, workspaceID, ledgerID, pacing.StateFailed)
		return Entry{}, err
	}
	appendErr := tr.AppendSent(ctx, built)
	_ = p.recordAttempt(ctx, workspaceID, ledgerID, appendErr)
	return p.Advance(ctx, workspaceID, ledgerID, pacing.StateSent)
}

func (p *Postgres) recordAttempt(ctx context.Context, workspaceID, ledgerID uuid.UUID, appendErr error) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	outcome := "sent"
	resp := pgtype.Text{}
	if appendErr != nil {
		outcome = "append_failed"
		resp = pgtype.Text{String: appendErr.Error(), Valid: true}
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertSendAttempt(ctx, db.InsertSendAttemptParams{
			WorkspaceID:      workspaceID,
			ID:               id,
			LedgerID:         ledgerID,
			Attempt:          1,
			Outcome:          outcome,
			ProviderResponse: resp,
		})
	})
}
