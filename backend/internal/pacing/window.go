package pacing

import (
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Window struct {
	Days  []int32
	Start time.Duration
	End   time.Duration
	Loc   *time.Location
}

func WindowFrom(days []int32, start, end pgtype.Time, loc *time.Location) Window {
	if loc == nil {
		loc = time.UTC
	}
	return Window{
		Days:  days,
		Start: time.Duration(start.Microseconds) * time.Microsecond,
		End:   time.Duration(end.Microseconds) * time.Microsecond,
		Loc:   loc,
	}
}

func LoadZone(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

func InWindow(now time.Time, w Window) bool {
	if w.Loc == nil {
		w.Loc = time.UTC
	}
	local := now.In(w.Loc)
	if !dayAllowed(local.Weekday(), w.Days) {
		return false
	}
	tod := timeOfDay(local)
	if w.Start <= w.End {
		return tod >= w.Start && tod < w.End
	}
	return tod >= w.Start || tod < w.End
}

func DayBounds(now time.Time, loc *time.Location) (start, end time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	y, m, d := local.Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, loc).UTC()
	end = time.Date(y, m, d+1, 0, 0, 0, 0, loc).UTC()
	return start, end
}

func NextOpen(now time.Time, w Window) time.Time {
	if w.Loc == nil {
		w.Loc = time.UTC
	}
	t := now.In(w.Loc).Truncate(time.Minute)
	if InWindow(t, w) {
		return t.UTC()
	}
	for i := 0; i < 8*24*60; i++ {
		t = t.Add(time.Minute)
		if InWindow(t, w) {
			return t.UTC()
		}
	}
	return now.UTC().Add(time.Hour)
}

func dayAllowed(day time.Weekday, days []int32) bool {
	n := int32(day)
	for _, d := range days {
		if d == n {
			return true
		}
	}
	return false
}

func timeOfDay(t time.Time) time.Duration {
	return time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second
}
