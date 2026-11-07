package campaign

import "testing"

func TestLoadPacksMarksCommercial(t *testing.T) {
	t.Parallel()
	packs, err := LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	job := packs["job"]
	if job.Commercial || len(job.Steps) != 3 {
		t.Fatalf("%+v", job)
	}
	cold := packs["cold"]
	if !cold.CommercialStart() {
		t.Fatalf("%+v", cold)
	}
}
