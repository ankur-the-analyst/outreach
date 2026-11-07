package inbound

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
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
	return p.Apply(ctx, workspaceID, messageID, from, class, conf, src)
}

func (p *Processor) Apply(ctx context.Context, workspaceID, messageID uuid.UUID, from string, class Class, confidence float64, source string) error {
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
	return nil
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
	return p.Apply(ctx, workspaceID, messageID, from, class, conf, src)
}

func textOr(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
