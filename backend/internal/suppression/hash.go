package suppression

import (
	"fmt"
	"strings"
)

const (
	KindEmail  = "email"
	KindDomain = "domain"
)

type Hasher interface {
	HMAC(msg []byte) []byte
}

func Normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func DomainOf(email string) string {
	email = Normalize(email)
	_, domain, ok := strings.Cut(email, "@")
	if !ok {
		return ""
	}
	return domain
}

func Hash(h Hasher, kind, value string) ([]byte, string, error) {
	if h == nil {
		return nil, "", fmt.Errorf("suppression: hasher required")
	}
	norm := Normalize(value)
	if norm == "" {
		return nil, "", fmt.Errorf("suppression: value required")
	}
	switch kind {
	case KindEmail, KindDomain:
	default:
		return nil, "", fmt.Errorf("suppression: kind %q", kind)
	}
	return h.HMAC([]byte(kind + ":" + norm)), norm, nil
}
