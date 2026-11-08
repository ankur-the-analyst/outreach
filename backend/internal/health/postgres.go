package health

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"outreach/internal/db"
)

var ErrNotFound = errors.New("health: not found")

type Postgres struct {
	pool *db.Pool
	now  func() time.Time
}

func NewPostgres(pool *db.Pool) *Postgres {
	return &Postgres{pool: pool, now: time.Now}
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

func (p *Postgres) Snapshot(ctx context.Context, workspaceID uuid.UUID) error {
	return p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		unknown, err := q.CountLedgerState(ctx, db.CountLedgerStateParams{WorkspaceID: workspaceID, State: "unknown"})
		if err != nil {
			return err
		}
		paused, err := q.CountMailboxesStatus(ctx, db.CountMailboxesStatusParams{WorkspaceID: workspaceID, Status: "paused"})
		if err != nil {
			return err
		}
		var lag *float64
		oldest, err := q.OldestSyncAt(ctx, workspaceID)
		if err != nil {
			return err
		}
		if oldest.Valid {
			s := p.now().UTC().Sub(oldest.Time.UTC()).Seconds()
			if s < 0 {
				s = 0
			}
			lag = &s
		}
		for _, row := range Metrics(unknown, paused, lag) {
			if _, err := q.InsertHealthSample(ctx, db.InsertHealthSampleParams{
				WorkspaceID: workspaceID,
				Metric:      row.Metric,
				Value:       row.Value,
				Window:      row.Window,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID) ([]Sample, error) {
	var out []Sample
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListHealthSamples(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]Sample, 0, len(rows))
		for _, row := range rows {
			out = append(out, Sample{
				ID:        row.ID,
				Metric:    row.Metric,
				Value:     row.Value,
				Window:    row.Window,
				CreatedAt: row.CreatedAt.Time,
			})
		}
		return nil
	})
	return out, err
}
