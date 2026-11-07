package contacts

import "strings"

var genericLocal = map[string]struct{}{
	"careers": {}, "hr": {}, "jobs": {}, "recruiting": {}, "recruitment": {},
	"talent": {}, "people": {}, "info": {}, "hello": {}, "support": {},
	"team": {}, "admin": {}, "office": {}, "contact": {},
}

func classifyEmail(email string) (roleClass string, generic bool) {
	local, _, ok := strings.Cut(strings.ToLower(email), "@")
	if !ok || local == "" {
		return "unknown", false
	}
	local, _, _ = strings.Cut(local, "+")
	if _, ok := genericLocal[local]; ok {
		return "generic_mailbox", true
	}
	return "unknown", false
}

func domainOf(email string) string {
	_, domain, ok := strings.Cut(strings.ToLower(strings.TrimSpace(email)), "@")
	if !ok {
		return ""
	}
	return strings.TrimSpace(domain)
}

func readiness(email string, generic bool) string {
	if email == "" || !strings.Contains(email, "@") {
		return "blocked"
	}
	if generic {
		return "risky"
	}
	return "ready"
}
