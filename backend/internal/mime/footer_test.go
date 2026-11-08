package mime

import (
	"strings"
	"testing"
)

func TestCommercialFooter(t *testing.T) {
	t.Parallel()
	got := CommercialFooter("Ada", "ada@acme.com", "1 Main St", "http://127.0.0.1:8080/u")
	for _, want := range []string{"Ada", "ada@acme.com", "1 Main St", "Unsubscribe: http://127.0.0.1:8080/u"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestWithFooterHTMLEscapes(t *testing.T) {
	t.Parallel()
	text, htmlBody := WithFooter("hi", "<p>hi</p>", "Ada <office>")
	if !strings.Contains(text, "Ada <office>") {
		t.Fatalf("%q", text)
	}
	if strings.Contains(htmlBody, "Ada <office>") || !strings.Contains(htmlBody, "Ada &lt;office&gt;") {
		t.Fatalf("%q", htmlBody)
	}
}
