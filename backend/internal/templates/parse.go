package templates

import (
	"bytes"
	"encoding/json"
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
	ErrDesign    = errors.New("invalid canvas")
)

var known = map[string]struct{}{
	"contact.first_name":   {},
	"contact.last_name":    {},
	"contact.email":        {},
	"contact.title":        {},
	"contact.linkedin_url": {},
	"contact.timezone":     {},
	"contact.region":       {},
	"user.name":            {},
	"user.email":           {},
	"company.name":         {},
	"job.title":            {},
	"profile.name":         {},
}

var (
	slotRE = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
	pathRE = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)
)

func allowedPath(path string, extra map[string]struct{}) bool {
	if Known(path) {
		return true
	}
	_, ok := extra[path]
	return ok
}

func Known(path string) bool {
	if _, ok := known[path]; ok {
		return true
	}
	return false
}

func ValidateAllowed(s string, extra map[string]struct{}) error {
	return validate(s, extra)
}

func ValidateJSON(raw json.RawMessage) error {
	return validateJSON(raw, nil)
}

func validateJSON(raw json.RawMessage, extra map[string]struct{}) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("{}")) {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ErrDesign
	}
	return walkVars(v, extra)
}

func walkVars(v any, extra map[string]struct{}) error {
	switch t := v.(type) {
	case string:
		return validate(t, extra)
	case []any:
		for _, x := range t {
			if err := walkVars(x, extra); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, x := range t {
			if err := walkVars(x, extra); err != nil {
				return err
			}
		}
	}
	return nil
}

func Validate(s string) error {
	return validate(s, nil)
}

func validate(s string, extra map[string]struct{}) error {
	if strings.Count(s, "{{") != strings.Count(s, "}}") {
		return ErrUnclosed
	}
	for _, m := range slotRE.FindAllStringSubmatch(s, -1) {
		path, _, ok := parseSlot(m[1])
		if !ok || !allowedPath(path, extra) {
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
