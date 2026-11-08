package health

import "testing"

func TestMetricsAlwaysIncludeUnknownAndPaused(t *testing.T) {
	t.Parallel()
	lag := 12.5
	got := Metrics(3, 1, &lag)
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Metric != "unknown" || got[0].Value != 3 {
		t.Fatalf("unknown %+v", got[0])
	}
	if got[1].Metric != "paused" || got[1].Value != 1 {
		t.Fatalf("paused %+v", got[1])
	}
	if got[2].Metric != "sync_lag_s" || got[2].Value != 12.5 {
		t.Fatalf("lag %+v", got[2])
	}
	if Label("unknown") != "Sends we're still checking" {
		t.Fatal("label")
	}
	if n := Metrics(0, 0, nil); len(n) != 2 {
		t.Fatalf("no lag %d", len(n))
	}
}
