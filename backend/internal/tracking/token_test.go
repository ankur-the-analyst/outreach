package tracking

import (
	"testing"

	"github.com/google/uuid"
	"outreach/internal/crypto"
)

func TestTokenRoundTrip(t *testing.T) {
	t.Parallel()
	k, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	in := TokenData{
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		DraftID:     uuid.MustParse("01700000-0000-7000-8000-000000000002"),
		Kind:        KindClick,
		URL:         "https://jobs.example/ada",
	}
	tok, err := Token(k, in)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseToken(k, tok)
	if err != nil || got != in {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := ParseToken(k, tok+"x"); err == nil {
		t.Fatal("want invalid")
	}
	open := in
	open.Kind = KindOpen
	open.URL = ""
	tok, err = Token(k, open)
	if err != nil {
		t.Fatal(err)
	}
	got, err = ParseToken(k, tok)
	if err != nil || got != open {
		t.Fatalf("open %+v %v", got, err)
	}
}
