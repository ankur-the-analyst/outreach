package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/ai"
	"outreach/internal/compose"
	"outreach/internal/cv"
	"outreach/internal/db"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrPack      = errors.New("unknown campaign type")
	ErrIdentity  = errors.New("sender identity required")
	ErrContext   = errors.New("add a note, a job, or campaign instructions")
	ErrDuplicate = errors.New("already in this campaign")
	ErrStatus    = errors.New("invalid campaign status")
	ErrName      = errors.New("name required")
	ErrMailbox   = errors.New("mailbox required")
	ErrFrom      = errors.New("add a mailbox to this campaign or group")
	ErrStage     = errors.New("pick a valid stage")
	ErrTemplate  = errors.New("pick a saved email")
)

type Identity struct {
	LegalName     string `json:"legal_name"`
	ReplyAddress  string `json:"reply_address"`
	PostalAddress string `json:"postal_address"`
}

type Step struct {
	ID         uuid.UUID  `json:"id"`
	Position   int32      `json:"position"`
	Kind       string     `json:"kind"`
	WaitDays   *int32     `json:"wait_days,omitempty"`
	Content    string     `json:"content,omitempty"`
	TemplateID *uuid.UUID `json:"template_id,omitempty"`
}

type Sender struct {
	MailboxID uuid.UUID `json:"mailbox_id"`
	Position  int32     `json:"position"`
	Weight    int32     `json:"weight"`
}

type Campaign struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	ProfileID         *uuid.UUID `json:"profile_id,omitempty"`
	Intent            string     `json:"intent"`
	SenderStrategy    string     `json:"sender_strategy"`
	Status            string     `json:"status"`
	ListUnsubscribe   bool       `json:"list_unsubscribe"`
	NewContactsPerDay int32      `json:"new_contacts_per_day"`
	AIInstructions    string     `json:"ai_instructions,omitempty"`
	UndoHoldS         int32      `json:"undo_hold_s"`
	Commercial        bool       `json:"commercial"`
	Steps             []Step     `json:"steps"`
	Senders           []Sender   `json:"senders"`
	PoolID            *uuid.UUID `json:"pool_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type Enrolment struct {
	ID         uuid.UUID       `json:"id"`
	CampaignID uuid.UUID       `json:"campaign_id"`
	ContactID  uuid.UUID       `json:"contact_id"`
	JobID      *uuid.UUID      `json:"job_id,omitempty"`
	ProfileID  *uuid.UUID      `json:"profile_id,omitempty"`
	Intent     string          `json:"intent"`
	Context    json.RawMessage `json:"context"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
}

type CreateInput struct {
	Name              string
	Intent            string
	SenderStrategy    string
	MailboxIDs        []uuid.UUID
	PoolID            *uuid.UUID
	NewContactsPerDay int32
	AIInstructions    string
	ProfileID         *uuid.UUID
	TemplateID        *uuid.UUID
}

type EnrolInput struct {
	ContactID uuid.UUID
	JobID     *uuid.UUID
	ProfileID *uuid.UUID
	Notes     string
}

type Binder interface {
	Provider(ctx context.Context, workspaceID uuid.UUID, role ai.Role) (ai.Provider, error)
}

type Stamper interface {
	Stamp(ctx context.Context, q *db.Queries, workspaceID, profileID, mailboxID uuid.UUID) (cv.Render, error)
}

type Postgres struct {
	pool     *db.Pool
	packs    map[string]Pack
	retrieve Retriever
	write    compose.Completer
	bind     Binder
	stamp    Stamper
}

func NewPostgres(pool *db.Pool, packs map[string]Pack) *Postgres {
	return &Postgres{pool: pool, packs: packs}
}

func (p *Postgres) WithRetrieve(r Retriever) *Postgres {
	p.retrieve = r
	return p
}

func (p *Postgres) WithWriter(w compose.Completer) *Postgres {
	p.write = w
	return p
}

func (p *Postgres) WithBinder(b Binder) *Postgres {
	p.bind = b
	return p
}

func (p *Postgres) WithStamp(s Stamper) *Postgres {
	p.stamp = s
	return p
}

func (p *Postgres) writer(ctx context.Context, workspaceID uuid.UUID) (compose.Completer, error) {
	if p.bind == nil {
		return p.write, nil
	}
	got, err := p.bind.Provider(ctx, workspaceID, ai.RoleCompose)
	if err != nil {
		return nil, err
	}
	return got, nil
}

