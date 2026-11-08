package compose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
	"outreach/internal/evidence"
	"outreach/internal/mime"
	"outreach/internal/pacing"
	"outreach/internal/sender"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("invalid draft transition")
)

type Draft struct {
	ID           uuid.UUID    `json:"id"`
	MailboxID    uuid.UUID    `json:"mailbox_id"`
	To           string       `json:"to"`
	RFCMessageID string       `json:"rfc_message_id"`
	Kind         string       `json:"kind"`
	Subject      string       `json:"subject"`
	BodyHTML     string       `json:"body_html"`
	BodyText     string       `json:"body_text"`
	Status       string       `json:"status"`
	FactsUsed    []Fact       `json:"facts_used,omitempty"`
	LedgerID     *uuid.UUID   `json:"ledger_id,omitempty"`
	LedgerState  pacing.State `json:"ledger_state,omitempty"`
	ReleaseAt    *time.Time   `json:"release_at,omitempty"`
}

type CreateInput struct {
	MailboxID uuid.UUID
	To        string
	Subject   string
	Text      string
	HTML      string
	Facts     []Fact
}

type Holder interface {
	Hold(ctx context.Context, workspaceID uuid.UUID, in sender.HoldInput) (sender.Entry, error)
	Cancel(ctx context.Context, workspaceID, id uuid.UUID) (sender.Entry, error)
}

type Citer interface {
	Cite(ctx context.Context, workspaceID uuid.UUID, ids []uuid.UUID) ([]evidence.Item, error)
}

type ComposeInput struct {
	MailboxID uuid.UUID
	To        string
	Subject   string
	Facts     []Fact
}

type ComposeResult struct {
	Draft       Draft     `json:"draft"`
	Failures    []Failure `json:"failures"`
	Regenerated bool      `json:"regenerated"`
}

type Postgres struct {
	pool  *db.Pool
	hold  Holder
	write Completer
	cite  Citer
}

func NewPostgres(pool *db.Pool, hold Holder) *Postgres {
	return &Postgres{pool: pool, hold: hold}
}

func (p *Postgres) WithWriter(w Completer) *Postgres {
	p.write = w
	return p
}

func (p *Postgres) WithCite(c Citer) *Postgres {
	p.cite = c
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
		return "", ErrNotFound
	}
	return role, err
}

func (p *Postgres) Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Draft, error) {
	to := strings.ToLower(strings.TrimSpace(in.To))
	if to == "" || !strings.Contains(to, "@") {
		return Draft{}, fmt.Errorf("compose: to required")
	}
	subject := strings.TrimSpace(in.Subject)
	if subject == "" {
		return Draft{}, fmt.Errorf("compose: subject required")
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return Draft{}, fmt.Errorf("compose: body required")
	}
	if in.MailboxID == uuid.Nil {
		return Draft{}, fmt.Errorf("compose: mailbox_id required")
	}
	msgid, err := mime.NewMessageID("outreach.test")
	if err != nil {
		return Draft{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Draft{}, err
	}
	factsJSON, err := json.Marshal(in.Facts)
	if err != nil {
		return Draft{}, err
	}
	if in.Facts == nil {
		factsJSON = []byte("[]")
	}
	var out Draft
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: in.MailboxID}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		row, err := q.InsertDraft(ctx, db.InsertDraftParams{
			WorkspaceID:  workspaceID,
			ID:           id,
			MailboxID:    in.MailboxID,
			ToAddr:       to,
			RfcMessageID: msgid,
			Kind:         "outbound",
			Subject:      subject,
			BodyHtml:     in.HTML,
			BodyText:     text,
			FactsUsed:    factsJSON,
			Status:       "pending",
			CreatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		})
		if err != nil {
			return err
		}
		out = draftFrom(row.ID, row.MailboxID, row.ToAddr, row.RfcMessageID, row.Kind, row.Subject, row.BodyHtml, row.BodyText, row.Status, nil, "", pgtype.Timestamptz{})
		out.FactsUsed = in.Facts
		return nil
	})
	return out, err
}

