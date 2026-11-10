package smtpimap

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/mime"
)

func TestMailpitAttachment(t *testing.T) {
	smtpAddr := os.Getenv("MAILPIT_SMTP")
	api := os.Getenv("MAILPIT_API")
	if smtpAddr == "" || api == "" {
		t.Skip("MAILPIT_SMTP or MAILPIT_API not set")
	}
	host, portStr, err := net.SplitHostPort(smtpAddr)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	c := New(Config{SMTPHost: host, SMTPPort: port})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	built, err := mime.Build(mime.Message{
		From: "sender@outreach.test", To: []string{"sink@outreach.test"},
		Subject: "m8.1 attach", Text: "hi", HTML: "<p>hi</p>",
		Files: []mime.File{{Name: "cv.pdf", ContentType: "application/pdf", Bytes: []byte("%PDF-1.4")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Send(ctx, built); err != nil {
		t.Fatal(err)
	}
	want := strings.Trim(built.MessageID, "<>")
	deadline := time.Now().Add(5 * time.Second)
	for {
		n, err := mailpitAttachmentCount(ctx, api, want)
		if err == nil && n == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("mailpit attachments for %s: n=%d err=%v", want, n, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestMailpit(t *testing.T) {
	smtpAddr := os.Getenv("MAILPIT_SMTP")
	if smtpAddr == "" {
		t.Skip("MAILPIT_SMTP not set")
	}
	host, portStr, err := net.SplitHostPort(smtpAddr)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	c := New(Config{SMTPHost: host, SMTPPort: port})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Test(ctx); err != nil {
		t.Fatalf("Test: %v", err)
	}

	id := uuid.Must(uuid.NewV7()).String()
	msgid := "<" + id + "@outreach.test>"
	raw := []byte("From: sender@outreach.test\r\nTo: sink@outreach.test\r\nMessage-ID: " + msgid + "\r\nSubject: m2.2\r\n\r\nhello\r\n")
	_, err = c.Send(ctx, &mime.Built{
		MessageID: msgid,
		From:      "sender@outreach.test",
		To:        []string{"sink@outreach.test"},
		Bytes:     raw,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	api := os.Getenv("MAILPIT_API")
	if api == "" {
		return
	}
	want := id + "@outreach.test"
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok, err := mailpitHas(ctx, api, want)
		if err == nil && ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("mailpit did not see %s: err=%v", want, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func mailpitAttachmentCount(ctx context.Context, api, messageID string) (int, error) {
	id, err := mailpitID(ctx, api, messageID)
	if err != nil {
		return 0, err
	}
	if id == "" {
		return 0, nil
	}
	u := strings.TrimRight(api, "/") + "/api/v1/message/" + id
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status %d", res.StatusCode)
	}
	var body struct {
		Attachments []struct {
			FileName string `json:"FileName"`
		} `json:"Attachments"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return 0, err
	}
	return len(body.Attachments), nil
}

func mailpitID(ctx context.Context, api, messageID string) (string, error) {
	u := strings.TrimRight(api, "/") + "/api/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", res.StatusCode)
	}
	var body struct {
		Messages []struct {
			ID        string `json:"ID"`
			MessageID string `json:"MessageID"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", err
	}
	for _, m := range body.Messages {
		if m.MessageID == messageID {
			return m.ID, nil
		}
	}
	return "", nil
}

func mailpitHas(ctx context.Context, api, messageID string) (bool, error) {
	u := strings.TrimRight(api, "/") + "/api/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("status %d", res.StatusCode)
	}
	var body struct {
		Messages []struct {
			MessageID string `json:"MessageID"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return false, err
	}
	for _, m := range body.Messages {
		if m.MessageID == messageID {
			return true, nil
		}
	}
	return false, nil
}
