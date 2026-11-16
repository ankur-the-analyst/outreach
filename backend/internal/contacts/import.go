package contacts

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"outreach/internal/fields"
)

type ImportResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

func applyImport(rows []PreviewRow, create func(PreviewRow) error) (created, updated, skipped int, err error) {
	seen := map[string]struct{}{}
	for _, row := range rows {
		email := strings.ToLower(strings.TrimSpace(row.Email))
		if email == "" || !strings.Contains(email, "@") {
			skipped++
			continue
		}
		if _, ok := seen[email]; ok {
			skipped++
			continue
		}
		if err := create(row); err != nil {
			var up Updated
			if errors.As(err, &up) {
				seen[email] = struct{}{}
				updated++
				continue
			}
			if errors.Is(err, ErrDuplicate) {
				seen[email] = struct{}{}
				skipped++
				continue
			}
			return created, updated, skipped, err
		}
		seen[email] = struct{}{}
		created++
	}
	return created, updated, skipped, nil
}

func (p *Postgres) Import(ctx context.Context, workspaceID, userID uuid.UUID, source, csv string, columns ColumnMap) (ImportResult, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return ImportResult{}, ErrProvenance
	}
	var extra []fields.Hint
	if p.catalog != nil {
		h, err := p.catalog.Hints(ctx, workspaceID, "contact")
		if err != nil {
			return ImportResult{}, err
		}
		extra = h
	}
	parsed, err := ParseCSV(csv, columns, extra)
	if err != nil {
		return ImportResult{}, err
	}
	if strings.TrimSpace(parsed.Columns["email"]) == "" {
		return ImportResult{}, ErrMapEmail
	}
	created, updated, skipped, err := applyImport(parsed.Rows, func(row PreviewRow) error {
		_, err := p.Create(ctx, workspaceID, CreateInput{
			Email:       row.Email,
			FirstName:   row.FirstName,
			LastName:    row.LastName,
			Title:       row.Title,
			Company:     row.Company,
			LinkedInURL: row.LinkedInURL,
			Timezone:    row.Timezone,
			Region:      row.Region,
			Source:      source,
			AddedBy:     userID,
			Custom:      row.Custom,
		})
		return err
	})
	return ImportResult{Created: created, Updated: updated, Skipped: skipped}, err
}
