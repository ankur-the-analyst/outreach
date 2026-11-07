package pacing

import "testing"

func TestStateCountsTowardBudget(t *testing.T) {
	t.Parallel()
	count := []State{StateHeld, StateReserved, StateSending, StateSent, StateUnknown}
	free := []State{StateReleased, StateCancelled, StateFailed}
	for _, s := range count {
		if !s.CountsTowardBudget() {
			t.Fatalf("%s should count", s)
		}
	}
	for _, s := range free {
		if s.CountsTowardBudget() {
			t.Fatalf("%s should not count", s)
		}
	}
}

func TestStateMachine(t *testing.T) {
	t.Parallel()
	ok := []struct{ from, to State }{
		{StateHeld, StateReserved},
		{StateHeld, StateCancelled},
		{StateReserved, StateSending},
		{StateReserved, StateReleased},
		{StateSending, StateSent},
		{StateSending, StateUnknown},
		{StateSending, StateFailed},
		{StateUnknown, StateSent},
		{StateUnknown, StateFailed},
	}
	for _, c := range ok {
		if !c.from.CanAdvance(c.to) {
			t.Fatalf("%s → %s should be allowed", c.from, c.to)
		}
	}
	bad := []struct{ from, to State }{
		{StateHeld, StateSent},
		{StateCancelled, StateHeld},
		{StateSent, StateSending},
		{StateFailed, StateReserved},
		{StateReleased, StateReserved},
	}
	for _, c := range bad {
		if c.from.CanAdvance(c.to) {
			t.Fatalf("%s → %s should be rejected", c.from, c.to)
		}
	}
}
