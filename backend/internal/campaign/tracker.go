package campaign

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
)

type TrackerRow struct {
	ID            uuid.UUID  `json:"id"`
	ApplicationID *uuid.UUID `json:"application_id,omitempty"`
	Contact       string     `json:"contact"`
	Email         string     `json:"email"`
	Company       string     `json:"company,omitempty"`
	Campaign      string     `json:"campaign"`
	CampaignID    uuid.UUID  `json:"campaign_id"`
	Stage         string     `json:"stage"`
	OpenedAt      *time.Time `json:"opened_at,omitempty"`
	ClickedAt     *time.Time `json:"clicked_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func TrackerStage(enrolmentStatus, draftStatus, ledgerState string) string {
	switch enrolmentStatus {
	case "replied":
		return "replied"
	case "stopped":
		return "stopped"
	case "bounced":
		return "bounced"
	case "unsubscribed":
		return "unsubscribed"
	}
	if ledgerState == "sent" || draftStatus == "sent" {
		return "sent"
	}
	if enrolmentStatus == "waiting_review" || enrolmentStatus == "held" ||
		draftStatus == "pending" || draftStatus == "approved" ||
		ledgerState == "held" || ledgerState == "reserved" || ledgerState == "sending" || ledgerState == "unknown" {
		return "drafted"
	}
	return "queued"
}

func PreferApplicationStage(derived, app string) string {
	switch app {
	case "followed_up", "interview", "offer", "rejected", "closed":
		return app
	default:
		return derived
	}
}

func (p *Postgres) ListTracker(ctx context.Context, workspaceID uuid.UUID) ([]TrackerRow, error) {
	var out []TrackerRow
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListTrackerRows(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]TrackerRow, 0, len(rows))
		for _, row := range rows {
			updated := row.CreatedAt.Time
			if row.DraftCreatedAt.Valid {
				updated = row.DraftCreatedAt.Time
			}
			derived := TrackerStage(row.EnrolmentStatus, row.DraftStatus, row.LedgerState)
			out = append(out, TrackerRow{
				ID:            row.ID,
				ApplicationID: row.ApplicationID,
				Contact:       displayName(row.FirstName, row.LastName, row.Email),
				Email:         row.Email,
				Company:       textString(row.CompanyName),
				Campaign:      row.CampaignName,
				CampaignID:    row.CampaignID,
				Stage:         PreferApplicationStage(derived, row.ApplicationStage),
				OpenedAt:      timestamptzPtr(row.OpenedAt),
				ClickedAt:     timestamptzPtr(row.ClickedAt),
				UpdatedAt:     updated,
			})
		}
		return nil
	})
	return out, err
}

func displayName(first, last pgtype.Text, email string) string {
	n := stringsJoin(textString(first), textString(last))
	if n == "" {
		return email
	}
	return n
}

func stringsJoin(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + " " + b
}

func textString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	u := t.Time.UTC()
	return &u
}
