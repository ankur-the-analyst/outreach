package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
	"outreach/internal/mime"
	"outreach/internal/pacing"
)

var ErrOutsideWindow = errors.New("outside send window")

type OutsideWindowError struct {
	Until time.Time
}

func (e *OutsideWindowError) Error() string { return ErrOutsideWindow.Error() }

func (e *OutsideWindowError) Unwrap() error { return ErrOutsideWindow }

type Draft struct {
	ID      uuid.UUID `json:"id"`
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body_text"`
	Status  string    `json:"status"`
}

func (p *Postgres) DraftDueStep(ctx context.Context, workspaceID, enrolmentID uuid.UUID, stepPosition int32, now time.Time) (Draft, error) {
	var out Draft
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		en, err := q.GetEnrolment(ctx, db.GetEnrolmentParams{WorkspaceID: workspaceID, ID: enrolmentID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if en.CurrentStep != stepPosition {
			return nil
		}
		switch en.Status {
		case "stopped", "completed", "failed", "unsubscribed", "bounced", "replied":
			return nil
		}
		existing, err := q.GetDraftByEnrolmentStep(ctx, db.GetDraftByEnrolmentStepParams{
			WorkspaceID:  workspaceID,
			EnrolmentID:  &enrolmentID,
			StepPosition: pgtype.Int4{Int32: stepPosition, Valid: true},
		})
		if err == nil {
			out = Draft{ID: existing.ID, To: existing.ToAddr, Subject: existing.Subject, Body: existing.BodyText, Status: existing.Status}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		camp, err := loadCampaign(ctx, q, workspaceID, en.CampaignID, p.packs)
		if err != nil {
			return err
		}
		if camp.Status == "stopped" || camp.Status == "completed" {
			return nil
		}
		if camp.Status != "running" {
			return fmt.Errorf("campaign: not running")
		}
		ct, err := q.GetContact(ctx, db.GetContactParams{WorkspaceID: workspaceID, ID: en.ContactID})
		if err != nil {
			return err
		}
		mailboxID := en.MailboxID
		if mailboxID == nil {
			if len(camp.Senders) == 0 {
				return ErrMailbox
			}
			id := camp.Senders[0].MailboxID
			mailboxID = &id
		}
		win, err := q.GetMailboxWindow(ctx, db.GetMailboxWindowParams{WorkspaceID: workspaceID, ID: *mailboxID})
		if err != nil {
			return err
		}
		zone := win.Timezone
		if ct.Timezone.Valid && strings.TrimSpace(ct.Timezone.String) != "" {
			zone = ct.Timezone.String
		}
		w := pacing.WindowFrom(win.WindowDays, win.WindowStart, win.WindowEnd, pacing.LoadZone(zone))
		if !pacing.InWindow(now, w) {
			return &OutsideWindowError{Until: pacing.NextOpen(now, w)}
		}
		msgid, err := mime.NewMessageID("outreach.test")
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		body := notesFromContext(en.Context)
		row, err := q.InsertDraft(ctx, db.InsertDraftParams{
			WorkspaceID:  workspaceID,
			ID:           id,
			EnrolmentID:  &enrolmentID,
			StepPosition: pgtype.Int4{Int32: stepPosition, Valid: true},
			MailboxID:    *mailboxID,
			ToAddr:       ct.Email,
			RfcMessageID: msgid,
			Kind:         "outbound",
			Subject:      camp.Name,
			BodyHtml:     "",
			BodyText:     body,
			FactsUsed:    json.RawMessage("[]"),
			Status:       "pending",
		})
		if err != nil {
			return fmt.Errorf("campaign: draft step: %w", err)
		}
		if _, err := q.UpdateEnrolmentStatus(ctx, db.UpdateEnrolmentStatusParams{
			WorkspaceID: workspaceID, ID: enrolmentID, Status: "waiting_review",
		}); err != nil {
			return err
		}
		out = Draft{ID: row.ID, To: row.ToAddr, Subject: row.Subject, Body: row.BodyText, Status: row.Status}
		return nil
	})
	return out, err
}

func notesFromContext(raw json.RawMessage) string {
	var bag struct {
		Notes string `json:"notes"`
	}
	_ = json.Unmarshal(raw, &bag)
	notes := strings.TrimSpace(bag.Notes)
	if notes == "" {
		return "Following up."
	}
	return notes
}
