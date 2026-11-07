package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	inner *pgxpool.Pool
	q     *Queries
}

func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	inner, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}
	p := &Pool{inner: inner, q: New(inner)}
	if err := p.Ping(ctx); err != nil {
		inner.Close()
		return nil, err
	}
	return p, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	if _, err := p.q.Ping(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	return nil
}

func (p *Pool) Close() {
	p.inner.Close()
}

func (p *Pool) Queries() *Queries {
	return p.q
}

func (p *Pool) Pgx() *pgxpool.Pool {
	return p.inner
}
