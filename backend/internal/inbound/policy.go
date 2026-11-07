package inbound

type Effect struct {
	Suppress   bool
	Hold       bool
	Stop       bool
	ReplyDraft bool
}

func Policy(c Class) Effect {
	switch c {
	case ClassUnsubscribe, ClassComplaint, ClassDSN:
		return Effect{Suppress: true}
	case ClassOutOfOffice:
		return Effect{Hold: true}
	case ClassNotNow, ClassRejection:
		return Effect{Stop: true}
	case ClassInterested, ClassInterview, ClassQuestion, ClassReferral:
		return Effect{ReplyDraft: true}
	case ClassOther:
		return Effect{}
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
