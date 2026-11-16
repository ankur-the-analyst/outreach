package contacts

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
	"outreach/internal/db"
	"outreach/internal/suppression"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrProvenance   = errors.New("provenance required")
	ErrDuplicate    = errors.New("contact already exists")
	ErrEmail        = errors.New("email required")
	ErrCSV          = errors.New("csv required")
	ErrMapEmail     = errors.New("choose which column is email")
	ErrUnknownField = errors.New("unknown field")
)

type Provenance struct {
	Source  string    `json:"source"`
	AddedBy uuid.UUID `json:"added_by"`
	At      time.Time `json:"at"`
}

type Contact struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	FirstName   string     `json:"first_name,omitempty"`
	LastName    string     `json:"last_name,omitempty"`
	Title       string     `json:"title,omitempty"`
	CompanyID   *uuid.UUID `json:"company_id,omitempty"`
	LinkedInURL string     `json:"linkedin_url,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	Region      string     `json:"region,omitempty"`
	RoleClass   string     `json:"role_class,omitempty"`
	Provenance  Provenance `json:"provenance"`
	Readiness   string     `json:"readiness"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateInput struct {
	Email       string
	FirstName   string
	LastName    string
	Title       string
	Company     string
	LinkedInURL string
	Timezone    string
	Region      string
	Source      string
	AddedBy     uuid.UUID
	Custom      map[string]string
}

type PatchInput struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Title       string `json:"title"`
	LinkedInURL string `json:"linkedin_url"`
	Timezone    string `json:"timezone"`
	Region      string `json:"region"`
}

type Postgres struct {
	pool    *db.Pool
	hmac    suppression.Hasher
	catalog Catalog
}

func NewPostgres(pool *db.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) WithCatalog(c Catalog) *Postgres {
	p.catalog = c
	return p
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

func (p *Postgres) Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Contact, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || !strings.Contains(email, "@") {
		return Contact{}, ErrEmail
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		return Contact{}, ErrProvenance
	}
	if err := p.checkCustom(ctx, workspaceID, in.Custom); err != nil {
		return Contact{}, err
	}
	cfg, err := p.Dedup(ctx, workspaceID)
	if err != nil {
		return Contact{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Contact{}, err
	}
	role, generic := classifyEmail(email)
	ready := readiness(email, generic)
	prov := Provenance{Source: source, AddedBy: in.AddedBy, At: time.Now().UTC()}
	provJSON, err := json.Marshal(prov)
	if err != nil {
		return Contact{}, err
	}
	var out Contact
	var updated bool
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		existing, err := q.GetContactByEmail(ctx, db.GetContactByEmailParams{WorkspaceID: workspaceID, Email: email})
		if err == nil {
			if cfg.Action != "update" {
				return ErrDuplicate
			}
			out, err = p.mergeExisting(ctx, q, workspaceID, existing, in, cfg.Overwrite)
			updated = err == nil
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var companyID *uuid.UUID
		if name := strings.TrimSpace(in.Company); name != "" || domainOf(email) != "" {
			cid, err := upsertCompany(ctx, q, workspaceID, domainOf(email), name)
			if err != nil {
				return err
			}
			companyID = cid
		}
		row, err := q.InsertContact(ctx, db.InsertContactParams{
			WorkspaceID: workspaceID,
			ID:          id,
			Email:       email,
			FirstName:   text(in.FirstName),
			LastName:    text(in.LastName),
			Title:       text(in.Title),
			CompanyID:   companyID,
			LinkedinUrl: text(in.LinkedInURL),
			Timezone:    text(in.Timezone),
			Region:      text(in.Region),
			RoleClass:   text(role),
			Provenance:  provJSON,
			Readiness:   ready,
			Custom:      customJSON(in.Custom),
		})
		if err != nil {
			return err
		}
		out = contactFrom(row.ID, row.Email, row.FirstName, row.LastName, row.Title, row.CompanyID, row.LinkedinUrl, row.Timezone, row.Region, row.RoleClass, row.Provenance, row.Readiness, row.CreatedAt)
		return nil
	})
	if isUnique(err) {
		if cfg.Action == "update" {
			return p.mergeByEmail(ctx, workspaceID, in, cfg.Overwrite)
		}
		return Contact{}, ErrDuplicate
	}
	if err != nil {
		return Contact{}, err
	}
	if updated {
		return out, Updated{Contact: out}
	}
	return out, nil
}