func (p *Postgres) Compose(ctx context.Context, workspaceID uuid.UUID, in ComposeInput) (ComposeResult, error) {
	if p.cite == nil {
		return ComposeResult{}, fmt.Errorf("compose: citer required")
	}
	ids := make([]uuid.UUID, 0, len(in.Facts))
	for _, f := range in.Facts {
		ids = append(ids, f.EvidenceID)
	}
	cited, err := p.cite.Cite(ctx, workspaceID, ids)
	if err != nil {
		return ComposeResult{}, err
	}
	body, regen, fails, err := Generate(ctx, p.write, cited, in.Facts)
	if err != nil {
		return ComposeResult{}, err
	}
	d, err := p.Create(ctx, workspaceID, CreateInput{
		MailboxID: in.MailboxID, To: in.To, Subject: in.Subject, Text: body, Facts: in.Facts,
	})
	if err != nil {
		return ComposeResult{}, err
	}
	if fails == nil {
		fails = []Failure{}
	}
	return ComposeResult{Draft: d, Failures: fails, Regenerated: regen}, nil
}

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID) ([]Draft, error) {
	var out []Draft
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListReviewDrafts(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]Draft, 0, len(rows))
		for _, row := range rows {
			out = append(out, draftFrom(row.ID, row.MailboxID, row.ToAddr, row.RfcMessageID, row.Kind, row.Subject, row.BodyHtml, row.BodyText, row.Status, row.LedgerID, row.LedgerState.String, row.ReleaseAt))
		}
		return nil
	})
	return out, err
}

func (p *Postgres) Approve(ctx context.Context, workspaceID, userID, draftID uuid.UUID) (Draft, error) {
	if p.hold == nil {
		return Draft{}, fmt.Errorf("compose: holder required")
	}
	d, err := p.get(ctx, workspaceID, draftID)
	if err != nil {
		return Draft{}, err
	}
	if d.Status == "approved" {
		return d, nil
	}
	if d.Status != "pending" {
		return Draft{}, ErrConflict
	}
	ent, err := p.hold.Hold(ctx, workspaceID, sender.HoldInput{
		MailboxID: d.MailboxID, RFCMessageID: d.RFCMessageID, DraftID: d.ID,
	})
	if err != nil {
		return Draft{}, err
	}
	return p.setStatus(ctx, workspaceID, draftID, "approved", userID, &ent)
}

func (p *Postgres) Reject(ctx context.Context, workspaceID, draftID uuid.UUID) (Draft, error) {
	d, err := p.get(ctx, workspaceID, draftID)
	if err != nil {
		return Draft{}, err
	}
	if d.Status != "pending" {
		return Draft{}, ErrConflict
	}
	return p.setStatus(ctx, workspaceID, draftID, "rejected", uuid.Nil, nil)
}

func (p *Postgres) Regenerate(ctx context.Context, workspaceID, draftID uuid.UUID) (ComposeResult, error) {
	d, err := p.get(ctx, workspaceID, draftID)
	if err != nil {
		return ComposeResult{}, err
	}
	if d.Status != "pending" {
		return ComposeResult{}, ErrConflict
	}
	cited := []evidence.Item{}
	if len(d.FactsUsed) > 0 {
		if p.cite == nil {
			return ComposeResult{}, fmt.Errorf("compose: citer required")
		}
		ids := make([]uuid.UUID, 0, len(d.FactsUsed))
		for _, f := range d.FactsUsed {
			ids = append(ids, f.EvidenceID)
		}
		cited, err = p.cite.Cite(ctx, workspaceID, ids)
		if err != nil {
			return ComposeResult{}, err
		}
	}
	body, regen, fails, err := Generate(ctx, p.write, cited, d.FactsUsed)
	if err != nil {
		return ComposeResult{}, err
	}
	var out Draft
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.UpdateDraftBody(ctx, db.UpdateDraftBodyParams{
			WorkspaceID: workspaceID, ID: draftID, BodyText: body, BodyHtml: "",
		})
		if err != nil {
			return err
		}
		out = draftFrom(row.ID, row.MailboxID, row.ToAddr, row.RfcMessageID, row.Kind, row.Subject, row.BodyHtml, row.BodyText, row.Status, nil, "", pgtype.Timestamptz{})
		out.FactsUsed = d.FactsUsed
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ComposeResult{}, ErrConflict
	}
	if err != nil {
		return ComposeResult{}, err
	}
	if fails == nil {
		fails = []Failure{}
	}
	return ComposeResult{Draft: out, Failures: fails, Regenerated: regen}, nil
}

