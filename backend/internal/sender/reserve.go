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
	"outreach/internal/suppression"
)

var (
	ErrNoneEligible = errors.New("no eligible mailbox")
	errRetry        = errors.New("retry next mailbox")
)

type ReserveInput struct {
	MailboxIDs   []uuid.UUID
	Strategy     string
	RFCMessageID string
	DraftID      uuid.UUID
	To           []string
}

func (p *Postgres) Reserve(ctx context.Context, workspaceID uuid.UUID, in ReserveInput) (Entry, error) {
	msgid := strings.TrimSpace(in.RFCMessageID)
	if msgid == "" {
		return Entry{}, fmt.Errorf("sender: rfc_message_id required")
	}
	if len(in.MailboxIDs) == 0 {
		return Entry{}, fmt.Errorf("sender: mailbox_ids required")
	}
	if err := p.guardReserve(ctx, workspaceID, in.To); err != nil {
		return Entry{}, err
	}
	strat, err := pacing.Lookup(in.Strategy)
	if err != nil {
		return Entry{}, err
	}
	draftID := in.DraftID
	if draftID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return Entry{}, err
		}
		draftID = id
	}

	var unlockedOrder []uuid.UUID
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		views, ledger, err := snapshot(ctx, q, workspaceID, in.MailboxIDs)
		if err != nil {
			return err
		}
		unlockedOrder = strat.Order(eligibleOnly(views), ledger, nil)
		return nil
	})
	if err != nil {
		return Entry{}, err
	}
	if len(unlockedOrder) == 0 {
		return Entry{}, ErrNoneEligible
	}

	for _, candidate := range unlockedOrder {
		var out Entry
		err := p.pool.ExecTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries, _ pgx.Tx) error {
			mb, err := q.LockMailbox(ctx, db.LockMailboxParams{WorkspaceID: workspaceID, ID: candidate})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errRetry
				}
				return err
			}
			if _, err := q.LockSendingDomain(ctx, db.LockSendingDomainParams{WorkspaceID: workspaceID, ID: mb.SendingDomainID}); err != nil {
				return err
			}
			views, ledger, err := snapshot(ctx, q, workspaceID, in.MailboxIDs)
			if err != nil {
				return err
			}
			order := strat.Order(eligibleOnly(views), ledger, nil)
			if len(order) == 0 || order[0] != candidate {
				return errRetry
			}
			id, err := uuid.NewV7()
			if err != nil {
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
				State:           string(pacing.StateReserved),
				ReservedAt:      pgtype.Timestamptz{Time: p.now(), Valid: true},
			})
			if err != nil {
				return err
			}
			out = entryFrom(row.ID, row.MailboxID, row.SendingDomainID, row.DraftID, row.RfcMessageID, row.State, row.ReleaseAt)
			return nil
		})
		if errors.Is(err, errRetry) {
			continue
		}
		if err != nil {
			return Entry{}, err
		}
		return out, nil
	}
	return Entry{}, ErrNoneEligible
}

func (p *Postgres) guardReserve(ctx context.Context, workspaceID uuid.UUID, to []string) error {
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		ws, err := q.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return err
		}
		if ws.SendsPaused {
			return ErrPaused
		}
		return nil
	})
	if err != nil {
		return err
	}
	if p.block == nil {
		return nil
	}
	if err := p.block.Blocked(ctx, workspaceID, to); err != nil {
		if errors.Is(err, suppression.ErrSuppressed) {
			return ErrSuppressed
		}
		return err
	}
	return nil
}

func snapshot(ctx context.Context, q *db.Queries, workspaceID uuid.UUID, ids []uuid.UUID) ([]pacing.MailboxView, pacing.LedgerView, error) {
	rows, err := q.ListMailboxesByIDs(ctx, db.ListMailboxesByIDsParams{WorkspaceID: workspaceID, Ids: ids})
	if err != nil {
		return nil, pacing.LedgerView{}, err
	}
	touches, err := q.ListFirstTouchAt(ctx, workspaceID)
	if err != nil {
		return nil, pacing.LedgerView{}, err
	}
	ledger := pacing.LedgerView{LastFirstTouch: firstTouchMap(touches)}
	domainUsed := map[uuid.UUID]int{}
	views := make([]pacing.MailboxView, 0, len(rows))
	for _, row := range rows {
		used, err := q.CountMailboxUsage(ctx, db.CountMailboxUsageParams{WorkspaceID: workspaceID, MailboxID: row.ID})
		if err != nil {
			return nil, pacing.LedgerView{}, err
		}
		dused, ok := domainUsed[row.SendingDomainID]
		if !ok {
			n, err := q.CountDomainUsage(ctx, db.CountDomainUsageParams{WorkspaceID: workspaceID, SendingDomainID: row.SendingDomainID})
			if err != nil {
				return nil, pacing.LedgerView{}, err
			}
			dused = int(n)
			domainUsed[row.SendingDomainID] = dused
		}
		dom, err := q.GetSendingDomain(ctx, db.GetSendingDomainParams{WorkspaceID: workspaceID, ID: row.SendingDomainID})
		if err != nil {
			return nil, pacing.LedgerView{}, err
		}
		views = append(views, pacing.MailboxView{
			ID: row.ID, DomainID: row.SendingDomainID, Status: row.Status,
			DailyLimit: int(row.EffectiveDailyLimit), Used: int(used),
			DomainLimit: int(dom.DailyLimit), DomainUsed: dused,
		})
	}
	return views, ledger, nil
}

func firstTouchMap(rows []db.ListFirstTouchAtRow) map[uuid.UUID]time.Time {
	m := make(map[uuid.UUID]time.Time, len(rows))
	for _, r := range rows {
		if r.LastAt.Valid {
			m[r.MailboxID] = r.LastAt.Time
		}
	}
	return m
}

func eligibleOnly(views []pacing.MailboxView) []pacing.MailboxView {
	out := make([]pacing.MailboxView, 0, len(views))
	for _, v := range views {
		if pacing.Eligible(v) {
			out = append(out, v)
		}
	}
	return out
}
