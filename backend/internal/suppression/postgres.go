package suppression

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
)

var ErrSuppressed = errors.New("recipient suppressed")

type Postgres struct {
	pool *db.Pool
	hmac Hasher
}

func NewPostgres(pool *db.Pool, hmac Hasher) *Postgres {
	return &Postgres{pool: pool, hmac: hmac}
}

type Entry struct {
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
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
	return role, err
}

func (p *Postgres) Add(ctx context.Context, workspaceID uuid.UUID, kind, value, reason string) (Entry, error) {
	reason = Normalize(reason)
	if reason == "" {
		return Entry{}, fmt.Errorf("suppression: reason required")
	}
	sum, norm, err := Hash(p.hmac, kind, value)
	if err != nil {
		return Entry{}, err
	}
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertSuppression(ctx, db.InsertSuppressionParams{
			WorkspaceID: workspaceID,
			Kind:        kind,
			ValueHash:   sum,
			ValueCitext: pgtype.Text{String: norm, Valid: true},
			Reason:      reason,
		})
	})
	if err != nil {
		return Entry{}, err
	}
	return Entry{Kind: kind, Value: norm, Reason: reason}, nil
}

func (p *Postgres) Blocked(ctx context.Context, workspaceID uuid.UUID, emails []string) error {
	if len(emails) == 0 {
		return nil
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		for _, raw := range emails {
			if err := p.exists(ctx, q, workspaceID, KindEmail, Normalize(raw)); err != nil {
				return err
			}
			if domain := DomainOf(raw); domain != "" {
				if err := p.exists(ctx, q, workspaceID, KindDomain, domain); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (p *Postgres) exists(ctx context.Context, q *db.Queries, workspaceID uuid.UUID, kind, value string) error {
	sum, _, err := Hash(p.hmac, kind, value)
	if err != nil {
		return err
	}
	ok, err := q.SuppressionExists(ctx, db.SuppressionExistsParams{
		WorkspaceID: workspaceID, Kind: kind, ValueHash: sum,
	})
	if err != nil {
		return err
	}
	if ok {
		return ErrSuppressed
	}
	return nil
}
