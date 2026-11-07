package suppression

import (
	"bytes"
	"testing"

	"outreach/internal/crypto"
)

func TestHashNormalizesEmail(t *testing.T) {
	t.Parallel()
	k, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	a, na, err := Hash(k, KindEmail, " You@Ex.CO ")
	if err != nil {
		t.Fatal(err)
	}
	b, nb, err := Hash(k, KindEmail, "you@ex.co")
	if err != nil {
		t.Fatal(err)
	}
	if na != "you@ex.co" || nb != na || !bytes.Equal(a, b) {
		t.Fatalf("%s %s %x %x", na, nb, a, b)
	}
	if DomainOf("You@Ex.CO") != "ex.co" {
		t.Fatalf("%s", DomainOf("You@Ex.CO"))
	}
}
