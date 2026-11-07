package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type GUC struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	SessionID   string
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func applyGUC(ctx context.Context, e execer, g GUC, local bool) error {
	if g.WorkspaceID != uuid.Nil {
		if err := setConfig(ctx, e, "app.workspace_id", g.WorkspaceID.String(), local); err != nil {
			return err
		}
	}
	if g.UserID != uuid.Nil {
		if err := setConfig(ctx, e, "app.user_id", g.UserID.String(), local); err != nil {
			return err
		}
	}
	if g.SessionID != "" {
		if err := setConfig(ctx, e, "app.session_id", g.SessionID, local); err != nil {
			return err
		}
	}
	return nil
}

func setConfig(ctx context.Context, e execer, key, value string, local bool) error {
	_, err := e.Exec(ctx, "select set_config($1, $2, $3)", key, value, local)
	if err != nil {
		return fmt.Errorf("db guc %s: %w", key, err)
	}
	return nil
}

func resetGUC(ctx context.Context, e execer) {
	_, _ = e.Exec(ctx, "reset app.workspace_id")
	_, _ = e.Exec(ctx, "reset app.user_id")
	_, _ = e.Exec(ctx, "reset app.session_id")
}

func (p *Pool) InTx(ctx context.Context, g GUC, fn func(ctx context.Context, q *Queries) error) error {
	return p.ExecTx(ctx, g, func(ctx context.Context, q *Queries, _ pgx.Tx) error {
		return fn(ctx, q)
	})
}

func (p *Pool) ExecTx(ctx context.Context, g GUC, fn func(ctx context.Context, q *Queries, tx pgx.Tx) error) error {
	tx, err := p.inner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := applyGUC(ctx, tx, g, true); err != nil {
		return err
	}
	if err := fn(ctx, New(tx), tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		if err == pgx.ErrTxClosed {
			return nil
		}
		return fmt.Errorf("db commit: %w", err)
	}
	return nil
}

func (p *Pool) WithGUC(ctx context.Context, g GUC, fn func(ctx context.Context, q *Queries) error) error {
	conn, err := p.inner.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("db acquire: %w", err)
	}
	defer conn.Release()
	defer resetGUC(ctx, conn)
	if err := applyGUC(ctx, conn, g, false); err != nil {
		return err
	}
	return fn(ctx, New(conn))
}
