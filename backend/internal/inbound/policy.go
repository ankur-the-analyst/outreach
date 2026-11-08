package inbound

type Effect struct {
	Suppress   bool
	Hold       bool
	Stop       bool
	ReplyDraft bool
	Enrolment  string
}

func Policy(c Class) Effect {
	switch c {
	case ClassUnsubscribe, ClassComplaint:
		return Effect{Suppress: true, Enrolment: "unsubscribed"}
	case ClassDSN:
		return Effect{Suppress: true, Enrolment: "bounced"}
	case ClassOutOfOffice:
		return Effect{Hold: true, Enrolment: "held"}
	case ClassNotNow, ClassRejection:
		return Effect{Stop: true, Enrolment: "stopped"}
	case ClassInterested, ClassInterview, ClassQuestion, ClassReferral:
		return Effect{ReplyDraft: true, Enrolment: "replied"}
	case ClassOther:
		return Effect{Enrolment: "replied"}
	default:
		return Effect{}
	}
}

func Decide(c Class, confidence float64) Effect {
	e := Policy(c)
	if c == ClassOther && confidence >= 0.6 {
		e.ReplyDraft = true
	}
	return e
}
