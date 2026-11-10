package templates

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrUnknown   = errors.New("that placeholder is not a field we fill")
	ErrUnclosed  = errors.New("unclosed placeholder")
	ErrName      = errors.New("name required")
	ErrSubject   = errors.New("subject required")
	ErrBody      = errors.New("body required")
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

var known = map[string]struct{}{
	"contact.first_name": {},
	"contact.last_name":  {},
	"contact.email":      {},
	"contact.title":      {},
	"company.name":       {},
	"job.title":          {},
	"profile.name":       {},
}

var (
	slotRE = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
	pathRE = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)
)

func Known(path string) bool {
	_, ok := known[path]
	return ok
}

func Validate(s string) error {
	if strings.Count(s, "{{") != strings.Count(s, "}}") {
		return ErrUnclosed
	}
	for _, m := range slotRE.FindAllStringSubmatch(s, -1) {
		path, _, ok := parseSlot(m[1])
		if !ok || !Known(path) {
			return fmt.Errorf("%w", ErrUnknown)
		}
	}
	if strings.Contains(slotRE.ReplaceAllString(s, ""), "{{") {
		return ErrUnclosed
	}
	return nil
}

func Render(s string, values map[string]string) string {
	return slotRE.ReplaceAllStringFunc(s, func(raw string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(raw, "{{"), "}}")
		path, fallback, ok := parseSlot(inner)
		if !ok {
			return raw
		}
		v := strings.TrimSpace(values[path])
		if v == "" {
			return fallback
		}
		return v
	})
}

func parseSlot(inner string) (path, fallback string, ok bool) {
	inner = strings.TrimSpace(inner)
	path, rest, cut := strings.Cut(inner, "|")
	path = strings.TrimSpace(path)
	if !pathRE.MatchString(path) {
		return "", "", false
	}
	if cut {
		fallback = strings.TrimSpace(rest)
	}
	return path, fallback, true
}
