package cv

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"outreach/internal/ai"
	"outreach/internal/evidence"
)

func TestCheckSubsetDropsAndRejectsInvented(t *testing.T) {
	t.Parallel()
	base := Spec{
		Name: "Ada",
		Sections: []Section{
			{Title: "Outcode", Lines: []string{"Shipped Go APIs", "Closed 99 unicorns"}},
		},
	}
	dropped := Spec{Name: "Ada", Sections: []Section{{Title: "Outcode", Lines: []string{"Shipped Go APIs"}}}}
	if err := CheckSubset(base, dropped); err != nil {
		t.Fatal(err)
	}
	invented := Spec{Name: "Ada", Sections: []Section{{Title: "Google", Lines: []string{"Staff engineer"}}}}
	if !errors.Is(CheckSubset(base, invented), ErrInvented) {
		t.Fatal("want invented")
	}
}

func TestCheckClaimsDroppedUnconfirmedOK(t *testing.T) {
	t.Parallel()
	id := evidence.Item{Text: "Shipped Go APIs"}
	spec := Spec{Name: "Ada", Sections: []Section{{Title: "Outcode", Lines: []string{"Shipped Go APIs"}}}}
	if err := CheckClaims(spec, []evidence.Item{id}, ""); err != nil {
		t.Fatal(err)
	}
	bad := Spec{Name: "Ada", Sections: []Section{{Title: "Outcode", Lines: []string{"Closed 99 unicorns"}}}}
	if err := CheckClaims(bad, []evidence.Item{id}, ""); !errors.Is(err, evidence.ErrClaim) {
		t.Fatalf("err %v", err)
	}
}

func TestProposeInventedEmployerFails(t *testing.T) {
	t.Parallel()
	base := Spec{Name: "Ada", Sections: []Section{{Title: "Outcode", Lines: []string{"Shipped Go APIs"}}}}
	w := &ai.Fake{JSON: json.RawMessage(`{"name":"Ada","sections":[{"title":"Google","lines":["Staff"]}]}`)}
	_, _, err := Propose(context.Background(), w, base, "Need a Go lead", nil)
	if !errors.Is(err, ErrInvented) {
		t.Fatalf("err %v", err)
	}
	if w.JSONN != 2 {
		t.Fatalf("one regen want 2 calls, got %d", w.JSONN)
	}
}

func TestProposeJunkJSONUsesBase(t *testing.T) {
	t.Parallel()
	base := Spec{Name: "Ada", Sections: []Section{{Title: "Outcode", Lines: []string{"Shipped Go APIs"}}}}
	w := &ai.Fake{JSON: json.RawMessage(`{"score":82}`)}
	got, regen, err := Propose(context.Background(), w, base, "JD", nil)
	if err != nil || regen || got.Name != "Ada" {
		t.Fatalf("%+v %v regen=%v", got, err, regen)
	}
}

func TestProposeDropUnconfirmedOK(t *testing.T) {
	t.Parallel()
	base := Spec{
		Name: "Ada",
		Sections: []Section{
			{Title: "Outcode", Lines: []string{"Shipped Go APIs", "Closed 99 unicorns"}},
		},
	}
	w := &ai.Fake{JSON: json.RawMessage(`{"name":"Ada","sections":[{"title":"Outcode","lines":["Shipped Go APIs"]}]}`)}
	got, regen, err := Propose(context.Background(), w, base, "Need Go APIs", []evidence.Item{{Text: "Shipped Go APIs"}})
	if err != nil || regen {
		t.Fatalf("%+v %v regen=%v", got, err, regen)
	}
	if len(got.Sections) != 1 || len(got.Sections[0].Lines) != 1 {
		t.Fatalf("%+v", got)
	}
}
