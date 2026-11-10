package fields

import "testing"

func TestRejectsBuiltinAndBadKey(t *testing.T) {
	t.Parallel()
	if !IsBuiltin("contact", "email") || IsBuiltin("contact", "linkedin") {
		t.Fatal("builtins")
	}
	if ValidKey("Email") || !ValidKey("linkedin") {
		t.Fatal("key")
	}
	if NormalizeHeader("Work Email") != "work_email" {
		t.Fatal("header")
	}
}
