package templates

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
)

type Template struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Subject   string          `json:"subject"`
	BodyText  string          `json:"body_text"`
	BodyHTML  string          `json:"body_html,omitempty"`
	Design    json.RawMessage `json:"design_json,omitempty"`
	CreatedAt string          `json:"created_at,omitempty"`
}

type CreateInput struct {
	Name     string
	Subject  string
	BodyText string
	BodyHTML string
	Design   json.RawMessage
}

type Postgres struct {
	pool *db.Pool
}

func NewPostgres(pool *db.Pool) *Postgres {
	return &Postgres{pool: pool}
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

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID) ([]Template, error) {
	var out []Template
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListTemplates(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]Template, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromRow(row.ID, row.Name, row.Subject, row.BodyText, row.BodyHtml, row.DesignJson, row.CreatedAt))
		}
		return nil
	})
	return out, err
}

func (p *Postgres) Get(ctx context.Context, workspaceID, id uuid.UUID) (Template, error) {
	var out Template
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetTemplate(ctx, db.GetTemplateParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			return err
		}
		out = fromRow(row.ID, row.Name, row.Subject, row.BodyText, row.BodyHtml, row.DesignJson, row.CreatedAt)
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Template, error) {
	extra, err := p.paths(ctx, workspaceID)
	if err != nil {
		return Template{}, err
	}
	name, subject, body, html, design, err := normalize(in, extra)
	if err != nil {
		return Template{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Template{}, err
	}
	var out Template
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.InsertTemplate(ctx, db.InsertTemplateParams{
			WorkspaceID: workspaceID,
			ID:          id,
			Name:        name,
			Subject:     subject,
			BodyText:    body,
			BodyHtml:    html,
			DesignJson:  design,
		})
		if err != nil {
			return err
		}
		out = fromRow(row.ID, row.Name, row.Subject, row.BodyText, row.BodyHtml, row.DesignJson, row.CreatedAt)
		return nil
	})
	return out, err
}

func (p *Postgres) Update(ctx context.Context, workspaceID, id uuid.UUID, in CreateInput) (Template, error) {
	extra, err := p.paths(ctx, workspaceID)
	if err != nil {
		return Template{}, err
	}
	name, subject, body, html, design, err := normalize(in, extra)
	if err != nil {
		return Template{}, err
	}
	var out Template
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.UpdateTemplate(ctx, db.UpdateTemplateParams{
			WorkspaceID: workspaceID,
			ID:          id,
			Name:        name,
			Subject:     subject,
			BodyText:    body,
			BodyHtml:    html,
			DesignJson:  design,
		})
		if err != nil {
			return err
		}
		out = fromRow(row.ID, row.Name, row.Subject, row.BodyText, row.BodyHtml, row.DesignJson, row.CreatedAt)
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) paths(ctx context.Context, workspaceID uuid.UUID) (map[string]struct{}, error) {
	extra := map[string]struct{}{}
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListFieldDefs(ctx, db.ListFieldDefsParams{WorkspaceID: workspaceID})
		if err != nil {
			return err
		}
		for _, row := range rows {
			switch row.Object {
			case "contact":
				extra["contact."+row.Key] = struct{}{}
			case "member":
				extra["user."+row.Key] = struct{}{}
			}
		}
		return nil
	})
	return extra, err
}

func normalize(in CreateInput, extra map[string]struct{}) (name, subject, body string, html pgtype.Text, design json.RawMessage, err error) {
	name = strings.TrimSpace(in.Name)
	subject = strings.TrimSpace(in.Subject)
	body = strings.TrimSpace(in.BodyText)
	if name == "" {
		return "", "", "", pgtype.Text{}, nil, ErrName
	}
	if subject == "" {
		return "", "", "", pgtype.Text{}, nil, ErrSubject
	}
	if body == "" {
		return "", "", "", pgtype.Text{}, nil, ErrBody
	}
	if err := ValidateAllowed(subject, extra); err != nil {
		return "", "", "", pgtype.Text{}, nil, err
	}
	if err := ValidateAllowed(body, extra); err != nil {
		return "", "", "", pgtype.Text{}, nil, err
	}
	if h := strings.TrimSpace(in.BodyHTML); h != "" {
		if err := ValidateAllowed(h, extra); err != nil {
			return "", "", "", pgtype.Text{}, nil, err
		}
	}
	if err := validateJSON(in.Design, extra); err != nil {
		return "", "", "", pgtype.Text{}, nil, err
	}
	return name, subject, body, text(in.BodyHTML), designOrEmpty(in.Design), nil
}

func designOrEmpty(raw json.RawMessage) json.RawMessage {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage(`{}`)
	}
	return raw
}

func fromRow(id uuid.UUID, name, subject, body string, html pgtype.Text, design json.RawMessage, created pgtype.Timestamptz) Template {
	t := Template{ID: id, Name: name, Subject: subject, BodyText: body, BodyHTML: html.String, Design: design}
	if created.Valid {
		t.CreatedAt = created.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return t
}

func text(s string) pgtype.Text {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
