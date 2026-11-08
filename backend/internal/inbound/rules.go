package inbound

import (
	"strings"
)

type Class string

const (
	ClassInterested  Class = "interested"
	ClassInterview   Class = "interview"
	ClassQuestion    Class = "question"
	ClassNotNow      Class = "not_now"
	ClassRejection   Class = "rejection"
	ClassReferral    Class = "referral"
	ClassOutOfOffice Class = "out_of_office"
	ClassUnsubscribe Class = "unsubscribe"
	ClassComplaint   Class = "complaint"
	ClassDSN         Class = "dsn"
	ClassOther       Class = "other"
)

func Rules(from, subject, body string) (Class, bool) {
	blob := strings.ToLower(from + "\n" + subject + "\n" + body)
	switch {
	case containsAny(blob, "mailer-daemon", "postmaster@", "delivery status notification", "undeliverable", "delivery failure"):
		return ClassDSN, true
	case containsAny(blob, "unsubscribe", "opt-out", "opt out", "remove me from"):
		return ClassUnsubscribe, true
	case containsAny(blob, "mark as spam", "this is spam", "report spam"):
		return ClassComplaint, true
	case containsAny(blob, "out of office", "out-of-office", "automatic reply", "autoreply", "auto-reply", "vacation"):
		return ClassOutOfOffice, true
	case containsAny(blob, "interview"):
		return ClassInterview, true
	case containsAny(blob, "i'm interested", "i am interested", "we're interested", "we are interested"):
		return ClassInterested, true
	default:
		return "", false
	}
}

func containsAny(blob string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(blob, n) {
			return true
		}
	}
	return false
}
