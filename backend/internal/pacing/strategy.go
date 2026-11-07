package pacing

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MailboxView struct {
	ID          uuid.UUID
	DomainID    uuid.UUID
	Status      string
	DailyLimit  int
	Used        int
	DomainLimit int
	DomainUsed  int
}

type LedgerView struct {
	LastFirstTouch map[uuid.UUID]time.Time
}

type CampaignSender struct {
	MailboxID uuid.UUID
	Position  int
	Weight    int
}

type Strategy interface {
	Order(eligible []MailboxView, ledger LedgerView, senders []CampaignSender) []uuid.UUID
}

var (
	strategiesMu sync.RWMutex
	strategies   = map[string]Strategy{}
)

func Register(name string, s Strategy) {
	strategiesMu.Lock()
	defer strategiesMu.Unlock()
	strategies[name] = s
}

func Lookup(name string) (Strategy, error) {
	if name == "" {
		name = "round_robin"
	}
	strategiesMu.RLock()
	s, ok := strategies[name]
	strategiesMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("pacing: unknown strategy %q", name)
	}
	return s, nil
}

func Eligible(v MailboxView) bool {
	switch v.Status {
	case StatusHealthy, StatusWarming:
	default:
		return false
	}
	if v.DailyLimit > 0 && v.Used >= v.DailyLimit {
		return false
	}
	if v.DomainLimit > 0 && v.DomainUsed >= v.DomainLimit {
		return false
	}
	return true
}

type RoundRobin struct{}

func (RoundRobin) Order(eligible []MailboxView, ledger LedgerView, _ []CampaignSender) []uuid.UUID {
	cp := append([]MailboxView(nil), eligible...)
	sort.SliceStable(cp, func(i, j int) bool {
		ti := ledger.LastFirstTouch[cp[i].ID]
		tj := ledger.LastFirstTouch[cp[j].ID]
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return cp[i].ID.String() < cp[j].ID.String()
	})
	out := make([]uuid.UUID, len(cp))
	for i, v := range cp {
		out[i] = v.ID
	}
	return out
}

func init() {
	Register("round_robin", RoundRobin{})
}
