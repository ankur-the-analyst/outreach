package templates

import (
	"errors"
	"testing"
)

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
