package contacts

import (
	"encoding/csv"
	"strings"

	"outreach/internal/fields"
)

type ColumnMap map[string]string

type PreviewRow struct {
	Email       string            `json:"email"`
	FirstName   string            `json:"first_name,omitempty"`
	LastName    string            `json:"last_name,omitempty"`
	Title       string            `json:"title,omitempty"`
	Company     string            `json:"company,omitempty"`
	LinkedInURL string            `json:"linkedin_url,omitempty"`
	Timezone    string            `json:"timezone,omitempty"`
	Region      string            `json:"region,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
	RoleClass   string            `json:"role_class,omitempty"`
	Readiness   string            `json:"readiness"`
	Errors      []string          `json:"errors"`
	Duplicate   bool              `json:"duplicate"`
	Generic     bool              `json:"generic_mailbox"`
}

type Preview struct {
	Headers []string     `json:"headers"`
	Samples [][]string   `json:"samples"`
	Guessed ColumnMap    `json:"guessed"`
	Columns ColumnMap    `json:"columns"`
	Rows    []PreviewRow `json:"rows"`
}

func PreviewCSV(text string) ([]PreviewRow, error) {
	p, err := ParseCSV(text, nil, nil)
	if err != nil {
		return nil, err
	}
	return p.Rows, nil
}

func ParseCSV(text string, columns ColumnMap, extra []fields.Hint) (Preview, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Preview{}, ErrCSV
	}
	r := csv.NewReader(strings.NewReader(text))
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return Preview{}, err
	}
	if len(records) == 0 {
		return Preview{Guessed: ColumnMap{}, Columns: ColumnMap{}, Rows: []PreviewRow{}}, nil
	}
	headers := records[0]
	guessed := guessColumns(headers, extra)
	effective := guessed
	if compact := compactColumns(columns); len(compact) > 0 {
		effective = compact
	}
	idx := indexFromMap(headers, effective)
	out := Preview{
		Headers: headers,
		Guessed: guessed,
		Columns: effective,
		Rows:    []PreviewRow{},
	}
	data := records[1:]
	for i, row := range data {
		if i < 3 {
			out.Samples = append(out.Samples, row)
		}
	}
	if idx.email < 0 {
		return out, nil
	}
	seen := map[string]int{}
	for i := 1; i < len(records); i++ {
		row := records[i]
		email := strings.ToLower(strings.TrimSpace(cell(row, idx.email)))
		item := PreviewRow{
			Email:       email,
			FirstName:   cell(row, idx.first),
			LastName:    cell(row, idx.last),
			Title:       cell(row, idx.title),
			Company:     cell(row, idx.company),
			LinkedInURL: cell(row, idx.linkedin),
			Timezone:    cell(row, idx.timezone),
			Region:      cell(row, idx.region),
			Custom:      map[string]string{},
			Errors:      []string{},
		}
		for key, hi := range idx.custom {
			if v := cell(row, hi); v != "" {
				item.Custom[key] = v
			}
		}
		if len(item.Custom) == 0 {
			item.Custom = nil
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
			out.Rows[n].Duplicate = true
			out.Rows[n].Errors = appendUnique(out.Rows[n].Errors, "duplicate in file")
		} else if email != "" {
			seen[email] = len(out.Rows)
		}
		out.Rows = append(out.Rows, item)
	}
	return out, nil
}

func compactColumns(m ColumnMap) ColumnMap {
	out := ColumnMap{}
	for k, v := range m {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func guessColumns(headers []string, extra []fields.Hint) ColumnMap {
	out := ColumnMap{}
	used := map[int]struct{}{}
	for i, raw := range headers {
		n := fields.NormalizeHeader(raw)
		key := aliasField(n)
		if key == "" {
			for _, h := range extra {
				if h.Key == n || fields.NormalizeHeader(h.Label) == n {
					key = h.Key
					break
				}
			}
		}
		if key == "" {
			continue
		}
		if _, taken := out[key]; taken {
			continue
		}
		if _, ok := used[i]; ok {
			continue
		}
		out[key] = raw
		used[i] = struct{}{}
	}
	return out
}

func aliasField(norm string) string {
	switch norm {
	case "email", "e_mail", "mail":
		return "email"
	case "first_name", "firstname", "first":
		return "first_name"
	case "last_name", "lastname", "last":
		return "last_name"
	case "title", "role":
		return "title"
	case "company", "company_name", "org":
		return "company"
	case "linkedin_url", "linkedin", "linkedin_profile":
		return "linkedin_url"
	case "timezone", "tz":
		return "timezone"
	case "region":
		return "region"
	default:
		return ""
	}
}

type colIndex struct {
	email, first, last, title, company, linkedin, timezone, region int
	custom                                                         map[string]int
}

func indexFromMap(headers []string, columns ColumnMap) colIndex {
	idx := colIndex{
		email: -1, first: -1, last: -1, title: -1, company: -1,
		linkedin: -1, timezone: -1, region: -1, custom: map[string]int{},
	}
	pos := map[string]int{}
	for i, h := range headers {
		pos[h] = i
		pos[fields.NormalizeHeader(h)] = i
	}
	pick := func(field string) int {
		header, ok := columns[field]
		if !ok || strings.TrimSpace(header) == "" {
			return -1
		}
		if i, ok := pos[header]; ok {
			return i
		}
		if i, ok := pos[fields.NormalizeHeader(header)]; ok {
			return i
		}
		return -1
	}
	idx.email = pick("email")
	idx.first = pick("first_name")
	idx.last = pick("last_name")
	idx.title = pick("title")
	idx.company = pick("company")
	idx.linkedin = pick("linkedin_url")
	idx.timezone = pick("timezone")
	idx.region = pick("region")
	for field, header := range columns {
		switch field {
		case "email", "first_name", "last_name", "title", "company", "linkedin_url", "timezone", "region":
			continue
		}
		if i, ok := pos[header]; ok {
			idx.custom[field] = i
		} else if i, ok := pos[fields.NormalizeHeader(header)]; ok {
			idx.custom[field] = i
		}
	}
	return idx
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
