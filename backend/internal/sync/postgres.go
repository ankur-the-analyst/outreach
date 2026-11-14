package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
	"outreach/internal/mailbox"
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
	ID             uuid.UUID       `json:"id"`
	Direction      string          `json:"direction"`
	RFCMessageID   string          `json:"rfc_message_id"`
	From           string          `json:"from"`
	To             []string        `json:"to"`
	Subject        string          `json:"subject"`
	BodyText       string          `json:"body_text"`
	BodyHTML       string          `json:"body_html,omitempty"`
	At             *time.Time      `json:"at,omitempty"`
	Classification json.RawMessage `json:"classification,omitempty"`
}

type InboundHandler interface {
	Handle(ctx context.Context, workspaceID, mailboxID, messageID uuid.UUID, from, subject, body string) error
}

type Inbox interface {
	Open(ctx context.Context, workspaceID, mailboxID uuid.UUID) (mailbox.Operations, string, error)
}

type Postgres struct {
	pool   *db.Pool
	source Source
	inb    InboundHandler
	open   Inbox
}

func NewPostgres(pool *db.Pool, source Source) *Postgres {
	return &Postgres{pool: pool, source: source}
}

func (p *Postgres) WithInbound(h InboundHandler) *Postgres {
	p.inb = h
	return p
}

func (p *Postgres) WithOpener(o Inbox) *Postgres {
	p.open = o
	return p
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
	return p.syncMailbox(ctx, workspaceID, mailboxID, time.Time{}, time.Time{})
}

func (p *Postgres) SyncRange(ctx context.Context, workspaceID, mailboxID uuid.UUID, from, to time.Time) (int, error) {
	return p.syncMailbox(ctx, workspaceID, mailboxID, from, to)
}

func (p *Postgres) syncMailbox(ctx context.Context, workspaceID, mailboxID uuid.UUID, from, to time.Time) (int, error) {
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
	items, cursor, err := p.listItems(ctx, workspaceID, mailboxID, from, to)
	if err != nil {
		_ = p.mark(ctx, workspaceID, mailboxID, err, "")
		return 0, err
	}
	n := 0
	for _, it := range items {
		if !forMailbox(it, email) {
			continue
		}
		if err := p.ingest(ctx, workspaceID, mailboxID, email, it); err != nil {
			_ = p.mark(ctx, workspaceID, mailboxID, err, cursor)
			return n, err
		}
		n++
	}
	if err := p.mark(ctx, workspaceID, mailboxID, nil, cursor); err != nil {
		return n, err
	}
	return n, nil
}

func (p *Postgres) listItems(ctx context.Context, workspaceID, mailboxID uuid.UUID, from, to time.Time) ([]Item, string, error) {
	if p.open != nil {
		ops, _, err := p.open.Open(ctx, workspaceID, mailboxID)
		if err != nil && !errors.Is(err, mailbox.ErrNotConfigured) && !errors.Is(err, mailbox.ErrInboxFallback) {
			return nil, "", err
		}
		if err == nil {
			token := p.syncCursor(ctx, workspaceID, mailboxID)
			if from.IsZero() && to.IsZero() {
				start, end, n := p.missingHTMLSpan(ctx, workspaceID, mailboxID)
				if n > 0 {
					from = start.AddDate(0, 0, -1)
					to = end
				}
			}
			cur := mailbox.FetchCursor{Token: token, After: from, Before: to}
			res, ferr := ops.Fetch(ctx, cur)
			if ferr == nil {
				next := res.Cursor.Token
				if token != "" && (!cur.After.IsZero() || !cur.Before.IsZero()) {
					next = token
				}
				return messagesToItems(res.Messages), next, nil
			}
			if !errors.Is(ferr, mailbox.ErrInboxFallback) {
				return nil, "", ferr
			}
		}
	}
	if p.source == nil {
		return nil, "", fmt.Errorf("sync: source required")
	}
	items, err := p.source.List(ctx)
	return items, "", err
}

func (p *Postgres) missingHTMLSpan(ctx context.Context, workspaceID, mailboxID uuid.UUID) (time.Time, time.Time, int64) {
	var start, end time.Time
	var n int64
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.MissingHTMLSpan(ctx, db.MissingHTMLSpanParams{
			WorkspaceID: workspaceID, MailboxID: mailboxID,
		})
		if err != nil {
			return err
		}
		n = row.N
		if row.StartAt.Valid {
			start = row.StartAt.Time
		}
		if row.EndAt.Valid {
			end = row.EndAt.Time
		}
		return nil
	})
	if err != nil || n == 0 || start.IsZero() {
		return time.Time{}, time.Time{}, 0
	}
	return start, end, n
}

