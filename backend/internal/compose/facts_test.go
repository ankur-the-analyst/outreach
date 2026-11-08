package compose

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/evidence"
)

type stubRetrieve struct {
	items []evidence.Item
	err   error
	got   *uuid.UUID
}

func (s *stubRetrieve) Retrieve(_ context.Context, _ uuid.UUID, profileID *uuid.UUID, _ string) ([]evidence.Item, error) {
	s.got = profileID
	return s.items, s.err
}

func TestFillFactsFromRetrieve(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	profile := uuid.MustParse("01700000-0000-7000-8000-00000000000a")
	r := &stubRetrieve{items: []evidence.Item{{ID: id, Text: "closed 12 deals"}}}
	in, err := fillFacts(context.Background(), r, uuid.MustParse("01700000-0000-7000-8000-000000000002"), ComposeInput{
		Subject: "hi", ProfileID: &profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.got == nil || *r.got != profile {
		t.Fatalf("profile = %v", r.got)
	}
	if len(in.Facts) != 1 || in.Facts[0].EvidenceID != id || in.Facts[0].Claim != "closed 12 deals" {
		t.Fatalf("facts = %+v", in.Facts)
	}
}

func TestFillFactsSkipsWhenProvided(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	r := &stubRetrieve{items: []evidence.Item{{ID: id, Text: "other"}}}
	in, err := fillFacts(context.Background(), r, uuid.Nil, ComposeInput{
		Facts: []Fact{{Claim: "typed", EvidenceID: id}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.got != nil {
		t.Fatal("retrieve should not run")
	}
	if in.Facts[0].Claim != "typed" {
		t.Fatalf("facts = %+v", in.Facts)
	}
}

func TestFillFactsNoClaims(t *testing.T) {
	t.Parallel()
	_, err := fillFacts(context.Background(), &stubRetrieve{}, uuid.Nil, ComposeInput{})
	if !errors.Is(err, ErrNoClaims) {
		t.Fatalf("err = %v", err)
	}
}
