package compose

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/evidence"
)

func TestCheckGates(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	cited := []evidence.Item{{ID: id, Text: "Closed 12 deals last year."}}
	facts := []Fact{{Claim: "12 deals", EvidenceID: id}}
	if fails := Check("You closed 12 deals last year.", facts, cited); len(fails) != 0 {
		t.Fatalf("%+v", fails)
	}
	if fails := Check("Hi {{name}}, guaranteed https://a.example https://b.example", facts, cited); len(fails) < 3 {
		t.Fatalf("want template+banned+links, got %+v", fails)
	}
	long := strings.Repeat("word ", MaxWords+1)
	if fails := Check(long, nil, nil); !hasCode(fails, "length") {
		t.Fatalf("%+v", fails)
	}
	other := uuid.MustParse("01700000-0000-7000-8000-000000000002")
	if fails := Check("You closed 99 deals.", []Fact{{EvidenceID: other}}, cited); !hasCode(fails, "uncited") || !hasCode(fails, "claim") {
		t.Fatalf("%+v", fails)
	}
	if !Leftover("ok", "<p>Hi {{contact.first_name}}</p>") || Leftover("Hi Ada", "<p>Hi Ada</p>") {
		t.Fatal("leftover html")
	}
}

func hasCode(fails []Failure, code string) bool {
	for _, f := range fails {
		if f.Code == code {
			return true
		}
	}
	return false
}
