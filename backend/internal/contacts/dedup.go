package contacts

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"outreach/internal/db"
)

// Dedup is the workspace rule for an email that is already saved.
// Action is "prevent" or "update". Overwrite replaces filled values when updating.
type Dedup struct {
	Action    string `json:"action"`
	Overwrite bool   `json:"overwrite"`
}

// Updated is returned from Create when the existing person was filled in.
type Updated struct {
	Contact Contact
}

func (Updated) Error() string { return "contact updated" }

func parseDedup(raw json.RawMessage) Dedup {
	d := Dedup{Action: "prevent"}
	if len(raw) == 0 {
		return d
	}
	_ = json.Unmarshal(raw, &d)
	if d.Action != "update" {
		d.Action = "prevent"
		d.Overwrite = false
	}
	return d
}

func (p *Postgres) Dedup(ctx context.Context, workspaceID uuid.UUID) (Dedup, error) {
	var raw json.RawMessage
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetContactDedup(ctx, workspaceID)
		if err != nil {
			return err
		}
		raw = row
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Dedup{Action: "prevent"}, nil
	}
	if err != nil {
		return Dedup{}, err
	}
	return parseDedup(raw), nil
}

func (p *Postgres) SetDedup(ctx context.Context, workspaceID uuid.UUID, in Dedup) (Dedup, error) {
	in = parseDedup(mustJSON(in))
	raw, err := json.Marshal(in)
	if err != nil {
		return Dedup{}, err
	}
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.SetContactDedup(ctx, db.SetContactDedupParams{ID: workspaceID, ContactDedup: raw})
	})
	if err != nil {
		return Dedup{}, err
	}
	return in, nil
}

func mustJSON(d Dedup) json.RawMessage {
	b, _ := json.Marshal(d)
	return b
}

func fill(old, next string, overwrite bool) string {
	next = strings.TrimSpace(next)
	old = strings.TrimSpace(old)
	if overwrite && next != "" {
		return next
	}
	if old == "" {
		return next
	}
	return old
}

func mergeCustom(oldRaw json.RawMessage, next map[string]string, overwrite bool) map[string]string {
	old := map[string]string{}
	if len(oldRaw) > 0 {
		_ = json.Unmarshal(oldRaw, &old)
	}
	out := map[string]string{}
	for k, v := range old {
		out[k] = v
	}
	for k, v := range next {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if overwrite || strings.TrimSpace(out[k]) == "" {
			out[k] = v
		}
	}
	return out
}

func (p *Postgres) mergeExisting(ctx context.Context, q *db.Queries, workspaceID uuid.UUID, existing db.GetContactByEmailRow, in CreateInput, overwrite bool) (Contact, error) {
	var companyID *uuid.UUID
	if (overwrite || existing.CompanyID == nil) && (strings.TrimSpace(in.Company) != "" || domainOf(in.Email) != "") {
		cid, err := upsertCompany(ctx, q, workspaceID, domainOf(in.Email), strings.TrimSpace(in.Company))
		if err != nil {
			return Contact{}, err
		}
		companyID = cid
	}
	row, err := q.MergeContact(ctx, db.MergeContactParams{
		WorkspaceID: workspaceID,
		ID:          existing.ID,
		FirstName:   text(fill(existing.FirstName.String, in.FirstName, overwrite)),
		LastName:    text(fill(existing.LastName.String, in.LastName, overwrite)),
		Title:       text(fill(existing.Title.String, in.Title, overwrite)),
		LinkedinUrl: text(fill(existing.LinkedinUrl.String, in.LinkedInURL, overwrite)),
		Timezone:    text(fill(existing.Timezone.String, in.Timezone, overwrite)),
		Region:      text(fill(existing.Region.String, in.Region, overwrite)),
		CompanyID:   companyID,
		Custom:      customJSON(mergeCustom(existing.Custom, in.Custom, overwrite)),
	})
	if err != nil {
		return Contact{}, err
	}
	return contactFrom(row.ID, row.Email, row.FirstName, row.LastName, row.Title, row.CompanyID, row.LinkedinUrl, row.Timezone, row.Region, row.RoleClass, row.Provenance, row.Readiness, row.CreatedAt), nil
}
