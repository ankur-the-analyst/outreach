package pacing

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestInWindowWeekdaysAndHours(t *testing.T) {
	t.Parallel()
	weekdays := []int32{1, 2, 3, 4, 5}
	nine := pgtype.Time{Microseconds: 9 * 3600 * 1_000_000, Valid: true}
	five := pgtype.Time{Microseconds: 17 * 3600 * 1_000_000, Valid: true}
	w := WindowFrom(weekdays, nine, five, time.UTC)

	mon10 := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	if !InWindow(mon10, w) {
		t.Fatal("monday 10:00")
	}
	if InWindow(time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC), w) {
		t.Fatal("sunday")
	}
	if InWindow(time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC), w) {
		t.Fatal("before open")
	}
	if InWindow(time.Date(2026, 11, 9, 17, 0, 0, 0, time.UTC), w) {
		t.Fatal("at close")
	}

	pt, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	wPT := WindowFrom(weekdays, nine, five, pt)
	utc18 := time.Date(2026, 11, 9, 18, 0, 0, 0, time.UTC)
	if InWindow(utc18, w) {
		t.Fatal("18:00 UTC mailbox")
	}
	if !InWindow(utc18, wPT) {
		t.Fatal("10:00 PT recipient")
	}

	next := NextOpen(time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC), w)
	want := time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next %s want %s", next, want)
	}

	from, to := DayBounds(time.Date(2026, 11, 9, 18, 0, 0, 0, time.UTC), time.UTC)
	if !from.Equal(time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)) || !to.Equal(time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("bounds %s %s", from, to)
	}

	sun := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	if OpenToday(sun, w) {
		t.Fatal("sunday not open")
	}
	if !OpenToday(mon10, w) {
		t.Fatal("monday open")
	}
	if OpenToday(time.Date(2026, 11, 9, 18, 0, 0, 0, time.UTC), w) {
		t.Fatal("after close")
	}
}
