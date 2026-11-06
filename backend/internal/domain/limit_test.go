package domain

import "testing"

func TestDefaultDailyLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		age  int
		want int
	}{
		{0, 40},
		{29, 40},
		{30, 100},
		{400, 100},
	}
	for _, tt := range tests {
		if got := DefaultDailyLimit(tt.age); got != tt.want {
			t.Fatalf("age %d: got %d want %d", tt.age, got, tt.want)
		}
	}
}
