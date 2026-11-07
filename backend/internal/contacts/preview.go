package contacts

import (
	"encoding/csv"
	"strings"
)

type PreviewRow struct {
	Email     string   `json:"email"`
	FirstName string   `json:"first_name,omitempty"`
	LastName  string   `json:"last_name,omitempty"`
	Title     string   `json:"title,omitempty"`
	Company   string   `json:"company,omitempty"`
	RoleClass string   `json:"role_class,omitempty"`
	Readiness string   `json:"readiness"`
	Errors    []string `json:"errors"`
	Duplicate bool     `json:"duplicate"`
	Generic   bool     `json:"generic_mailbox"`
}

func PreviewCSV(text string) ([]PreviewRow, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrCSV
	}
	r := csv.NewReader(strings.NewReader(text))
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []PreviewRow{}, nil
	}
	idx := headerIndex(records[0])
	start := 1
	if idx.email < 0 {
		idx = headerIndex([]string{"email", "first_name", "last_name", "title", "company"})
		start = 0
	}
	seen := map[string]int{}
	var out []PreviewRow
	for i := start; i < len(records); i++ {
		row := records[i]
		email := strings.ToLower(strings.TrimSpace(cell(row, idx.email)))
		item := PreviewRow{
			Email:     email,
			FirstName: cell(row, idx.first),
			LastName:  cell(row, idx.last),
			Title:     cell(row, idx.title),
			Company:   cell(row, idx.company),
			Errors:    []string{},
		}
		item.RoleClass, item.Generic = classifyEmail(email)
		item.Readiness = readiness(email, item.Generic)
		if email == "" || !strings.Contains(email, "@") {
			item.Errors = append(item.Errors, "email required")
			item.Readiness = "blocked"
		}
		if n, ok := seen[email]; ok && email != "" {
			item.Duplicate = true
			item.Errors = append(item.Errors, "duplicate in file")
			out[n].Duplicate = true
			out[n].Errors = appendUnique(out[n].Errors, "duplicate in file")
		} else if email != "" {
			seen[email] = len(out)
		}
		out = append(out, item)
	}
	return out, nil
}

type colIndex struct {
	email, first, last, title, company int
}

func headerIndex(h []string) colIndex {
	idx := colIndex{email: -1, first: -1, last: -1, title: -1, company: -1}
	for i, raw := range h {
		switch normalizeHeader(raw) {
		case "email", "e-mail", "mail":
			idx.email = i
		case "first_name", "firstname", "first":
			idx.first = i
		case "last_name", "lastname", "last":
			idx.last = i
		case "title", "role":
			idx.title = i
		case "company", "company_name", "org":
			idx.company = i
		}
	}
	return idx
}

func normalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}
