package sync

import (
	"testing"
	"time"
)

func TestBackoffDoublesThenCaps(t *testing.T) {
	t.Parallel()
	if Backoff(1) != time.Second || Backoff(2) != 2*time.Second || Backoff(3) != 4*time.Second {
		t.Fatalf("%v %v %v", Backoff(1), Backoff(2), Backoff(3))
	}
	if Backoff(6) != 32*time.Second || Backoff(9) != 32*time.Second {
		t.Fatalf("%v", Backoff(9))
	}
}
