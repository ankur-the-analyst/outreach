package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
)

type Thread struct {
	ID            uuid.UUID  `json:"id"`
	MailboxID     uuid.UUID  `json:"mailbox_id"`
	Subject       string     `json:"subject"`
	Unread        bool       `json:"unread"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	Messages      []Message  `json:"messages,omitempty"`
}

type Message struct {
	ID           uuid.UUID `json:"id"`
	Direction    string    `json:"direction"`
	RFCMessageID string    `json:"rfc_message_id"`
	From         string    `json:"from"`
	To           []string  `json:"to"`
	Subject      string    `json:"subject"`
	BodyText     string    `json:"body_text"`
}

type Postgres struct {
	pool   *db.Pool
	source Source
}

func NewPostgres(pool *db.Pool, source Source) *Postgres {
	return &Postgres{pool: pool, source: source}
}

func (p *Postgres) Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error) {
	var role string
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID, UserID: userID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMembership(ctx, db.GetMembershipParams{WorkspaceID: workspaceID, UserID: userID})
		if err != nil {
			return err
		}
		role = row.Role
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("not found")
	}
	return role, err
}

func (p *Postgres) SyncMailbox(ctx context.Context, workspaceID, mailboxID uuid.UUID) (int, error) {
	if p.source == nil {
		return 0, fmt.Errorf("sync: source required")
	}
	var email string
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		mb, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: mailboxID})
		if err != nil {
			return err
		}
		email = strings.ToLower(mb.Email)
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("not found")
	}
	if err != nil {
		return 0, err
	}
	items, err := p.source.List(ctx)
	if err != nil {
		_ = p.mark(ctx, workspaceID, mailboxID, err)
		return 0, err
	}
	n := 0
	for _, it := range items {
		if !forMailbox(it, email) {
			continue
		}
		if err := p.ingest(ctx, workspaceID, mailboxID, email, it); err != nil {
			_ = p.mark(ctx, workspaceID, mailboxID, err)
			return n, err
		}
		n++
	}
	if err := p.mark(ctx, workspaceID, mailboxID, nil); err != nil {
		return n, err
	}
	return n, nil
}

func (p *Postgres) ListThreads(ctx context.Context, workspaceID, mailboxID uuid.UUID) ([]Thread, error) {
	var out []Thread
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: mailboxID}); err != nil {
			return err
		}
		rows, err := q.ListThreadsForMailbox(ctx, db.ListThreadsForMailboxParams{WorkspaceID: workspaceID, MailboxID: mailboxID})
		if err != nil {
			return err
		}
		out = make([]Thread, 0, len(rows))
		for _, row := range rows {
			th := Thread{ID: row.ID, MailboxID: row.MailboxID, Subject: textVal(row.Subject), Unread: row.Unread}
			if row.LastMessageAt.Valid {
				t := row.LastMessageAt.Time.UTC()
				th.LastMessageAt = &t
			}
			msgs, err := q.ListMessagesForThread(ctx, db.ListMessagesForThreadParams{WorkspaceID: workspaceID, ThreadID: row.ID})
			if err != nil {
				return err
			}
			for _, m := range msgs {
				th.Messages = append(th.Messages, Message{
					ID: m.ID, Direction: m.Direction, RFCMessageID: m.RfcMessageID,
					From: textVal(m.FromAddr), To: m.ToAddrs, Subject: textVal(m.Subject), BodyText: textVal(m.BodyText),
				})
			}
			out = append(out, th)
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("not found")
	}
	return out, err
}

func (p *Postgres) ingest(ctx context.Context, workspaceID, mailboxID uuid.UUID, mailboxEmail string, it Item) error {
	if it.RFCMessageID == "" {
		return nil
	}
	return p.pool.ExecTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries, _ pgx.Tx) error {
		if err := q.AdvisoryLockMailbox(ctx, mailboxID.String()); err != nil {
			return err
		}
		threadID, err := p.resolveThread(ctx, q, workspaceID, mailboxID, mailboxEmail, it)
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		dir := "in"
		if strings.EqualFold(it.From, mailboxEmail) {
			dir = "out"
		}
		at := it.At
		if at.IsZero() {
			at = time.Now().UTC()
		}
		ts := pgtype.Timestamptz{Time: at, Valid: true}
		from := pgtype.Text{String: it.From, Valid: it.From != ""}
		subj := pgtype.Text{String: it.Subject, Valid: it.Subject != ""}
		body := pgtype.Text{String: it.Text, Valid: it.Text != ""}
		inReply := pgtype.Text{String: it.InReplyTo, Valid: it.InReplyTo != ""}
		if err := q.InsertMessage(ctx, db.InsertMessageParams{
			WorkspaceID:  workspaceID,
			ID:           id,
			MailboxID:    mailboxID,
			ThreadID:     threadID,
			Direction:    dir,
			RfcMessageID: it.RFCMessageID,
			InReplyTo:    inReply,
			FromAddr:     from,
			ToAddrs:      it.To,
			Subject:      subj,
			BodyText:     body,
			SentAt:       ts,
			ReceivedAt:   ts,
		}); err != nil {
			return err
		}
		return q.TouchThread(ctx, db.TouchThreadParams{
			WorkspaceID:   workspaceID,
			ID:            threadID,
			MailboxID:     mailboxID,
			LastMessageAt: ts,
			Unread:        dir == "in",
		})
	})
}

func (p *Postgres) resolveThread(ctx context.Context, q *db.Queries, workspaceID, mailboxID uuid.UUID, mailboxEmail string, it Item) (uuid.UUID, error) {
	if it.InReplyTo != "" {
		row, err := q.GetThreadByRFC(ctx, db.GetThreadByRFCParams{
			WorkspaceID: workspaceID, MailboxID: mailboxID, RfcMessageID: it.InReplyTo,
		})
		if err == nil {
			return row.ID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, err
		}
	}
	subj := normalizeSubject(it.Subject)
	if subj != "" {
		row, err := q.GetThreadBySubject(ctx, db.GetThreadBySubjectParams{
			WorkspaceID: workspaceID, MailboxID: mailboxID,
			Subject: pgtype.Text{String: subj, Valid: true},
		})
		if err == nil {
			return row.ID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, err
		}
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	at := it.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err = q.InsertThread(ctx, db.InsertThreadParams{
		WorkspaceID:   workspaceID,
		ID:            id,
		MailboxID:     mailboxID,
		Subject:       pgtype.Text{String: it.Subject, Valid: it.Subject != ""},
		LastMessageAt: pgtype.Timestamptz{Time: at, Valid: true},
		Unread:        !strings.EqualFold(it.From, mailboxEmail),
	})
	return id, err
}

func (p *Postgres) mark(ctx context.Context, workspaceID, mailboxID uuid.UUID, syncErr error) error {
	errText := pgtype.Text{}
	if syncErr != nil {
		errText = pgtype.Text{String: syncErr.Error(), Valid: true}
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.UpsertSyncState(ctx, db.UpsertSyncStateParams{
			WorkspaceID:  workspaceID,
			MailboxID:    mailboxID,
			Folder:       "INBOX",
			LastSyncedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			LastError:    errText,
		})
	})
}

func forMailbox(it Item, email string) bool {
	if strings.EqualFold(it.From, email) {
		return true
	}
	for _, to := range it.To {
		if strings.EqualFold(to, email) {
			return true
		}
	}
	return false
}

func normalizeSubject(s string) string {
	s = strings.TrimSpace(s)
	for {
		next := s
		next = strings.TrimSpace(strings.TrimPrefix(next, "Re:"))
		next = strings.TrimSpace(strings.TrimPrefix(next, "RE:"))
		next = strings.TrimSpace(strings.TrimPrefix(next, "Fwd:"))
		next = strings.TrimSpace(strings.TrimPrefix(next, "FW:"))
		if next == s {
			return strings.ToLower(s)
		}
		s = next
	}
}

func textVal(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
