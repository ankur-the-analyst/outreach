package tracking

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/crypto"
)

func TestInjectWrapsHttpAndSkipsUnsub(t *testing.T) {
	t.Parallel()
	k, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ws := uuid.MustParse("01700000-0000-7000-8000-000000000001")
	draft := uuid.MustParse("01700000-0000-7000-8000-000000000002")
	unsub := "http://127.0.0.1:8080/u/abc.def"
	in := `<p>Hi</p><a href="https://jobs.example/ada">Ada</a>` +
		`<a href="mailto:ada@acme.com">mail</a><a href="#top">top</a>` +
		`<a href="` + unsub + `">stop</a>` +
		`<a href="http://127.0.0.1:8080/t/c/already">old</a>`
	got, err := Inject(InjectInput{
		HTML: in, Host: "http://127.0.0.1:8080", HMAC: k,
		WorkspaceID: ws, DraftID: draft, Opens: true, Clicks: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, `href="https://jobs.example/ada"`) {
		t.Fatalf("job href not wrapped: %s", got)
	}
	if !strings.Contains(got, `href="mailto:ada@acme.com"`) || !strings.Contains(got, `href="#top"`) {
		t.Fatalf("skipped hrefs changed: %s", got)
	}
	if !strings.Contains(got, unsub) {
		t.Fatalf("unsub rewritten: %s", got)
	}
	if !strings.Contains(got, `href="http://127.0.0.1:8080/t/c/already"`) {
		t.Fatalf("existing /t/ rewritten: %s", got)
	}
	if !strings.Contains(got, `/t/c/`) || !strings.Contains(got, `<img src="http://127.0.0.1:8080/t/o/`) {
		t.Fatalf("missing wraps: %s", got)
	}
	clickTok := between(got, `http://127.0.0.1:8080/t/c/`, `"`)
	parsed, err := ParseToken(k, clickTok)
	if err != nil || parsed.Kind != KindClick || parsed.URL != "https://jobs.example/ada" {
		t.Fatalf("click token %+v %v", parsed, err)
	}
}

func TestInjectRequiresHostWhenFlagsOn(t *testing.T) {
	t.Parallel()
	k, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Inject(InjectInput{
		HTML: "<p>Hi</p>", HMAC: k,
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		DraftID:     uuid.MustParse("01700000-0000-7000-8000-000000000002"),
		Opens:       true,
	})
	if err != ErrHost {
		t.Fatalf("%v", err)
	}
}

func between(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}
	s = s[i+len(a):]
	j := strings.Index(s, b)
	if j < 0 {
		return s
	}
	return s[:j]
}
