package mailbox

import (
	"errors"
	"strings"
)

var (
	ErrGmailServiceAccount = errors.New("cannot attach a service account to gmail.com")
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not found")
	ErrNotSupported        = errors.New("auth kind not supported yet")
	ErrEmailTaken          = errors.New("mailbox already connected")
	ErrNoSMTP              = errors.New("smtp host not set")
	ErrReplyMode           = errors.New("reply_mode must be draft or off")
	ErrWarmupName          = errors.New("name required")
	ErrWarmupStart         = errors.New("start must be at least 1")
	ErrCustomExists        = errors.New("a custom ramp already exists")
)

func EmailDomain(email string) string {
	_, domain, ok := strings.Cut(strings.ToLower(strings.TrimSpace(email)), "@")
	if !ok {
		return ""
	}
	return domain
}

func IsGmail(email string) bool {
	d := EmailDomain(email)
	return d == "gmail.com" || d == "googlemail.com"
}

func RejectGmailServiceAccount(email, authKind string) error {
	if authKind != "service_account" {
		return nil
	}
	if IsGmail(email) {
		return ErrGmailServiceAccount
	}
	return nil
}

func CanEdit(role string) bool {
	return role == "owner" || role == "member"
}

func ProviderFor(email string) string {
	if IsGmail(email) {
		return "gmail"
	}
	return "other"
}
