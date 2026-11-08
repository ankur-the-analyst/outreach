package campaign

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/ai"
	"outreach/internal/evidence"
)

func TestWriteBodyUsesCompleter(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	p := NewPostgres(nil, nil).WithWriter(&ai.Fake{
		Text: "You closed 12 deals last year. Open to a short chat?",
	})
	cited := []evidence.Item{{ID: id, Text: "closed 12 deals last year"}}
	facts := []draftFact{{Claim: "closed 12 deals last year", EvidenceID: id}}
	body, err := p.writeBody(context.Background(), uuid.Nil, "met at a meetup", cited, facts)
	if err != nil {
		t.Fatal(err)
	}
	if body != "You closed 12 deals last year. Open to a short chat?" {
		t.Fatalf("%q", body)
	}
}

func TestWriteBodyKeepsNotesWithoutWriter(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	p := NewPostgres(nil, nil)
	cited := []evidence.Item{{ID: id, Text: "closed 12 deals last year"}}
	facts := []draftFact{{Claim: "closed 12 deals last year", EvidenceID: id}}
	body, err := p.writeBody(context.Background(), uuid.Nil, "met at a meetup", cited, facts)
	if err != nil || body != "met at a meetup" {
		t.Fatalf("%q %v", body, err)
	}
}

func TestWriteBodyKeepsNotesWithoutClaims(t *testing.T) {
	t.Parallel()
	p := NewPostgres(nil, nil).WithWriter(&ai.Fake{Text: "should not run"})
	body, err := p.writeBody(context.Background(), uuid.Nil, "met at a meetup", nil, nil)
	if err != nil || body != "met at a meetup" {
		t.Fatalf("%q %v", body, err)
	}
}
