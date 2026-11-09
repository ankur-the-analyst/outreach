package pacing

import "testing"

func TestCustomSpecDay1(t *testing.T) {
	t.Parallel()
	if _, err := CustomSpec(0); err != ErrCustomStart {
		t.Fatalf("zero start %v", err)
	}
	spec, err := CustomSpec(4)
	if err != nil {
		t.Fatal(err)
	}
	if got := Ramp(spec, 1, Health{Status: StatusHealthy}); got != 4 {
		t.Fatalf("day 1 %d", got)
	}
}