func (p *Postgres) Packs() []Pack {
	out := make([]Pack, 0, len(p.packs))
	for _, pack := range p.packs {
		out = append(out, pack)
	}
	return out
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

func (p *Postgres) PutIdentity(ctx context.Context, workspaceID uuid.UUID, in Identity) (Identity, error) {
	in.LegalName = strings.TrimSpace(in.LegalName)
	in.ReplyAddress = strings.TrimSpace(in.ReplyAddress)
	in.PostalAddress = strings.TrimSpace(in.PostalAddress)
	if in.LegalName == "" || in.ReplyAddress == "" || in.PostalAddress == "" {
		return Identity{}, ErrIdentity
	}
	var out Identity
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.UpsertSenderIdentity(ctx, db.UpsertSenderIdentityParams{
			WorkspaceID: workspaceID, LegalName: in.LegalName, ReplyAddress: in.ReplyAddress, PostalAddress: in.PostalAddress,
		})
		if err != nil {
			return err
		}
		out = Identity{LegalName: row.LegalName, ReplyAddress: row.ReplyAddress, PostalAddress: row.PostalAddress}
		return nil
	})
	return out, err
}

func (p *Postgres) GetIdentity(ctx context.Context, workspaceID uuid.UUID) (Identity, error) {
	var out Identity
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetSenderIdentity(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = Identity{LegalName: row.LegalName, ReplyAddress: row.ReplyAddress, PostalAddress: row.PostalAddress}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Campaign, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Campaign{}, ErrName
	}
	pack, ok := p.packs[strings.TrimSpace(in.Intent)]
	if !ok {
		return Campaign{}, ErrPack
	}
	strategy := in.SenderStrategy
	if strategy == "" {
		strategy = "round_robin"
	}
	perDay := in.NewContactsPerDay
	if perDay <= 0 {
		perDay = 5
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Campaign{}, err
	}
	var out Campaign
	err = p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		ids := in.MailboxIDs
		if in.PoolID != nil {
			if _, err := q.GetMailboxPool(ctx, db.GetMailboxPoolParams{WorkspaceID: workspaceID, ID: *in.PoolID}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrNotFound
				}
				return err
			}
			members, err := q.ListMailboxPoolMembersForPool(ctx, db.ListMailboxPoolMembersForPoolParams{
				WorkspaceID: workspaceID, PoolID: *in.PoolID,
			})
			if err != nil {
				return err
			}
			ids = make([]uuid.UUID, 0, len(members))
			for _, m := range members {
				ids = append(ids, m.MailboxID)
			}
		} else if len(ids) == 0 {
			return ErrMailbox
		}
		row, err := q.InsertCampaign(ctx, db.InsertCampaignParams{
			WorkspaceID:       workspaceID,
			ID:                id,
			Name:              name,
			ProfileID:         in.ProfileID,
			Intent:            pack.Intent,
			SenderStrategy:    strategy,
			Status:            "draft",
			ListUnsubscribe:   pack.CommercialStart(),
			NewContactsPerDay: perDay,
			AiInstructions:    text(in.AIInstructions),
			UndoHoldS:         300,
			PoolID:            in.PoolID,
		})
		if err != nil {
			if isFK(err) {
				return ErrNotFound
			}
			return err
		}
		for i, mb := range ids {
			if err := q.InsertCampaignSender(ctx, db.InsertCampaignSenderParams{
				WorkspaceID: workspaceID, CampaignID: id, MailboxID: mb, Position: int32(i), Weight: 1,
			}); err != nil {
				return err
			}
		}
		if in.TemplateID != nil {
			if _, err := q.GetTemplate(ctx, db.GetTemplateParams{WorkspaceID: workspaceID, ID: *in.TemplateID}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrTemplate
				}
				return err
			}
		}
		applied := false
		for _, st := range pack.Steps {
			sid, err := uuid.NewV7()
			if err != nil {
				return err
			}
			var wait pgtype.Int4
			if st.Kind == "wait" && st.WaitDays > 0 {
				wait = pgtype.Int4{Int32: int32(st.WaitDays), Valid: true}
			}
			content := st.Content
			var tid *uuid.UUID
			if st.Kind == "email" && in.TemplateID != nil && !applied {
				content = "template"
				tid = in.TemplateID
				applied = true
			}
			if _, err := q.InsertCampaignStep(ctx, db.InsertCampaignStepParams{
				WorkspaceID: workspaceID, ID: sid, CampaignID: id, Position: int32(st.Position),
				Kind: st.Kind, WaitDays: wait, Content: text(content), TemplateID: tid,
			}); err != nil {
				return err
			}
		}
		loaded, err := loadCampaign(ctx, q, workspaceID, row.ID, p.packs)
		if err != nil {
			return err
		}
		out = loaded
		return nil
	})
	return out, err
}

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID, after *uuid.UUID, limit int) (Page, error) {
	limit = clampLimit(limit)
	var rows []Campaign
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		got, err := q.ListCampaigns(ctx, db.ListCampaignsParams{
			WorkspaceID: workspaceID, After: after, N: int32(limit + 1),
		})
		if err != nil {
			return err
		}
		rows = make([]Campaign, 0, len(got))
		for _, row := range got {
			c, err := loadCampaign(ctx, q, workspaceID, row.ID, p.packs)
			if err != nil {
				return err
			}
			rows = append(rows, c)
		}
		return nil
	})
	if err != nil {
		return Page{}, err
	}
	return clipPage(rows, limit), nil
}

