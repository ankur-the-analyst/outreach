package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"outreach/internal/mailbox"
	"outreach/internal/mime"
)

func TestSendAndFetchHistory(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/gmail/v1/users/me/messages/send", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Raw string `json:"raw"`
		}
		_ = json.Unmarshal(raw, &body)
		if _, err := base64.RawURLEncoding.DecodeString(body.Raw); err != nil {
			t.Errorf("raw: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "m1"})
	})
	mux.HandleFunc("/gmail/v1/users/me/profile", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"emailAddress": "a@gmail.com", "historyId": "99"})
	})
	mux.HandleFunc("/gmail/v1/users/me/history", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("startHistoryId") != "10" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"historyId": "20",
			"history": []map[string]any{{
				"messagesAdded": []map[string]any{{"message": map[string]string{"id": "abc"}}},
			}},
		})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages/abc", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           "abc",
			"internalDate": "1700000000000",
			"snippet":      "hello",
			"payload": map[string]any{
				"headers": []map[string]string{
					{"name": "From", "value": "Pat <pat@x.test>"},
					{"name": "To", "value": "a@gmail.com"},
					{"name": "Subject", "value": "Hi"},
					{"name": "Message-ID", "value": "<m@x>"},
					{"name": "In-Reply-To", "value": "<prev@x>"},
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Base: srv.URL}

	sent, err := c.Send(context.Background(), &mime.Built{Bytes: []byte("From: a\r\n\r\nhi")})
	if err != nil || sent.ProviderID != "m1" {
		t.Fatalf("send %+v %v", sent, err)
	}
	res, err := c.Fetch(context.Background(), mailbox.FetchCursor{Token: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Cursor.Token != "20" || len(res.Messages) != 1 {
		t.Fatalf("%+v", res)
	}
	m := res.Messages[0]
	if m.From != "pat@x.test" || m.RFCMessageID != "<m@x>" || m.InReplyTo != "<prev@x>" {
		t.Fatalf("%+v", m)
	}
}

func TestFetch404FallsBackToFullSync(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/gmail/v1/users/me/history", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/gmail/v1/users/me/profile", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"historyId": "7"})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "newer_than") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []any{}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Base: srv.URL}
	res, err := c.Fetch(context.Background(), mailbox.FetchCursor{Token: "1"})
	if err != nil || res.Cursor.Token != "7" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestFactoryNeedsRefreshToken(t *testing.T) {
	t.Parallel()
	f := NewFactory(Config{ClientID: "id", ClientSecret: "sec"})
	_, err := f(mailbox.Account{AuthKind: "oauth", Provider: "gmail"})
	if err != mailbox.ErrNotConfigured {
		t.Fatalf("got %v", err)
	}
}
