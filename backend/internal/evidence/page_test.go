package evidence

import (
	"testing"

	"github.com/google/uuid"
)

func TestClipPageSetsCursorWhenMoreRemain(t *testing.T) {
	t.Parallel()
	a := uuid.MustParse("01700000-0000-7000-8000-00000000000a")
	b := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	c := uuid.MustParse("01700000-0000-7000-8000-00000000000c")
	got := clipPage([]Item{{ID: a}, {ID: b}, {ID: c}}, 2)
	if len(got.Evidence) != 2 || got.Next == nil || *got.Next != b {
		t.Fatalf("%+v", got)
	}
	short := clipPage([]Item{{ID: a}}, 2)
	if short.Next != nil || len(short.Evidence) != 1 {
		t.Fatalf("%+v", short)
	}
}

func TestClampLimit(t *testing.T) {
	t.Parallel()
	if clampLimit(0) != defaultPage || clampLimit(500) != maxPage || clampLimit(10) != 10 {
		t.Fatal("clamp")
	}
}
