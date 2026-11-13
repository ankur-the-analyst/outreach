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
	"outreach/internal/compose"
	"outreach/internal/cv"
	"outreach/internal/db"
	"outreach/internal/evidence"
	"outreach/internal/mime"
	"outreach/internal/pacing"
	"outreach/internal/templates"
)

type Retriever interface {
	Retrieve(ctx context.Context, workspaceID uuid.UUID, profileID *uuid.UUID, query string) ([]evidence.Item, error)
}

var ErrOutsideWindow = errors.New("outside send window")

type OutsideWindowError struct {
	Until time.Time
}

func (e *OutsideWindowError) Error() string { return ErrOutsideWindow.Error() }

func (e *OutsideWindowError) Unwrap() error { return ErrOutsideWindow }

func (e *OutsideWindowError) RetryAt() time.Time { return e.Until }

var ErrDailyCap = errors.New("daily new-contact cap reached")

type DailyCapError struct {
	Until time.Time
}

func (e *DailyCapError) Error() string { return ErrDailyCap.Error() }

func (e *DailyCapError) Unwrap() error { return ErrDailyCap }

func (e *DailyCapError) RetryAt() time.Time { return e.Until }

func RetryAt(err error) (time.Time, bool) {
	var r interface{ RetryAt() time.Time }
	if errors.As(err, &r) {
		return r.RetryAt(), true
	}
	return time.Time{}, false
}

var ErrStickyDown = errors.New("follow-up mailbox unavailable")

type StickyDownError struct {
	Until time.Time
}

func (e *StickyDownError) Error() string { return ErrStickyDown.Error() }

func (e *StickyDownError) Unwrap() error { return ErrStickyDown }

func (e *StickyDownError) RetryAt() time.Time { return e.Until }

type Draft struct {
	ID        uuid.UUID   `json:"id"`
	MailboxID uuid.UUID   `json:"mailbox_id"`
	To        string      `json:"to"`
	Subject   string      `json:"subject"`
	Body      string      `json:"body_text"`
	BodyHTML  string      `json:"body_html,omitempty"`
	Status    string      `json:"status"`
	FactsUsed []draftFact `json:"facts_used,omitempty"`
}

type draftFact struct {
	Claim      string    `json:"claim"`
	EvidenceID uuid.UUID `json:"evidence_id"`
}