func (p *Postgres) Undo(ctx context.Context, workspaceID, draftID uuid.UUID) (Draft, error) {
	if p.hold == nil {
		return Draft{}, fmt.Errorf("compose: holder required")
	}
	d, err := p.get(ctx, workspaceID, draftID)
	if err != nil {
		return Draft{}, err
	}
	if d.Status != "approved" {
		return Draft{}, ErrConflict
	}
	var ledger db.SendLedger
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetLedgerByDraft(ctx, db.GetLedgerByDraftParams{WorkspaceID: workspaceID, DraftID: draftID})
		if err != nil {
			return err
		}
		ledger = row
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return p.setStatus(ctx, workspaceID, draftID, "cancelled", uuid.Nil, nil)
	}
	if err != nil {
		return Draft{}, err
	}
	if pacing.State(ledger.State) != pacing.StateHeld {
		return Draft{}, ErrConflict
	}
	if _, err := p.hold.Cancel(ctx, workspaceID, ledger.ID); err != nil {
		return Draft{}, err
	}
	return p.setStatus(ctx, workspaceID, draftID, "cancelled", uuid.Nil, nil)
}

func (p *Postgres) get(ctx context.Context, workspaceID, id uuid.UUID) (Draft, error) {
	var out Draft
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetDraft(ctx, db.GetDraftParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			return err
		}
		out = draftFrom(row.ID, row.MailboxID, row.ToAddr, row.RfcMessageID, row.Kind, row.Subject, row.BodyHtml, row.BodyText, row.Status, nil, "", pgtype.Timestamptz{})
		if len(row.FactsUsed) > 0 {
			if err := json.Unmarshal(row.FactsUsed, &out.FactsUsed); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) setStatus(ctx context.Context, workspaceID, id uuid.UUID, status string, approvedBy uuid.UUID, ent *sender.Entry) (Draft, error) {
	var out Draft
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		var by *uuid.UUID
		if approvedBy != uuid.Nil {
			by = &approvedBy
		}
		row, err := q.UpdateDraftStatus(ctx, db.UpdateDraftStatusParams{
			WorkspaceID: workspaceID, ID: id, Status: status, ApprovedBy: by,
		})
		if err != nil {
			return err
		}
		var lid *uuid.UUID
		var st string
		var rel pgtype.Timestamptz
		if ent != nil {
			lid = &ent.ID
			st = string(ent.State)
			if ent.ReleaseAt != nil {
				rel = pgtype.Timestamptz{Time: *ent.ReleaseAt, Valid: true}
			}
		}
		out = draftFrom(row.ID, row.MailboxID, row.ToAddr, row.RfcMessageID, row.Kind, row.Subject, row.BodyHtml, row.BodyText, row.Status, lid, st, rel)
		return nil
	})
	return out, err
}

func draftFrom(id, mailboxID uuid.UUID, to, rfc, kind, subject, html, text, status string, ledgerID *uuid.UUID, ledgerState string, release pgtype.Timestamptz) Draft {
	d := Draft{
		ID: id, MailboxID: mailboxID, To: to, RFCMessageID: rfc, Kind: kind,
		Subject: subject, BodyHTML: html, BodyText: text, Status: status,
		LedgerID: ledgerID, LedgerState: pacing.State(ledgerState),
	}
	if release.Valid {
		t := release.Time.UTC()
		d.ReleaseAt = &t
	}
	return d
}