func (p *Postgres) mergeByEmail(ctx context.Context, workspaceID uuid.UUID, in CreateInput, overwrite bool) (Contact, error) {
	var out Contact
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		existing, err := q.GetContactByEmail(ctx, db.GetContactByEmailParams{WorkspaceID: workspaceID, Email: strings.ToLower(strings.TrimSpace(in.Email))})
		if err != nil {
			return err
		}
		out, err = p.mergeExisting(ctx, q, workspaceID, existing, in, overwrite)
		return err
	})
	if err != nil {
		return Contact{}, err
	}
	return out, Updated{Contact: out}
}

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID, after *uuid.UUID, limit int) (Page, error) {
	limit = clampLimit(limit)
	var rows []Contact
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		got, err := q.ListContacts(ctx, db.ListContactsParams{
			WorkspaceID: workspaceID, After: after, N: int32(limit + 1),
		})
		if err != nil {
			return err
		}
		rows = make([]Contact, 0, len(got))
		for _, row := range got {
			rows = append(rows, contactFrom(row.ID, row.Email, row.FirstName, row.LastName, row.Title, row.CompanyID, row.LinkedinUrl, row.Timezone, row.Region, row.RoleClass, row.Provenance, row.Readiness, row.CreatedAt))
		}
		return nil
	})
	if err != nil {
		return Page{}, err
	}
	return clipPage(rows, limit), nil
}

func (p *Postgres) Get(ctx context.Context, workspaceID, id uuid.UUID) (Contact, error) {
	var out Contact
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetContact(ctx, db.GetContactParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			return err
		}
		out = contactFrom(row.ID, row.Email, row.FirstName, row.LastName, row.Title, row.CompanyID, row.LinkedinUrl, row.Timezone, row.Region, row.RoleClass, row.Provenance, row.Readiness, row.CreatedAt)
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Contact{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) Patch(ctx context.Context, workspaceID, id uuid.UUID, in PatchInput) (Contact, error) {
	var out Contact
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.UpdateContact(ctx, db.UpdateContactParams{
			WorkspaceID: workspaceID,
			ID:          id,
			FirstName:   text(in.FirstName),
			LastName:    text(in.LastName),
			Title:       text(in.Title),
			LinkedinUrl: text(in.LinkedInURL),
			Timezone:    text(in.Timezone),
			Region:      text(in.Region),
		})
		if err != nil {
			return err
		}
		out = contactFrom(row.ID, row.Email, row.FirstName, row.LastName, row.Title, row.CompanyID, row.LinkedinUrl, row.Timezone, row.Region, row.RoleClass, row.Provenance, row.Readiness, row.CreatedAt)
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Contact{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.DeleteContact(ctx, db.DeleteContactParams{WorkspaceID: workspaceID, ID: id})
	})
	return err
}

func upsertCompany(ctx context.Context, q *db.Queries, workspaceID uuid.UUID, domain, name string) (*uuid.UUID, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	name = strings.TrimSpace(name)
	if name == "" {
		name = domain
	}
	if name == "" {
		return nil, nil
	}
	if domain != "" {
		row, err := q.GetCompanyByDomain(ctx, db.GetCompanyByDomainParams{WorkspaceID: workspaceID, Domain: pgtype.Text{String: domain, Valid: true}})
		if err == nil {
			return &row.ID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	var d pgtype.Text
	if domain != "" {
		d = pgtype.Text{String: domain, Valid: true}
	}
	row, err := q.InsertCompany(ctx, db.InsertCompanyParams{
		WorkspaceID: workspaceID, ID: id, Domain: d, Name: name,
	})
	if isUnique(err) && domain != "" {
		existing, err := q.GetCompanyByDomain(ctx, db.GetCompanyByDomainParams{WorkspaceID: workspaceID, Domain: pgtype.Text{String: domain, Valid: true}})
		if err != nil {
			return nil, err
		}
		return &existing.ID, nil
	}
	if err != nil {
		return nil, err
	}
	return &row.ID, nil
}

func contactFrom(id uuid.UUID, email string, first, last, title pgtype.Text, companyID *uuid.UUID, linkedin, tz, region, role pgtype.Text, provenance json.RawMessage, readiness string, created pgtype.Timestamptz) Contact {
	var prov Provenance
	_ = json.Unmarshal(provenance, &prov)
	c := Contact{
		ID: id, Email: email, FirstName: first.String, LastName: last.String, Title: title.String,
		CompanyID: companyID, LinkedInURL: linkedin.String, Timezone: tz.String, Region: region.String,
		RoleClass: role.String, Provenance: prov, Readiness: readiness, CreatedAt: created.Time,
	}
	return c
}

func (p *Postgres) checkCustom(ctx context.Context, workspaceID uuid.UUID, m map[string]string) error {
	if len(m) == 0 {
		return nil
	}
	if p.catalog == nil {
		return ErrUnknownField
	}
	allowed, err := p.catalog.AllowedCustom(ctx, workspaceID, "contact")
	if err != nil {
		return err
	}
	for key := range m {
		if _, ok := allowed[key]; !ok {
			return ErrUnknownField
		}
	}
	return nil
}

func customJSON(m map[string]string) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage(`{}`)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

func text(s string) pgtype.Text {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
