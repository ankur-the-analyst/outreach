package pacing

import "fmt"

type State string

const (
	StateHeld      State = "held"
	StateReserved  State = "reserved"
	StateSending   State = "sending"
	StateSent      State = "sent"
	StateUnknown   State = "unknown"
	StateReleased  State = "released"
	StateCancelled State = "cancelled"
	StateFailed    State = "failed"
)

func (s State) CountsTowardBudget() bool {
	switch s {
	case StateHeld, StateReserved, StateSending, StateSent, StateUnknown:
		return true
	default:
		return false
	}
}

func (s State) CanAdvance(to State) bool {
	switch s {
	case StateHeld:
		return to == StateReserved || to == StateCancelled
	case StateReserved:
		return to == StateSending || to == StateReleased
	case StateSending:
		return to == StateSent || to == StateUnknown || to == StateFailed
	case StateUnknown:
		return to == StateSent || to == StateFailed
	default:
		return false
	}
}

func ParseState(v string) (State, error) {
	s := State(v)
	switch s {
	case StateHeld, StateReserved, StateSending, StateSent, StateUnknown, StateReleased, StateCancelled, StateFailed:
		return s, nil
	default:
		return "", fmt.Errorf("pacing: unknown ledger state %q", v)
	}
}
