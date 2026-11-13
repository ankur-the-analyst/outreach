package templates

import (
	"errors"
	"testing"
)

func TestValidateJSONWalksButtonHref(t *testing.T) {
	t.Parallel()
	err := ValidateJSON([]byte(`{"settings":{},"elements":[{"type":"button","href":"https://x.test?n={{contact.nope}}"}]}`))
	if !errors.Is(err, ErrUnknown) {
		t.Fatalf("%v", err)
	}
	if err := ValidateJSON([]byte(`{"elements":[{"href":"https://x.test?n={{contact.first_name}}"}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateJSON([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnknown(t *testing.T) {
	t.Parallel()
	if err := Validate("Hi {{nope}}"); !errors.Is(err, ErrUnknown) {
		t.Fatalf("%v", err)
	}
	if err := Validate("Hi {{contact.first_name | there}}"); err != nil {
		t.Fatal(err)
	}
}

func TestRenderUsesNameAndFallback(t *testing.T) {
	t.Parallel()
	got := Render("Hi {{contact.first_name | there}}", map[string]string{"contact.first_name": "Ada"})
	if got != "Hi Ada" {
		t.Fatalf("%q", got)
	}
	got = Render("Hi {{contact.first_name | there}}", map[string]string{})
	if got != "Hi there" {
		t.Fatalf("%q", got)
	}
}
