package contacts

import (
	"encoding/json"
	"testing"
)

func TestMergeFillsBlanksOnly(t *testing.T) {
	t.Parallel()
	got := mergeCustom(json.RawMessage(`{"role":"AE","note":"keep"}`), map[string]string{
		"role": "SDR",
		"city": "Austin",
	}, false)
	if got["role"] != "AE" || got["city"] != "Austin" {
		t.Fatalf("%v", got)
	}
	replaced := mergeCustom(json.RawMessage(`{"role":"AE"}`), map[string]string{"role": "SDR"}, true)
	if replaced["role"] != "SDR" {
		t.Fatalf("%v", replaced)
	}
	if fill("Ada", "Ann", false) != "Ada" || fill("", "Ann", false) != "Ann" || fill("Ada", "Ann", true) != "Ann" {
		t.Fatal("fill")
	}
}

func TestParseDedupRejectsUnknownAction(t *testing.T) {
	t.Parallel()
	d := parseDedup(json.RawMessage(`{"action":"allow","overwrite":true}`))
	if d.Action != "prevent" || d.Overwrite {
		t.Fatalf("%+v", d)
	}
}
