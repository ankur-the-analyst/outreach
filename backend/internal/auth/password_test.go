package auth

import "testing"

func TestRandomPassword(t *testing.T) {
	t.Parallel()
	a, err := RandomPassword()
	if err != nil || len(a) < minPassword {
		t.Fatalf("%q %v", a, err)
	}
	b, err := RandomPassword()
	if err != nil || a == b {
		t.Fatalf("got same password")
	}
}

func TestHashPasswordRoundTrip(t *testing.T) {
	t.Parallel()
	h, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(h, "correct-horse") {
		t.Fatal("expected match")
	}
	if VerifyPassword(h, "wrong") {
		t.Fatal("expected mismatch")
	}
	if VerifyPassword("not-a-hash", "correct-horse") {
		t.Fatal("expected reject")
	}
}
