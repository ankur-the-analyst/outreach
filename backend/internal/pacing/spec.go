package pacing

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrCustomStart = errors.New("start must be at least 1")

const (
	KindStandard     = "standard"
	KindConservative = "conservative"
	KindFast         = "fast"
	KindOff          = "off"
	KindCustom       = "custom"

	StatusHealthy        = "healthy"
	StatusWarming        = "warming"
	StatusThrottled      = "throttled"
	StatusPaused         = "paused"
	StatusNeedsReconnect = "needs_reconnect"
)

type Spec struct {
	Start        int     `json:"start"`
	Ramp         []int   `json:"ramp"`
	MaxGrowth    float64 `json:"max_growth"`
	IdleCounts   bool    `json:"idle_counts"`
	BounceGate   float64 `json:"bounce_gate"`
	DeferralGate float64 `json:"deferral_gate"`
	MinSample    int     `json:"min_sample"`
	Uncapped     bool    `json:"uncapped"`
}

type Health struct {
	Status       string
	BounceRate   float64
	DeferralRate float64
	AuthFailures int
	SampleSize   int
}

type Preset struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Spec Spec   `json:"spec"`
}

func ParseSpec(raw []byte) (Spec, error) {
	var s Spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return Spec{}, fmt.Errorf("pacing spec: %w", err)
	}
	return s, nil
}

func CustomSpec(start int) (Spec, error) {
	if start < 1 {
		return Spec{}, ErrCustomStart
	}
	return Spec{
		Start:        start,
		Ramp:         []int{start},
		MaxGrowth:    0,
		IdleCounts:   false,
		BounceGate:   0.03,
		DeferralGate: 0.15,
		MinSample:    20,
	}, nil
}
