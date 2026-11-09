package campaign

import "github.com/google/uuid"

const (
	defaultPage = 50
	maxPage     = 100
)

type Page struct {
	Campaigns []Campaign `json:"campaigns"`
	Next      *uuid.UUID `json:"next_cursor,omitempty"`
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

func clipPage(rows []Campaign, limit int) Page {
	limit = clampLimit(limit)
	if len(rows) <= limit {
		return Page{Campaigns: rows}
	}
	last := rows[limit-1].ID
	return Page{Campaigns: rows[:limit], Next: &last}
}
