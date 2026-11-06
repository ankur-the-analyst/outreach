package auth

import "testing"

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
