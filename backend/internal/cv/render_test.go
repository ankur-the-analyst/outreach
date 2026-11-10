package cv

import (
	"bytes"
	"testing"
)

func TestRenderStampsFromNotSpec(t *testing.T) {
	t.Parallel()
	spec := Spec{Name: "Ada Lovelace", Headline: "Engineer"}
	a, err := PDF(spec, "a@outreach.test")
	if err != nil {
		t.Fatal(err)
	}
	b, err := PDF(spec, "b@outreach.test")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(a, []byte("%PDF-")) {
		t.Fatalf("want pdf:\n%s", a)
	}
	if !bytes.Contains(a, []byte("a@outreach.test")) {
		t.Fatalf("want from a:\n%s", a)
	}
	if bytes.Contains(a, []byte("b@outreach.test")) {
		t.Fatal("a must not contain b")
	}
	if !bytes.Contains(b, []byte("b@outreach.test")) {
		t.Fatalf("want from b:\n%s", b)
	}
}
