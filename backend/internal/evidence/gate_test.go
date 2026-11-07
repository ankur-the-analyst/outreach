package evidence

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestGateNumbersNotInEvidenceFails(t *testing.T) {
	t.Parallel()
	cited := []Item{{ID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), Text: "Led a team of 3 engineers at Acme."}}
	if err := GateNumbers("I led 3 engineers.", cited); err != nil {
		t.Fatal(err)
	}
	err := GateNumbers("We shipped in 12 weeks with 3 engineers.", cited)
	if !errors.Is(err, ErrClaim) {
		t.Fatalf("got %v want claim 12", err)
	}
}

func TestGateNumbersEmptyBody(t *testing.T) {
	t.Parallel()
	if err := GateNumbers("hello there", nil); err != nil {
		t.Fatal(err)
	}
}
