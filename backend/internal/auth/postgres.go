package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
)

var ErrEmailTaken = errors.New("email taken")

type Postgres struct {
	pool *db.Pool
}

func NewPostgres(pool *db.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) CreateAccount(ctx context.Context, userID, workspaceID uuid.UUID, email, name, passwordHash, sessionID string, expires time.Time) error {
	wsName := workspaceName(name, email)
	err := p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID, UserID: userID, SessionID: sessionID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: workspaceID, Name: wsName}); err != nil {
			return fmt.Errorf("auth: workspace: %w", err)
		}
		if err := q.InsertUser(ctx, db.InsertUserParams{
			ID:           userID,
			Email:        email,
			Name:         text(name),
			PasswordHash: text(passwordHash),
		}); err != nil {
			return fmt.Errorf("auth: user: %w", err)
		}
		if err := q.InsertMembership(ctx, db.InsertMembershipParams{
			WorkspaceID: workspaceID,
			UserID:      userID,
			Role:        roleOwner,
		}); err != nil {
			return fmt.Errorf("auth: membership: %w", err)
		}
		return q.InsertSession(ctx, db.InsertSessionParams{
			ID:          sessionID,
			UserID:      userID,
			WorkspaceID: workspaceID,
			ExpiresAt:   timestamptz(expires),
		})
	})
	if isUnique(err) {
		return ErrEmailTaken
	}
	return err
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (p *Postgres) UserByEmail(ctx context.Context, email string) (uuid.UUID, string, string, error) {
	u, err := p.pool.Queries().GetUserByEmail(ctx, email)
	if err != nil {
		return uuid.Nil, "", "", err
	}
	return u.ID, u.PasswordHash.String, u.Name.String, nil
}

func (p *Postgres) Memberships(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var ws uuid.UUID
	err := p.pool.WithGUC(ctx, db.GUC{UserID: userID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListMembershipsByUser(ctx, userID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return pgx.ErrNoRows
		}
		ws = rows[0].WorkspaceID
		return nil
	})
	return ws, err
}

func (p *Postgres) CreateSession(ctx context.Context, s Session, expires time.Time) error {
	return p.pool.InTx(ctx, db.GUC{WorkspaceID: s.WorkspaceID, UserID: s.UserID, SessionID: s.ID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertSession(ctx, db.InsertSessionParams{
			ID:          s.ID,
			UserID:      s.UserID,
			WorkspaceID: s.WorkspaceID,
			ExpiresAt:   timestamptz(expires),
		})
	})
}

func (p *Postgres) SessionByID(ctx context.Context, id string) (Session, time.Time, error) {
	var out Session
	var exp time.Time
	err := p.pool.WithGUC(ctx, db.GUC{SessionID: id}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetSession(ctx, id)
		if err != nil {
			return err
		}
		out = Session{ID: row.ID, UserID: row.UserID, WorkspaceID: row.WorkspaceID}
		if row.ExpiresAt.Valid {
			exp = row.ExpiresAt.Time
		}
		return nil
	})
	return out, exp, err
}

func (p *Postgres) DeleteSession(ctx context.Context, id string) error {
	return p.pool.WithGUC(ctx, db.GUC{SessionID: id}, func(ctx context.Context, q *db.Queries) error {
		return q.DeleteSession(ctx, id)
	})
}

func text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
