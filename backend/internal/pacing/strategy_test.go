package pacing

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoundRobinOldestFirstTouchWins(t *testing.T) {
	t.Parallel()
	a := uuid.MustParse("01700000-0000-7000-8000-00000000000a")
	b := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	now := time.Now()
	order := RoundRobin{}.Order(
		[]MailboxView{{ID: a}, {ID: b}},
		LedgerView{LastFirstTouch: map[uuid.UUID]time.Time{a: now, b: now.Add(-time.Hour)}},
		nil,
	)
	if len(order) != 2 || order[0] != b {
		t.Fatalf("%v", order)
	}
}

func TestEligibleSkipsPausedAndOverCap(t *testing.T) {
	t.Parallel()
	ok := MailboxView{Status: StatusWarming, DailyLimit: 2, Used: 1, DomainLimit: 10, DomainUsed: 1}
	if !Eligible(ok) {
		t.Fatal("want eligible")
	}
	paused := ok
	paused.Status = StatusPaused
	if Eligible(paused) {
		t.Fatal("paused")
	}
	full := ok
	full.Used = 2
	if Eligible(full) {
		t.Fatal("full mailbox")
	}
}
