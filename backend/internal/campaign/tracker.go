package campaign

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
)

type TrackerRow struct {
	ID         uuid.UUID `json:"id"`
	Contact    string    `json:"contact"`
	Email      string    `json:"email"`
	Company    string    `json:"company,omitempty"`
	Campaign   string    `json:"campaign"`
	CampaignID uuid.UUID `json:"campaign_id"`
	Stage      string    `json:"stage"`
	UpdatedAt  time.Time `json:"updated_at"`
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
			out = append(out, TrackerRow{
				ID:         row.ID,
				Contact:    displayName(row.FirstName, row.LastName, row.Email),
				Email:      row.Email,
				Company:    textString(row.CompanyName),
				Campaign:   row.CampaignName,
				CampaignID: row.CampaignID,
				Stage:      TrackerStage(row.EnrolmentStatus, row.DraftStatus, row.LedgerState),
				UpdatedAt:  updated,
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
