package compose

import "github.com/google/uuid"

const (
	defaultPage = 50
	maxPage     = 100
)

type Page struct {
	Drafts []Draft    `json:"drafts"`
	Next   *uuid.UUID `json:"next_cursor,omitempty"`
}

func clampLimit(n int) int {
	if n <= 0 {
		return defaultPage
	}
	if n > maxPage {
		return maxPage
	}
	return n
}

func clipPage(rows []Draft, limit int) Page {
	limit = clampLimit(limit)
	if len(rows) <= limit {
		return Page{Drafts: rows}
	}
	last := rows[limit-1].ID
	return Page{Drafts: rows[:limit], Next: &last}
}
