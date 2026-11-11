package jobsjd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/cv"
	"outreach/internal/storage"
)

func TestExtractPlainAndPDF(t *testing.T) {
	t.Parallel()
	got, err := Extract("text/plain", "r.txt", []byte("Closed 12 deals last year."))
	if err != nil || got != "Closed 12 deals last year." {
		t.Fatalf("%q %v", got, err)
	}
	pdf, err := cv.PDF(cv.Spec{
		Name:     "Ada",
		Sections: []cv.Section{{Title: "Work", Lines: []string{"Closed 12 deals last year."}}},
	}, "ada@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("fixture:\n%s", pdf[:20])
	}
	text, err := Extract("application/pdf", "cv.pdf", pdf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Closed 12 deals last year.") && !strings.Contains(text, "Ada") {
		t.Fatalf("extract %q", text)
	}
}

func TestMemoryKeepsPDFBytes(t *testing.T) {
	t.Parallel()
	s := storage.NewMemory()
	ctx := context.Background()
	key := uuid.MustParse("01700000-0000-7000-8000-000000000009").String() + "/resumes/x"
	raw := []byte("%PDF-1.4 fixture")
	if err := s.Put(ctx, key, "application/pdf", raw); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, key)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("%q %v", got, err)
	}
}
