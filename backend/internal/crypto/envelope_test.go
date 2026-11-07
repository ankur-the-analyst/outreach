package crypto

import (
	"bytes"
	"errors"
	"testing"
)

const (
	keyV1 = "1:0000000000000000000000000000000000000000000000000000000000000001"
	keyV2 = "2:0000000000000000000000000000000000000000000000000000000000000002"
)

func TestSealOpenRoundTrip(t *testing.T) {
	t.Parallel()
	k, err := NewKeyring(keyV1, 1)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("app-password-secret")
	sealed, err := k.Seal(plain)
	if err != nil {
		t.Fatal(err)
	}
	if sealed.KeyVersion != 1 || len(sealed.Nonce) != nonceSize || len(sealed.WrappedDEK) == 0 {
		t.Fatalf("sealed shape: %+v", sealed)
	}
	got, err := k.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestOpenWrongKey(t *testing.T) {
	t.Parallel()
	a, err := NewKeyring(keyV1, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewKeyring("1:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 1)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := a.Seal([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Open(sealed); !errors.Is(err, ErrOpen) {
		t.Fatalf("got %v want %v", err, ErrOpen)
	}
}

func TestOpenTamperedCiphertext(t *testing.T) {
	t.Parallel()
	k, err := NewKeyring(keyV1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := k.Seal([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	sealed.Ciphertext[0] ^= 0xff
	if _, err := k.Open(sealed); !errors.Is(err, ErrOpen) {
		t.Fatalf("got %v want %v", err, ErrOpen)
	}
}

func TestOpenAfterRotation(t *testing.T) {
	t.Parallel()
	old, err := NewKeyring(keyV1, 1)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := old.Seal([]byte("keep"))
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := NewKeyring(keyV1+","+keyV2, 2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := rotated.Open(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep" {
		t.Fatalf("got %q", got)
	}
	fresh, err := rotated.Seal([]byte("new"))
	if err != nil {
		t.Fatal(err)
	}
	if fresh.KeyVersion != 2 {
		t.Fatalf("version %d", fresh.KeyVersion)
	}
}

func TestNewKeyringRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		spec   string
		active int
	}{
		{"", 1},
		{"1:ab", 1},
		{keyV1, 2},
		{"n:0000000000000000000000000000000000000000000000000000000000000001", 1},
	}
	for _, tt := range tests {
		if _, err := NewKeyring(tt.spec, tt.active); !errors.Is(err, ErrInvalidMasterKeys) {
			t.Fatalf("spec %q active %d: got %v", tt.spec, tt.active, err)
		}
	}
}

func TestHMACStable(t *testing.T) {
	t.Parallel()
	k, err := NewKeyring(keyV1, 1)
	if err != nil {
		t.Fatal(err)
	}
	a := k.HMAC([]byte("email:you@ex.co"))
	b := k.HMAC([]byte("email:you@ex.co"))
	if len(a) != 32 || !bytes.Equal(a, b) {
		t.Fatalf("hmac %x %x", a, b)
	}
	if bytes.Equal(a, k.HMAC([]byte("email:other@ex.co"))) {
		t.Fatal("different input hashed equal")
	}
}