func (p *Postgres) syncCursor(ctx context.Context, workspaceID, mailboxID uuid.UUID) string {
	var token string
	_ = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetSyncState(ctx, db.GetSyncStateParams{
			WorkspaceID: workspaceID, MailboxID: mailboxID, Folder: "INBOX",
		})
		if err != nil {
			return err
		}
		if row.GmailHistoryID.Valid {
			token = strconv.FormatInt(row.GmailHistoryID.Int64, 10)
		}
		return nil
	})
	return token
}

func messagesToItems(msgs []mailbox.Message) []Item {
	out := make([]Item, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, Item{
			RFCMessageID: m.RFCMessageID,
			InReplyTo:    m.InReplyTo,
			Subject:      m.Subject,
			From:         m.From,
			To:           m.To,
			Text:         m.Text,
			HTML:         m.HTML,
			At:           m.At,
		})
	}
	return out
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
				var at *time.Time
				when := m.ReceivedAt
				if !when.Valid {
					when = m.SentAt
				}
				if when.Valid {
					t := when.Time.UTC()
					at = &t
				}
				th.Messages = append(th.Messages, Message{
					ID: m.ID, Direction: m.Direction, RFCMessageID: m.RfcMessageID,
					From: textVal(m.FromAddr), To: m.ToAddrs, Subject: textVal(m.Subject),
					BodyText: textVal(m.BodyText), BodyHTML: textVal(m.BodyHtml), At: at,
					Classification: json.RawMessage(m.Classification),
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

func (p *Postgres) MarkRead(ctx context.Context, workspaceID, threadID uuid.UUID) error {
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		n, err := q.MarkThreadRead(ctx, db.MarkThreadReadParams{WorkspaceID: workspaceID, ID: threadID})
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("not found")
		}
		return nil
	})
	return err
}

func (p *Postgres) ingest(ctx context.Context, workspaceID, mailboxID uuid.UUID, mailboxEmail string, it Item) error {
	if it.RFCMessageID == "" {
		return nil
	}
	err := p.pool.ExecTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries, _ pgx.Tx) error {
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
		htmlBody := pgtype.Text{String: it.HTML, Valid: it.HTML != ""}
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
			BodyHtml:     htmlBody,
			SentAt:       ts,
			ReceivedAt:   ts,
		}); err != nil {
			return err
		}
		if err := q.TouchThread(ctx, db.TouchThreadParams{
			WorkspaceID:   workspaceID,
			ID:            threadID,
			MailboxID:     mailboxID,
			LastMessageAt: ts,
			Unread:        dir == "in",
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil || p.inb == nil {
		return err
	}
	if strings.EqualFold(it.From, mailboxEmail) {
		return nil
	}
	var msgID uuid.UUID
	var classified bool
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMessageByRFC(ctx, db.GetMessageByRFCParams{
			WorkspaceID: workspaceID, MailboxID: mailboxID, RfcMessageID: it.RFCMessageID,
		})
		if err != nil {
			return err
		}
		msgID = row.ID
		classified = len(row.Classification) > 0
		return nil
	})
	if err != nil || classified {
		return err
	}
	return p.inb.Handle(ctx, workspaceID, mailboxID, msgID, it.From, it.Subject, it.Text)
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

func (p *Postgres) mark(ctx context.Context, workspaceID, mailboxID uuid.UUID, syncErr error, cursor string) error {
	errText := pgtype.Text{}
	if syncErr != nil {
		errText = pgtype.Text{String: syncErr.Error(), Valid: true}
	}
	hist := pgtype.Int8{}
	if n, err := strconv.ParseInt(cursor, 10, 64); err == nil {
		hist = pgtype.Int8{Int64: n, Valid: true}
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.UpsertSyncState(ctx, db.UpsertSyncStateParams{
			WorkspaceID:    workspaceID,
			MailboxID:      mailboxID,
			Folder:         "INBOX",
			LastSyncedAt:   pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			LastError:      errText,
			GmailHistoryID: hist,
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
