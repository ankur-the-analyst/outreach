package health

import "time"

type Sample struct {
	ID        int64     `json:"id"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Window    string    `json:"window"`
	CreatedAt time.Time `json:"created_at"`
}

func Label(metric string) string {
	switch metric {
	case "unknown":
		return "Sends we're still checking"
	case "paused":
		return "Paused mailboxes"
	case "sync_lag_s":
		return "Oldest inbox sync (seconds)"
	default:
		return metric
	}
}

func Metrics(unknown, paused int64, lagSec *float64) []Sample {
	out := []Sample{
		{Metric: "unknown", Value: float64(unknown), Window: "live"},
		{Metric: "paused", Value: float64(paused), Window: "live"},
	}
	if lagSec != nil {
		out = append(out, Sample{Metric: "sync_lag_s", Value: *lagSec, Window: "live"})
	}
	return out
}
