package suppression

import (
	"strings"
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
	ws := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	tok, err := Token(k, ws, " You@Ex.CO ")
	if err != nil {
		t.Fatal(err)
	}
	gotWS, email, err := ParseToken(k, tok)
	if err != nil || gotWS != ws || email != "you@ex.co" {
		t.Fatalf("%s %s %v", gotWS, email, err)
	}
	if _, _, err := ParseToken(k, tok+"x"); err == nil {
		t.Fatal("want invalid")
	}
}

func TestOneClickURLUsesPublicPath(t *testing.T) {
	t.Parallel()
	k, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ws := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	u, err := OneClickURL("http://127.0.0.1:8080/", k, ws, "ada@acme.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "http://127.0.0.1:8080/u/") {
		t.Fatalf("%s", u)
	}
}
