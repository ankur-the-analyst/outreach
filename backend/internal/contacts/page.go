package contacts

import "github.com/google/uuid"

const (
	defaultPage = 50
	maxPage     = 100
)

type Page struct {
	Contacts []Contact  `json:"contacts"`
	Next     *uuid.UUID `json:"next_cursor,omitempty"`
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

func clipPage(rows []Contact, limit int) Page {
	limit = clampLimit(limit)
	if len(rows) <= limit {
		return Page{Contacts: rows}
	}
	last := rows[limit-1].ID
	return Page{Contacts: rows[:limit], Next: &last}
}
