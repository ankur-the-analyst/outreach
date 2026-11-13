package tracking

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

var ErrNotFound = errors.New("not found")

type Event struct {
	EnrolmentID uuid.UUID
	DraftID     uuid.UUID
	Kind        string
	URL         string
}

type Postgres struct {
	pool *db.Pool
	now  func() time.Time
}

func NewPostgres(pool *db.Pool, now func() time.Time) *Postgres {
	if now == nil {
		now = time.Now
	}
	return &Postgres{pool: pool, now: now}
}

func (p *Postgres) Record(ctx context.Context, workspaceID uuid.UUID, in Event) error {
	kind := strings.TrimSpace(in.Kind)
	if kind != KindOpen && kind != KindClick {
		return fmt.Errorf("tracking: invalid kind")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	at := p.now().UTC()
	url := pgtype.Text{}
	if u := strings.TrimSpace(in.URL); u != "" {
		url = pgtype.Text{String: u, Valid: true}
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		enID, err := q.GetDraftEnrolment(ctx, db.GetDraftEnrolmentParams{WorkspaceID: workspaceID, ID: in.DraftID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if enID == nil || *enID != in.EnrolmentID {
			return ErrNotFound
		}
		if _, err := q.InsertMailEvent(ctx, db.InsertMailEventParams{
			WorkspaceID: workspaceID,
			ID:          id,
			EnrolmentID: in.EnrolmentID,
			DraftID:     in.DraftID,
			Kind:        kind,
			Url:         url,
			At:          pgtype.Timestamptz{Time: at, Valid: true},
		}); err != nil {
			return fmt.Errorf("tracking: insert: %w", err)
		}
		stamp := pgtype.Timestamptz{Time: at, Valid: true}
		if kind == KindClick {
			return q.TouchEnrolmentClicked(ctx, db.TouchEnrolmentClickedParams{
				At: stamp, WorkspaceID: workspaceID, ID: in.EnrolmentID,
			})
		}
		return q.TouchEnrolmentOpened(ctx, db.TouchEnrolmentOpenedParams{
			At: stamp, WorkspaceID: workspaceID, ID: in.EnrolmentID,
		})
	})
}

type Row struct {
	Kind string
	URL  string
	At   time.Time
}

func (p *Postgres) List(ctx context.Context, workspaceID, enrolmentID uuid.UUID) ([]Row, error) {
	var out []Row
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListMailEventsForEnrolment(ctx, db.ListMailEventsForEnrolmentParams{
			WorkspaceID: workspaceID, EnrolmentID: enrolmentID,
		})
		if err != nil {
			return err
		}
		out = make([]Row, 0, len(rows))
		for _, r := range rows {
			out = append(out, Row{Kind: r.Kind, URL: r.Url.String, At: r.At.Time.UTC()})
		}
		return nil
	})
	return out, err
}

func (p *Postgres) Times(ctx context.Context, workspaceID, enrolmentID uuid.UUID) (opened, clicked *time.Time, err error) {
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetEnrolmentTracking(ctx, db.GetEnrolmentTrackingParams{WorkspaceID: workspaceID, ID: enrolmentID})
		if err != nil {
			return err
		}
		if row.OpenedAt.Valid {
			t := row.OpenedAt.Time.UTC()
			opened = &t
		}
		if row.ClickedAt.Valid {
			t := row.ClickedAt.Time.UTC()
			clicked = &t
		}
		return nil
	})
	return opened, clicked, err
}
