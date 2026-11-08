package contacts

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type ImportResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

func applyImport(rows []PreviewRow, create func(PreviewRow) error) (created, skipped int, err error) {
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
			if errors.Is(err, ErrDuplicate) {
				seen[email] = struct{}{}
				skipped++
				continue
			}
			return created, skipped, err
		}
		seen[email] = struct{}{}
		created++
	}
	return created, skipped, nil
}

func (p *Postgres) Import(ctx context.Context, workspaceID, userID uuid.UUID, source, csv string) (ImportResult, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return ImportResult{}, ErrProvenance
	}
	rows, err := PreviewCSV(csv)
	if err != nil {
		return ImportResult{}, err
	}
	created, skipped, err := applyImport(rows, func(row PreviewRow) error {
		_, err := p.Create(ctx, workspaceID, CreateInput{
			Email:     row.Email,
			FirstName: row.FirstName,
			LastName:  row.LastName,
			Title:     row.Title,
			Company:   row.Company,
			Source:    source,
			AddedBy:   userID,
		})
		return err
	})
	return ImportResult{Created: created, Skipped: skipped}, err
}
