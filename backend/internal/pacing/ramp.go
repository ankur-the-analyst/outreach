package pacing

import "math"

func Ramp(spec Spec, sendingDay int, h Health) int {
	if h.Status == StatusPaused || h.Status == StatusNeedsReconnect {
		return 0
	}
	if h.AuthFailures > 0 {
		return 0
	}
	if gated(h.BounceRate, spec.BounceGate, h.SampleSize, spec.MinSample) {
		return 0
	}
	if gated(h.DeferralRate, spec.DeferralGate, h.SampleSize, spec.MinSample) {
		return 0
	}
	if spec.Uncapped {
		return math.MaxInt32
	}
	return capForDay(spec, sendingDay)
}

func gated(rate, gate float64, sample, minSample int) bool {
	if minSample <= 0 || sample < minSample {
		return false
	}
	return gate > 0 && rate > gate
}

func capForDay(spec Spec, sendingDay int) int {
	day := sendingDay
	if day < 1 {
		day = 1
	}
	if len(spec.Ramp) > 0 {
		if day <= len(spec.Ramp) {
			return spec.Ramp[day-1]
		}
		if spec.MaxGrowth <= 0 {
			return spec.Ramp[len(spec.Ramp)-1]
		}
		prev := spec.Ramp[len(spec.Ramp)-1]
		for d := len(spec.Ramp) + 1; d <= day; d++ {
			prev = grow(prev, spec.MaxGrowth)
		}
		return prev
	}
	if spec.Start > 0 {
		prev := spec.Start
		for d := 2; d <= day; d++ {
			prev = grow(prev, spec.MaxGrowth)
		}
		return prev
	}
	return 0
}

func grow(prev int, maxGrowth float64) int {
	next := int(math.Ceil(float64(prev) * (1 + maxGrowth)))
	if next <= prev {
		return prev + 1
	}
	return next
}

func ShouldAdvance(used, cap int) bool {
	if cap <= 0 {
		return false
	}
	return used*2 >= cap
}

func NextSendingDay(day, used, cap int, idleCounts bool) int {
	cur := day
	if cur < 1 {
		cur = 1
	}
	if idleCounts || ShouldAdvance(used, cap) {
		return cur + 1
	}
	return day
}