func (p *Postgres) Get(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error) {
	var out Campaign
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		c, err := loadCampaign(ctx, q, workspaceID, id, p.packs)
		if err != nil {
			return err
		}
		out = c
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) Start(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error) {
	c, err := p.Get(ctx, workspaceID, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status != "draft" && c.Status != "paused" {
		return Campaign{}, ErrStatus
	}
	if err := p.ensureFrom(ctx, workspaceID, &c); err != nil {
		return Campaign{}, err
	}
	if c.Commercial {
		if _, err := p.GetIdentity(ctx, workspaceID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return Campaign{}, ErrIdentity
			}
			return Campaign{}, err
		}
	}
	return p.setStatus(ctx, workspaceID, id, "running", c.Commercial || c.ListUnsubscribe)
}

func (p *Postgres) ensureFrom(ctx context.Context, workspaceID uuid.UUID, c *Campaign) error {
	if len(c.Senders) > 0 {
		return nil
	}
	if c.PoolID == nil {
		return ErrFrom
	}
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		members, err := q.ListMailboxPoolMembersForPool(ctx, db.ListMailboxPoolMembersForPoolParams{
			WorkspaceID: workspaceID, PoolID: *c.PoolID,
		})
		if err != nil {
			return err
		}
		if len(members) == 0 {
			return ErrFrom
		}
		for i, m := range members {
			if err := q.InsertCampaignSender(ctx, db.InsertCampaignSenderParams{
				WorkspaceID: workspaceID, CampaignID: c.ID, MailboxID: m.MailboxID, Position: int32(i), Weight: 1,
			}); err != nil {
				if isUnique(err) {
					continue
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	loaded, err := p.Get(ctx, workspaceID, c.ID)
	if err != nil {
		return err
	}
	if len(loaded.Senders) == 0 {
		return ErrFrom
	}
	*c = loaded
	return nil
}

func (p *Postgres) Pause(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error) {
	c, err := p.Get(ctx, workspaceID, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status != "running" {
		return Campaign{}, ErrStatus
	}
	return p.setStatus(ctx, workspaceID, id, "paused", c.ListUnsubscribe)
}

func (p *Postgres) Stop(ctx context.Context, workspaceID, id uuid.UUID) (Campaign, error) {
	c, err := p.Get(ctx, workspaceID, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status != "running" && c.Status != "paused" {
		return Campaign{}, ErrStatus
	}
	return p.setStatus(ctx, workspaceID, id, "stopped", c.ListUnsubscribe)
}

func (p *Postgres) setStatus(ctx context.Context, workspaceID, id uuid.UUID, status string, unsub bool) (Campaign, error) {
	var out Campaign
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		_, err := q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
			WorkspaceID:     workspaceID,
			ID:              id,
			Status:          status,
			ListUnsubscribe: unsub,
			StartsAt:        pgtype.Timestamptz{Time: time.Now().UTC(), Valid: status == "running"},
		})
		if err != nil {
			return err
		}
		c, err := loadCampaign(ctx, q, workspaceID, id, p.packs)
		if err != nil {
			return err
		}
		out = c
		return nil
	})
	return out, err
}

func (p *Postgres) Enrol(ctx context.Context, workspaceID, campaignID uuid.UUID, in EnrolInput) (Enrolment, error) {
	c, err := p.Get(ctx, workspaceID, campaignID)
	if err != nil {
		return Enrolment{}, err
	}
	notes := strings.TrimSpace(in.Notes)
	if notes == "" && in.JobID == nil && strings.TrimSpace(c.AIInstructions) == "" {
		return Enrolment{}, ErrContext
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Enrolment{}, err
	}
	ctxJSON, err := json.Marshal(map[string]string{"notes": notes})
	if err != nil {
		return Enrolment{}, err
	}
	appID, err := uuid.NewV7()
	if err != nil {
		return Enrolment{}, err
	}
	var out Enrolment
	err = p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetContact(ctx, db.GetContactParams{WorkspaceID: workspaceID, ID: in.ContactID}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		row, err := q.InsertEnrolment(ctx, db.InsertEnrolmentParams{
			WorkspaceID: workspaceID,
			ID:          id,
			CampaignID:  campaignID,
			ContactID:   in.ContactID,
			JobID:       in.JobID,
			ProfileID:   in.ProfileID,
			Intent:      c.Intent,
			Context:     ctxJSON,
			Status:      "pending",
		})
		if err != nil {
			return err
		}
		if _, err := q.InsertApplication(ctx, db.InsertApplicationParams{
			WorkspaceID: workspaceID,
			ID:          appID,
			EnrolmentID: id,
			ContactID:   in.ContactID,
			JobID:       in.JobID,
			ProfileID:   in.ProfileID,
			Stage:       "drafted",
		}); err != nil {
			return err
		}
		out = enrolmentFrom(row.ID, row.CampaignID, row.ContactID, row.JobID, row.ProfileID, row.Intent, row.Context, row.Status, row.CreatedAt)
		return nil
	})
	if isUnique(err) {
		return Enrolment{}, ErrDuplicate
	}
	return out, err
}

