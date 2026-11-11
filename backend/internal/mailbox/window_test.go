package mailbox

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/pacing"
)

func TestApplyWindowSunday(t *testing.T) {
	t.Parallel()
	weekdays := []int32{1, 2, 3, 4, 5}
	nine := pgtype.Time{Microseconds: 9 * 3600 * 1_000_000, Valid: true}
	five := pgtype.Time{Microseconds: 17 * 3600 * 1_000_000, Valid: true}
	w := pacing.WindowFrom(weekdays, nine, five, time.UTC)
	var m Mailbox
	applyWindow(time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC), w, &m)
	if m.OpenToday || m.SendingNow {
		t.Fatalf("%+v", m)
	}
	want := time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC)
	if !m.NextOpen.Equal(want) {
		t.Fatalf("next %s", m.NextOpen)
	}
}

func TestApplyWindowMondayMorning(t *testing.T) {
	t.Parallel()
	weekdays := []int32{1, 2, 3, 4, 5}
	nine := pgtype.Time{Microseconds: 9 * 3600 * 1_000_000, Valid: true}
	five := pgtype.Time{Microseconds: 17 * 3600 * 1_000_000, Valid: true}
	w := pacing.WindowFrom(weekdays, nine, five, time.UTC)
	var m Mailbox
	applyWindow(time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC), w, &m)
	if !m.OpenToday || !m.SendingNow {
		t.Fatalf("%+v", m)
	}
}