func (p *Postgres) DraftDueStep(ctx context.Context, workspaceID, enrolmentID uuid.UUID, stepPosition int32, now time.Time) (Draft, error) {
	var out Draft
	err := p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
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
			out = Draft{
				ID: existing.ID, MailboxID: existing.MailboxID, To: existing.ToAddr,
				Subject: existing.Subject, Body: existing.BodyText, BodyHTML: existing.BodyHtml, Status: existing.Status,
			}
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
		if mailboxID == nil && stepPosition > 0 {
			prev, err := q.GetDraftByEnrolmentStep(ctx, db.GetDraftByEnrolmentStepParams{
				WorkspaceID:  workspaceID,
				EnrolmentID:  &enrolmentID,
				StepPosition: pgtype.Int4{Int32: 0, Valid: true},
			})
			if err == nil {
				id := prev.MailboxID
				mailboxID = &id
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		if mailboxID == nil {
			if stepPosition > 0 {
				return ErrMailbox
			}
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
		if stepPosition > 0 && !pacing.MailboxReady(win.Status) {
			return &StickyDownError{Until: now.UTC().Add(15 * time.Minute)}
		}
		zone := win.Timezone
		if ct.Timezone.Valid && strings.TrimSpace(ct.Timezone.String) != "" {
			zone = ct.Timezone.String
		}
		w := pacing.WindowFrom(win.WindowDays, win.WindowStart, win.WindowEnd, pacing.LoadZone(zone))
		if !pacing.InWindow(now, w) {
			return &OutsideWindowError{Until: pacing.NextOpen(now, w)}
		}
		if stepPosition == 0 {
			if _, err := q.LockCampaign(ctx, db.LockCampaignParams{WorkspaceID: workspaceID, ID: camp.ID}); err != nil {
				return err
			}
			from, to := pacing.DayBounds(now, pacing.LoadZone(win.Timezone))
			n, err := q.CountCampaignFirstTouchToday(ctx, db.CountCampaignFirstTouchTodayParams{
				WorkspaceID: workspaceID,
				CampaignID:  camp.ID,
				FromAt:      pgtype.Timestamptz{Time: from, Valid: true},
				ToAt:        pgtype.Timestamptz{Time: to, Valid: true},
			})
			if err != nil {
				return err
			}
			if n >= int64(camp.NewContactsPerDay) {
				return &DailyCapError{Until: to}
			}
		}
		msgid, err := mime.NewMessageID("outreach.test")
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		notes := notesFromContext(en.Context)
		profileID := en.ProfileID
		if profileID == nil {
			profileID = camp.ProfileID
		}
		cited, facts, err := retrieveFacts(ctx, p.retrieve, workspaceID, profileID, notes)
		if err != nil {
			return err
		}
		subject := camp.Name
		body := ""
		html := ""
		var step *Step
		for i := range camp.Steps {
			if camp.Steps[i].Position == stepPosition {
				step = &camp.Steps[i]
				break
			}
		}
		if step != nil && step.Content == "template" && step.TemplateID != nil {
			tpl, err := q.GetTemplate(ctx, db.GetTemplateParams{WorkspaceID: workspaceID, ID: *step.TemplateID})
			if err != nil {
				return err
			}
			vals := fillVars(ctx, q, workspaceID, ct, en, camp)
			subject = templates.Render(tpl.Subject, vals)
			body = templates.Render(tpl.BodyText, vals)
			if tpl.BodyHtml.Valid {
				html = templates.Render(tpl.BodyHtml.String, vals)
			}
			if strings.TrimSpace(subject) == "" {
				subject = camp.Name
			}
		} else {
			body, err = p.writeBody(ctx, workspaceID, notes, cited, facts)
			if err != nil {
				return err
			}
		}
		factsJSON, err := json.Marshal(facts)
		if err != nil {
			return err
		}
		row, err := q.InsertDraft(ctx, db.InsertDraftParams{
			WorkspaceID:  workspaceID,
			ID:           id,
			EnrolmentID:  &enrolmentID,
			StepPosition: pgtype.Int4{Int32: stepPosition, Valid: true},
			MailboxID:    *mailboxID,
			ToAddr:       ct.Email,
			RfcMessageID: msgid,
			Kind:         "outbound",
			Subject:      subject,
			BodyHtml:     html,
			BodyText:     body,
			FactsUsed:    factsJSON,
			Status:       "pending",
			CreatedAt:    pgtype.Timestamptz{Time: now.UTC(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("campaign: draft step: %w", err)
		}
		if err := q.UpdateEnrolmentMailbox(ctx, db.UpdateEnrolmentMailboxParams{
			WorkspaceID: workspaceID, ID: enrolmentID, MailboxID: mailboxID,
		}); err != nil {
			return err
		}
		if _, err := q.UpdateEnrolmentStatus(ctx, db.UpdateEnrolmentStatusParams{
			WorkspaceID: workspaceID, ID: enrolmentID, Status: "waiting_review",
		}); err != nil {
			return err
		}
		switch camp.Attach {
		case "original":
			if profileID != nil {
				res, err := q.LatestResume(ctx, db.LatestResumeParams{WorkspaceID: workspaceID, ProfileID: *profileID})
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					return err
				}
				if err == nil {
					if err := q.UpdateDraftAttachment(ctx, db.UpdateDraftAttachmentParams{
						WorkspaceID: workspaceID, ID: row.ID, AttachmentResumeID: &res.ID,
					}); err != nil {
						return err
					}
				}
			}
		case "render":
			if p.stamp != nil && profileID != nil {
				jd := ""
				if en.JobID != nil {
					job, err := q.GetJob(ctx, db.GetJobParams{WorkspaceID: workspaceID, ID: *en.JobID})
					if err == nil {
						jd = job.RawText
					}
				}
				r, err := p.stamp.Attach(ctx, q, workspaceID, *profileID, *mailboxID, en.JobID, cited, jd)
				if err != nil && !errors.Is(err, cv.ErrNoCV) && !errors.Is(err, cv.ErrInvented) && !errors.Is(err, evidence.ErrClaim) {
					return err
				}
				if err == nil {
					if err := q.UpdateDraftAttachment(ctx, db.UpdateDraftAttachmentParams{
						WorkspaceID: workspaceID, ID: row.ID, AttachmentRenderID: &r.ID,
					}); err != nil {
						return err
					}
				}
			}
		}
		out = Draft{
			ID: row.ID, MailboxID: row.MailboxID, To: row.ToAddr, Subject: row.Subject,
			Body: row.BodyText, BodyHTML: row.BodyHtml, Status: row.Status, FactsUsed: facts,
		}
		return nil
	})
	return out, err
}

func factsFromRetrieve(ctx context.Context, r Retriever, workspaceID uuid.UUID, profileID *uuid.UUID, query string) ([]draftFact, error) {
	_, facts, err := retrieveFacts(ctx, r, workspaceID, profileID, query)
	return facts, err
}

func retrieveFacts(ctx context.Context, r Retriever, workspaceID uuid.UUID, profileID *uuid.UUID, query string) ([]evidence.Item, []draftFact, error) {
	if r == nil {
		return nil, []draftFact{}, nil
	}
	items, err := r.Retrieve(ctx, workspaceID, profileID, query)
	if err != nil {
		return nil, nil, err
	}
	out := make([]draftFact, 0, len(items))
	for _, it := range items {
		out = append(out, draftFact{Claim: it.Text, EvidenceID: it.ID})
	}
	return items, out, nil
}

func (p *Postgres) writeBody(ctx context.Context, workspaceID uuid.UUID, notes string, cited []evidence.Item, facts []draftFact) (string, error) {
	if len(cited) == 0 {
		return notes, nil
	}
	w, err := p.writer(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	if w == nil {
		return notes, nil
	}
	cf := make([]compose.Fact, 0, len(facts))
	for _, f := range facts {
		cf = append(cf, compose.Fact{Claim: f.Claim, EvidenceID: f.EvidenceID})
	}
	body, _, _, err := compose.Generate(ctx, w, cited, cf)
	if err != nil {
		return "", err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return notes, nil
	}
	return body, nil
}

func fillVars(ctx context.Context, q *db.Queries, workspaceID uuid.UUID, ct db.GetContactRow, en db.GetEnrolmentRow, camp Campaign) map[string]string {
	vals := map[string]string{
		"contact.first_name": ct.FirstName.String,
		"contact.last_name":  ct.LastName.String,
		"contact.email":      ct.Email,
		"contact.title":      ct.Title.String,
	}
	if ct.CompanyID != nil {
		co, err := q.GetCompany(ctx, db.GetCompanyParams{WorkspaceID: workspaceID, ID: *ct.CompanyID})
		if err == nil {
			vals["company.name"] = co.Name
		}
	}
	if en.JobID != nil {
		job, err := q.GetJob(ctx, db.GetJobParams{WorkspaceID: workspaceID, ID: *en.JobID})
		if err == nil {
			vals["job.title"] = job.Title.String
		}
	}
	profileID := en.ProfileID
	if profileID == nil {
		profileID = camp.ProfileID
	}
	if profileID != nil {
		pr, err := q.GetProfile(ctx, db.GetProfileParams{WorkspaceID: workspaceID, ID: *profileID})
		if err == nil {
			vals["profile.name"] = pr.Name
		}
	}
	return vals
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