func (p *Postgres) ListEnrolments(ctx context.Context, workspaceID, campaignID uuid.UUID) ([]Enrolment, error) {
	var out []Enrolment
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListEnrolments(ctx, db.ListEnrolmentsParams{WorkspaceID: workspaceID, CampaignID: campaignID})
		if err != nil {
			return err
		}
		out = make([]Enrolment, 0, len(rows))
		for _, row := range rows {
			out = append(out, enrolmentFrom(row.ID, row.CampaignID, row.ContactID, row.JobID, row.ProfileID, row.Intent, row.Context, row.Status, row.CreatedAt))
		}
		return nil
	})
	return out, err
}

func loadCampaign(ctx context.Context, q *db.Queries, workspaceID, id uuid.UUID, packs map[string]Pack) (Campaign, error) {
	row, err := q.GetCampaign(ctx, db.GetCampaignParams{WorkspaceID: workspaceID, ID: id})
	if err != nil {
		return Campaign{}, err
	}
	steps, err := q.ListCampaignSteps(ctx, db.ListCampaignStepsParams{WorkspaceID: workspaceID, CampaignID: id})
	if err != nil {
		return Campaign{}, err
	}
	senders, err := q.ListCampaignSenders(ctx, db.ListCampaignSendersParams{WorkspaceID: workspaceID, CampaignID: id})
	if err != nil {
		return Campaign{}, err
	}
	c := Campaign{
		ID: row.ID, Name: row.Name, ProfileID: row.ProfileID, Intent: row.Intent, SenderStrategy: row.SenderStrategy,
		Status: row.Status, ListUnsubscribe: row.ListUnsubscribe, NewContactsPerDay: row.NewContactsPerDay,
		AIInstructions: row.AiInstructions.String, UndoHoldS: row.UndoHoldS, PoolID: row.PoolID, CreatedAt: row.CreatedAt.Time,
		Steps: make([]Step, 0, len(steps)), Senders: make([]Sender, 0, len(senders)),
	}
	if pack, ok := packs[row.Intent]; ok {
		c.Commercial = pack.CommercialStart()
	}
	for _, st := range steps {
		step := Step{ID: st.ID, Position: st.Position, Kind: st.Kind, Content: st.Content.String, TemplateID: st.TemplateID}
		if st.WaitDays.Valid {
			d := st.WaitDays.Int32
			step.WaitDays = &d
		}
		c.Steps = append(c.Steps, step)
	}
	for _, s := range senders {
		c.Senders = append(c.Senders, Sender{MailboxID: s.MailboxID, Position: s.Position, Weight: s.Weight})
	}
	return c, nil
}

type Application struct {
	ID             uuid.UUID `json:"id"`
	Stage          string    `json:"stage"`
	StageChangedAt time.Time `json:"stage_changed_at"`
}

func StageOK(s string) bool {
	switch s {
	case "drafted", "sent", "followed_up", "replied", "interview", "offer", "rejected", "closed":
		return true
	default:
		return false
	}
}

func (p *Postgres) SetStage(ctx context.Context, workspaceID, id uuid.UUID, stage string) (Application, error) {
	if !StageOK(stage) {
		return Application{}, ErrStage
	}
	var out Application
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.UpdateApplicationStage(ctx, db.UpdateApplicationStageParams{
			WorkspaceID: workspaceID, ID: id, Stage: stage,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		out = Application{ID: row.ID, Stage: row.Stage, StageChangedAt: row.StageChangedAt.Time}
		return nil
	})
	return out, err
}

func enrolmentFrom(id, campaignID, contactID uuid.UUID, jobID, profileID *uuid.UUID, intent string, context json.RawMessage, status string, created pgtype.Timestamptz) Enrolment {
	return Enrolment{
		ID: id, CampaignID: campaignID, ContactID: contactID, JobID: jobID, ProfileID: profileID,
		Intent: intent, Context: context, Status: status, CreatedAt: created.Time,
	}
}

func text(s string) pgtype.Text {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func isFK(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
