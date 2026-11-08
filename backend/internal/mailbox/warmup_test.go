package mailbox

import (
	"encoding/json"
	"testing"

	"outreach/internal/db"
)

func TestWarmupFromFirstDay(t *testing.T) {
	t.Parallel()
	got := warmupFrom(db.WarmupProfile{
		Name: "Standard", Kind: "standard",
		Spec: json.RawMessage(`{"start":2,"ramp":[2,3,5]}`),
	})
	if got.FirstDay == nil || *got.FirstDay != 2 {
		t.Fatalf("%+v", got)
	}
	off := warmupFrom(db.WarmupProfile{
		Kind: "off", Spec: json.RawMessage(`{"uncapped":true}`),
	})
	if off.FirstDay != nil {
		t.Fatalf("%+v", off)
	}
}
