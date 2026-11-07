package sender

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
	"outreach/internal/pacing"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("invalid ledger transition")
	ErrPaused     = errors.New("sends paused")
	ErrSuppressed = errors.New("recipient suppressed")
)

type Blocker interface {
	Blocked(ctx context.Context, workspaceID uuid.UUID, emails []string) error
}

type Entry struct {
	ID              uuid.UUID    `json:"id"`
	MailboxID       uuid.UUID    `json:"mailbox_id"`
	SendingDomainID uuid.UUID    `json:"sending_domain_id"`
	DraftID         uuid.UUID    `json:"draft_id"`
	RFCMessageID    string       `json:"rfc_message_id"`
	State           pacing.State `json:"state"`
	ReleaseAt       *time.Time   `json:"release_at,omitempty"`
}

type HoldInput struct {
	MailboxID    uuid.UUID
	RFCMessageID string
	DraftID      uuid.UUID
}

type ReconcileEnqueuer interface {
	EnqueueReconcile(ctx context.Context, workspaceID, ledgerID uuid.UUID, in SendInput) error
}

type Postgres struct {
	pool    *db.Pool
	now     func() time.Time
	hold    time.Duration
	unknown time.Duration
	dial    Dialer
	queue   ReconcileEnqueuer
	block   Blocker
}

func NewPostgres(pool *db.Pool, now func() time.Time, hold time.Duration) *Postgres {
	if now == nil {
		now = time.Now
	}
	if hold <= 0 {
		hold = 5 * time.Minute
	}
	return &Postgres{pool: pool, now: now, hold: hold, unknown: 15 * time.Minute}
}

func (p *Postgres) WithUnknownTimeout(d time.Duration) *Postgres {
	p.unknown = d
	return p
}

func (p *Postgres) WithQueue(q ReconcileEnqueuer) *Postgres {
	p.queue = q
	return p
}

func (p *Postgres) WithBlocker(b Blocker) *Postgres {
	p.block = b
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

func (p *Postgres) Hold(ctx context.Context, workspaceID uuid.UUID, in HoldInput) (Entry, error) {
	msgid := strings.TrimSpace(in.RFCMessageID)
	if msgid == "" {
		return Entry{}, fmt.Errorf("sender: rfc_message_id required")
	}
	draftID := in.DraftID
	if draftID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return Entry{}, err
		}
		draftID = id
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Entry{}, err
	}
	release := p.now().Add(p.hold)
	var out Entry
	err = p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		mb, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: in.MailboxID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		row, err := q.InsertLedger(ctx, db.InsertLedgerParams{
			WorkspaceID:     workspaceID,
			ID:              id,
			MailboxID:       mb.ID,
			SendingDomainID: mb.SendingDomainID,
			DraftID:         draftID,
			IsFirstTouch:    true,
			RfcMessageID:    msgid,
			State:           string(pacing.StateHeld),
			ReleaseAt:       pgtype.Timestamptz{Time: release, Valid: true},
		})
		if err != nil {
			return err
		}
		out = entryFrom(row.ID, row.MailboxID, row.SendingDomainID, row.DraftID, row.RfcMessageID, row.State, row.ReleaseAt)
		return nil
	})
	return out, err
}

func (p *Postgres) Advance(ctx context.Context, workspaceID, id uuid.UUID, to pacing.State) (Entry, error) {
	var out Entry
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		cur, err := q.GetLedger(ctx, db.GetLedgerParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		from, err := pacing.ParseState(cur.State)
		if err != nil {
			return err
		}
		if !from.CanAdvance(to) {
			return ErrConflict
		}
		row, err := q.UpdateLedgerState(ctx, db.UpdateLedgerStateParams{
			WorkspaceID: workspaceID,
			ID:          id,
			FromState:   string(from),
			NextState:   string(to),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrConflict
			}
			return err
		}
		out = entryFrom(row.ID, row.MailboxID, row.SendingDomainID, row.DraftID, row.RfcMessageID, row.State, row.ReleaseAt)
		return nil
	})
	return out, err
}

func (p *Postgres) Cancel(ctx context.Context, workspaceID, id uuid.UUID) (Entry, error) {
	return p.Advance(ctx, workspaceID, id, pacing.StateCancelled)
}

func (p *Postgres) Usage(ctx context.Context, workspaceID, mailboxID uuid.UUID) (int64, error) {
	var n int64
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: mailboxID}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		var err error
		n, err = q.CountMailboxUsage(ctx, db.CountMailboxUsageParams{WorkspaceID: workspaceID, MailboxID: mailboxID})
		return err
	})
	return n, err
}

func entryFrom(id, mailboxID, domainID, draftID uuid.UUID, rfc, state string, release pgtype.Timestamptz) Entry {
	e := Entry{
		ID: id, MailboxID: mailboxID, SendingDomainID: domainID, DraftID: draftID,
		RFCMessageID: rfc, State: pacing.State(state),
	}
	if release.Valid {
		t := release.Time.UTC()
		e.ReleaseAt = &t
	}
	return e
}
