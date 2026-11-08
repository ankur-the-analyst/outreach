package inbound

type Effect struct {
	Suppress    bool
	Hold        bool
	Stop        bool
	ReplyDraft  bool
	Enrolment   string
	Application string
}

func Policy(c Class) Effect {
	switch c {
	case ClassUnsubscribe, ClassComplaint:
		return Effect{Suppress: true, Enrolment: "unsubscribed", Application: "closed"}
	case ClassDSN:
		return Effect{Suppress: true, Enrolment: "bounced"}
	case ClassOutOfOffice:
		return Effect{Hold: true, Enrolment: "held"}
	case ClassNotNow, ClassRejection:
		return Effect{Stop: true, Enrolment: "stopped", Application: "rejected"}
	case ClassInterview:
		return Effect{ReplyDraft: true, Enrolment: "replied", Application: "interview"}
	case ClassInterested, ClassQuestion, ClassReferral:
		return Effect{ReplyDraft: true, Enrolment: "replied", Application: "replied"}
	case ClassOther:
		return Effect{Enrolment: "replied", Application: "replied"}
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
