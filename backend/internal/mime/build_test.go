package mime

import (
	"bytes"
	"errors"
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
	u := "http://127.0.0.1:8080/u/tok"
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

func TestBuildAttachmentDisposition(t *testing.T) {
	t.Parallel()
	built, err := Build(Message{
		From: "dev@example.test", To: []string{"sink@outreach.test"},
		Subject: "cv", Text: "hi", HTML: "<p>hi</p>",
		Files: []File{{Name: "cv.pdf", ContentType: "application/pdf", Bytes: []byte("%PDF-1.4")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := built.Bytes
	if !bytes.Contains(raw, []byte("multipart/mixed")) {
		t.Fatalf("want mixed:\n%s", raw)
	}
	if !bytes.Contains(raw, []byte("multipart/alternative")) {
		t.Fatalf("want alternative inside mixed:\n%s", raw)
	}
	if !bytes.Contains(raw, []byte("Content-Disposition: attachment")) {
		t.Fatalf("want attachment:\n%s", raw)
	}
	if !bytes.Contains(raw, []byte("filename=cv.pdf")) && !bytes.Contains(raw, []byte(`filename="cv.pdf"`)) {
		t.Fatalf("want filename:\n%s", raw)
	}
}

func TestBuildNoFilesStaysAlternative(t *testing.T) {
	t.Parallel()
	built, err := Build(Message{
		From: "dev@example.test", To: []string{"sink@outreach.test"},
		Text: "hi", HTML: "<p>hi</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(built.Bytes, []byte("multipart/mixed")) {
		t.Fatalf("unexpected mixed:\n%s", built.Bytes)
	}
	if !bytes.Contains(built.Bytes, []byte("multipart/alternative")) {
		t.Fatalf("want alternative:\n%s", built.Bytes)
	}
	if bytes.Contains(built.Bytes, []byte("Content-Disposition: attachment")) {
		t.Fatal("unexpected attachment")
	}
}

func TestBuildTooLarge(t *testing.T) {
	t.Parallel()
	_, err := Build(Message{
		From: "dev@example.test", To: []string{"sink@outreach.test"},
		Text:  "hi",
		Files: []File{{Name: "big.bin", Bytes: make([]byte, MaxBytes+1)}},
	})
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err %v", err)
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
