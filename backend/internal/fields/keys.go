package fields

import (
	"errors"
	"regexp"
	"strings"
)

const MaxPerObject = 50

var (
	ErrKey       = errors.New("fields: invalid key")
	ErrBuiltin   = errors.New("fields: cannot redefine a built-in field")
	ErrObject    = errors.New("fields: unknown object")
	ErrKind      = errors.New("fields: unknown type")
	ErrLimit     = errors.New("fields: too many fields")
	ErrDuplicate = errors.New("fields: that key already exists")
	ErrLabel     = errors.New("fields: label required")
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

var keyRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

var objects = map[string]struct{}{
	"contact": {}, "company": {}, "member": {}, "workspace": {},
}

var kinds = map[string]struct{}{
	"text": {}, "number": {}, "url": {}, "date": {}, "boolean": {}, "select": {},
}

var builtins = map[string]map[string]struct{}{
	"contact": {
		"email": {}, "first_name": {}, "last_name": {}, "title": {},
		"linkedin_url": {}, "timezone": {}, "region": {}, "role_class": {}, "tags": {},
		"company": {},
	},
	"company": {"domain": {}, "name": {}},
	"member":  {"email": {}, "name": {}, "role": {}},
	"workspace": {
		"name": {}, "legal_name": {}, "reply_address": {}, "postal_address": {},
	},
}

func ValidObject(object string) bool {
	_, ok := objects[object]
	return ok
}

func ValidKind(kind string) bool {
	_, ok := kinds[kind]
	return ok
}

func ValidKey(key string) bool {
	return keyRE.MatchString(key)
}

func IsBuiltin(object, key string) bool {
	m, ok := builtins[object]
	if !ok {
		return false
	}
	_, ok = m[key]
	return ok
}

func ContactMapTargets() []string {
	return []string{
		"email", "first_name", "last_name", "title", "company",
		"linkedin_url", "timezone", "region",
	}
}

func NormalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
