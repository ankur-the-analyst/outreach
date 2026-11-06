package pacing

import (
	"encoding/json"
	"math"
	"testing"
)

func standardSpec(t *testing.T) Spec {
	t.Helper()
	presets, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range presets {
		if p.Kind == KindStandard {
			return p.Spec
		}
	}
	t.Fatal("missing standard")
	return Spec{}
}

func TestRampDay1Cap2(t *testing.T) {
	t.Parallel()
	spec := standardSpec(t)
	h := Health{Status: StatusWarming}
	if got := Ramp(spec, 1, h); got != 2 {
		t.Fatalf("day 1: got %d want 2", got)
	}
	if got := Ramp(spec, 0, h); got != 2 {
		t.Fatalf("day 0: got %d want 2", got)
	}
}

func TestRampStandardCurve(t *testing.T) {
	t.Parallel()
	spec := standardSpec(t)
	h := Health{Status: StatusHealthy}
	tests := []struct {
		day  int
		want int
	}{
		{1, 2},
		{2, 3},
		{3, 5},
		{4, 6}, // ceil(5 * 1.2)
	}
	for _, tt := range tests {
		if got := Ramp(spec, tt.day, h); got != tt.want {
			t.Fatalf("day %d: got %d want %d", tt.day, got, tt.want)
		}
	}
}

func TestRampUnmarshalSpec(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"start":2,"ramp":[2,3,5],"max_growth":0.2,"bounce_gate":0.03,"min_sample":20}`)
	spec, err := ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}
	round, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseSpec(round)
	if err != nil {
		t.Fatal(err)
	}
	if again.Start != 2 || len(again.Ramp) != 3 || again.Ramp[2] != 5 {
		t.Fatalf("%+v", again)
	}
	if Ramp(again, 1, Health{Status: StatusHealthy}) != 2 {
		t.Fatal("unmarshaled spec day-1")
	}
}

func TestRampHealthGates(t *testing.T) {
	t.Parallel()
	spec := standardSpec(t)
	tests := []struct {
		name string
		h    Health
		want int
	}{
		{"paused", Health{Status: StatusPaused}, 0},
		{"reconnect", Health{Status: StatusNeedsReconnect}, 0},
		{"auth", Health{Status: StatusHealthy, AuthFailures: 1}, 0},
		{"bounce", Health{Status: StatusHealthy, BounceRate: 0.04, SampleSize: 20}, 0},
		{"bounce_small_n", Health{Status: StatusHealthy, BounceRate: 0.04, SampleSize: 10}, 2},
		{"ok", Health{Status: StatusHealthy, BounceRate: 0.01, SampleSize: 20}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Ramp(spec, 1, tt.h); got != tt.want {
				t.Fatalf("got %d want %d", got, tt.want)
			}
		})
	}
}

func TestRampOffUncapped(t *testing.T) {
	t.Parallel()
	presets, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	var spec Spec
	for _, p := range presets {
		if p.Kind == KindOff {
			spec = p.Spec
		}
	}
	got := Ramp(spec, 1, Health{Status: StatusHealthy})
	if got != math.MaxInt32 {
		t.Fatalf("got %d", got)
	}
}

func TestIdleDayDoesNotIncrement(t *testing.T) {
	t.Parallel()
	if ShouldAdvance(0, 2) {
		t.Fatal("0 of 2 is idle")
	}
	if !ShouldAdvance(1, 2) {
		t.Fatal("1 of 2 meets half")
	}
	if NextSendingDay(0, 0, 2, false) != 0 {
		t.Fatal("idle first day stays 0")
	}
	if NextSendingDay(1, 0, 3, false) != 1 {
		t.Fatal("idle day 1 stays 1")
	}
	if NextSendingDay(0, 1, 2, false) != 2 {
		t.Fatal("enough volume advances 0 → 2")
	}
	if NextSendingDay(2, 0, 5, true) != 3 {
		t.Fatal("idle_counts advances anyway")
	}
}
