package mime

import (
	"bytes"
	"testing"
)

func TestBuildUsesProvidedMessageID(t *testing.T) {
	t.Parallel()
	id := "<stable@outreach.test>"
	built, err := Build(Message{
		From: "dev@example.test", To: []string{"sink@outreach.test"},
		Subject: "hello", Text: "hi", HTML: "<p>hi</p>", MessageID: id,
	})
	if err != nil {
		t.Fatal(err)
	}
	if built.MessageID != id {
		t.Fatalf("id %q", built.MessageID)
	}
	if !bytes.Contains(built.Bytes, []byte("Message-ID: "+id)) {
		t.Fatalf("bytes missing Message-ID:\n%s", built.Bytes)
	}
	if bytes.Contains(built.Bytes, []byte("Precedence:")) {
		t.Fatal("must not set Precedence")
	}
	if built.From != "dev@example.test" || len(built.To) != 1 {
		t.Fatalf("%+v", built)
	}
}

func TestBuildRFC8058Headers(t *testing.T) {
	t.Parallel()
	u := "http://127.0.0.1:8080/api/v1/unsubscribe/tok"
	built, err := Build(Message{
		From: "dev@example.test", To: []string{"sink@outreach.test"},
		Text: "hi", ListUnsubscribe: u,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(built.Bytes, []byte("List-Unsubscribe: <"+u+">")) {
		t.Fatalf("missing List-Unsubscribe:\n%s", built.Bytes)
	}
	if !bytes.Contains(built.Bytes, []byte("List-Unsubscribe-Post: List-Unsubscribe=One-Click")) {
		t.Fatalf("missing List-Unsubscribe-Post:\n%s", built.Bytes)
	}
	if bytes.Contains(built.Bytes, []byte("Precedence:")) {
		t.Fatal("must not set Precedence")
	}
}

func TestBuildGeneratesMessageID(t *testing.T) {
	t.Parallel()
	built, err := Build(Message{From: "a@x.test", To: []string{"b@x.test"}, Text: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if built.MessageID == "" || !bytes.Contains(built.Bytes, []byte("Message-ID: "+built.MessageID)) {
		t.Fatalf("%q", built.MessageID)
	}
}
