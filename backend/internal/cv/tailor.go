package cv

import (
	"errors"
	"strings"

	"outreach/internal/evidence"
)

var ErrInvented = errors.New("this CV adds a role that is not on the profile")

func CheckSubset(base, got Spec) error {
	if strings.TrimSpace(got.Name) != strings.TrimSpace(base.Name) {
		return ErrInvented
	}
	gh := strings.TrimSpace(got.Headline)
	if gh != "" && gh != strings.TrimSpace(base.Headline) {
		return ErrInvented
	}
	allow := bag(base)
	for _, s := range got.Sections {
		if t := strings.TrimSpace(s.Title); t != "" {
			if _, ok := allow[t]; !ok {
				return ErrInvented
			}
		}
		for _, ln := range s.Lines {
			if t := strings.TrimSpace(ln); t != "" {
				if _, ok := allow[t]; !ok {
					return ErrInvented
				}
			}
		}
	}
	return nil
}

func CheckClaims(spec Spec, cited []evidence.Item, extra string) error {
	items := cited
	if strings.TrimSpace(extra) != "" {
		items = append(append([]evidence.Item{}, cited...), evidence.Item{Text: extra})
	}
	return evidence.GateNumbers(SpecText(spec), items)
}

func SpecText(spec Spec) string {
	var b strings.Builder
	b.WriteString(spec.Name)
	b.WriteByte('\n')
	b.WriteString(spec.Headline)
	b.WriteByte('\n')
	for _, s := range spec.Sections {
		b.WriteString(s.Title)
		b.WriteByte('\n')
		for _, ln := range s.Lines {
			b.WriteString(ln)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func bag(spec Spec) map[string]struct{} {
	out := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			out[s] = struct{}{}
		}
	}
	add(spec.Name)
	add(spec.Headline)
	for _, s := range spec.Sections {
		add(s.Title)
		for _, ln := range s.Lines {
			add(ln)
		}
	}
	return out
}
