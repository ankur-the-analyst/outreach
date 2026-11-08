package campaign

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/evidence"
)

type stubRetrieve struct {
	items []evidence.Item
	got   *uuid.UUID
}

func (s *stubRetrieve) Retrieve(_ context.Context, _ uuid.UUID, profileID *uuid.UUID, _ string) ([]evidence.Item, error) {
	s.got = profileID
	return s.items, nil
}

func TestFactsFromRetrieve(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	profile := uuid.MustParse("01700000-0000-7000-8000-00000000000a")
	r := &stubRetrieve{items: []evidence.Item{{ID: id, Text: "closed 12 deals"}}}
	facts, err := factsFromRetrieve(context.Background(), r, uuid.Nil, &profile, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if r.got == nil || *r.got != profile {
		t.Fatalf("profile = %v", r.got)
	}
	if len(facts) != 1 || facts[0].EvidenceID != id || facts[0].Claim != "closed 12 deals" {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestFactsFromRetrieveNil(t *testing.T) {
	t.Parallel()
	facts, err := factsFromRetrieve(context.Background(), nil, uuid.Nil, nil, "hi")
	if err != nil || len(facts) != 0 {
		t.Fatalf("%+v %v", facts, err)
	}
}
