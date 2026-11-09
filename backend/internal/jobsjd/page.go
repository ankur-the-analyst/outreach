package jobsjd

import "github.com/google/uuid"

const (
	defaultPage = 50
	maxPage     = 100
)

type JobPage struct {
	Jobs []Job      `json:"jobs"`
	Next *uuid.UUID `json:"next_cursor,omitempty"`
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

func clipJobs(rows []Job, limit int) JobPage {
	limit = clampLimit(limit)
	if len(rows) <= limit {
		return JobPage{Jobs: rows}
	}
	last := rows[limit-1].ID
	return JobPage{Jobs: rows[:limit], Next: &last}
}
