package compose

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/ai"
	"outreach/internal/evidence"
)

func TestGenerateRegeneratesThenShowsFailures(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	cited := []evidence.Item{{ID: id, Text: "Closed 12 deals last year."}}
	facts := []Fact{{Claim: "12 deals", EvidenceID: id}}
	w := &ai.Fake{Texts: []string{
		"Hi {{name}}, you closed 12 deals last year.",
		"You closed 12 deals last year. Open to a quick chat?",
	}}
	body, regen, fails, err := Generate(context.Background(), w, cited, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !regen || len(fails) != 0 || body == "" || w.CompleteN != 2 {
		t.Fatalf("body %q regen %v fails %+v n=%d", body, regen, fails, w.CompleteN)
	}
}

func TestGenerateShowsFailuresAfterOneRegen(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	cited := []evidence.Item{{ID: id, Text: "Closed 12 deals last year."}}
	facts := []Fact{{Claim: "12 deals", EvidenceID: id}}
	w := &ai.Fake{Texts: []string{
		"Hi {{name}}, guaranteed income.",
		"Still {{broken}} and guaranteed.",
	}}
	_, regen, fails, err := Generate(context.Background(), w, cited, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !regen || !hasCode(fails, "template") || !hasCode(fails, "banned") {
		t.Fatalf("regen %v fails %+v", regen, fails)
	}
}
