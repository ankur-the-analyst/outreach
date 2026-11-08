package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
	"outreach/internal/mime"
	"outreach/internal/suppression"
)

type Classifier interface {
	Enqueue(ctx context.Context, workspaceID, messageID uuid.UUID) error
}

type Processor struct {
	pool  *db.Pool
	supp  *suppression.Postgres
	queue Classifier
}

func New(pool *db.Pool, supp *suppression.Postgres, queue Classifier) *Processor {
	return &Processor{pool: pool, supp: supp, queue: queue}
}

func (p *Processor) Handle(ctx context.Context, workspaceID, mailboxID, messageID uuid.UUID, from, subject, body string) error {
	class, ok := Rules(from, subject, body)
	if !ok {
		if p.queue != nil {
			return p.queue.Enqueue(ctx, workspaceID, messageID)
		}
		class = ClassOther
	}
	conf := 1.0
	src := "rules"
	if !ok {
		conf = 0
		src = "classify"
	}
	return p.Apply(ctx, workspaceID, messageID, from, subject, class, conf, src)
}

func (p *Processor) Apply(ctx context.Context, workspaceID, messageID uuid.UUID, from, subject string, class Class, confidence float64, source string) error {
	eff := Decide(class, confidence)
	payload, err := json.Marshal(map[string]any{
		"class":       class,
		"confidence":  confidence,
		"source":      source,
		"suppress":    eff.Suppress,
		"reply_draft": eff.ReplyDraft,
		"hold":        eff.Hold,
		"stop":        eff.Stop,
	})
	if err != nil {
		return err
	}
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.UpdateMessageClassification(ctx, db.UpdateMessageClassificationParams{
			WorkspaceID:    workspaceID,
			ID:             messageID,
			Classification: payload,
			Kind:           pgtype.Text{String: string(class), Valid: true},
		})
	})
	if err != nil {
		return err
	}
	if eff.Suppress && p.supp != nil && from != "" && class != ClassDSN {
		if _, err := p.supp.Add(ctx, workspaceID, suppression.KindEmail, from, string(class)); err != nil {
			return fmt.Errorf("inbound: suppress: %w", err)
		}
	}
	if from != "" && eff.Enrolment != "" {
		if err := p.setEnrolments(ctx, workspaceID, from, eff.Enrolment); err != nil {
			return err
		}
	}
	if !eff.ReplyDraft || from == "" {
		return nil
	}
	return p.insertReply(ctx, workspaceID, messageID, from, subject)
}

func (p *Processor) setEnrolments(ctx context.Context, workspaceID uuid.UUID, from, status string) error {
	email := strings.ToLower(strings.TrimSpace(from))
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListOpenEnrolmentsByEmail(ctx, db.ListOpenEnrolmentsByEmailParams{
			WorkspaceID: workspaceID, Email: email,
		})
		if err != nil {
			return fmt.Errorf("inbound: enrolments: %w", err)
		}
		for _, row := range rows {
			if _, err := q.UpdateEnrolmentStatus(ctx, db.UpdateEnrolmentStatusParams{
				WorkspaceID: workspaceID, ID: row.ID, Status: status,
			}); err != nil {
				return fmt.Errorf("inbound: enrolment status: %w", err)
			}
		}
		return nil
	})
}

func (p *Processor) ClassifyStored(ctx context.Context, workspaceID, messageID uuid.UUID) error {
	var from, subject, body string
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMessage(ctx, db.GetMessageParams{WorkspaceID: workspaceID, ID: messageID})
		if err != nil {
			return err
		}
		if len(row.Classification) > 0 {
			return nil
		}
		from = textOr(row.FromAddr)
		subject = textOr(row.Subject)
		body = textOr(row.BodyText)
		return nil
	})
	if err != nil {
		return err
	}
	if from == "" && subject == "" && body == "" {
		return nil
	}
	class, ok := Rules(from, subject, body)
	conf := 1.0
	src := "rules"
	if !ok {
		class = ClassOther
		conf = 0
		src = "classify"
	}
	return p.Apply(ctx, workspaceID, messageID, from, subject, class, conf, src)
}

func (p *Processor) insertReply(ctx context.Context, workspaceID, messageID uuid.UUID, from, subject string) error {
	var mailboxID uuid.UUID
	var mode string
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMessage(ctx, db.GetMessageParams{WorkspaceID: workspaceID, ID: messageID})
		if err != nil {
			return err
		}
		mailboxID = row.MailboxID
		mode, err = q.GetMailboxReplyMode(ctx, db.GetMailboxReplyModeParams{WorkspaceID: workspaceID, ID: mailboxID})
		return err
	})
	if err != nil {
		return fmt.Errorf("inbound: reply mode: %w", err)
	}
	if mode != "draft" {
		return nil
	}
	msgid, err := mime.NewMessageID("outreach.test")
	if err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	to := strings.ToLower(strings.TrimSpace(from))
	subj := replySubject(subject)
	body := "Thanks for writing — this reply is waiting in review."
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		_, err := q.InsertDraft(ctx, db.InsertDraftParams{
			WorkspaceID:  workspaceID,
			ID:           id,
			MailboxID:    mailboxID,
			ToAddr:       to,
			RfcMessageID: msgid,
			Kind:         "reply",
			Subject:      subj,
			BodyHtml:     "",
			BodyText:     body,
			FactsUsed:    []byte("[]"),
			Status:       "pending",
			CreatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		})
		return err
	})
}

func replySubject(subject string) string {
	s := strings.TrimSpace(subject)
	if s == "" {
		s = "(no subject)"
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "re:") {
		return s
	}
	return "Re: " + s
}

func textOr(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
