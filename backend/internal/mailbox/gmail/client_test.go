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
	"time"

	"outreach/internal/mailbox"
	"outreach/internal/mime"
	"outreach/internal/storage"
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
			"snippet":      "you&#39;ll now be able to ask",
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
	if m.Text != "you'll now be able to ask" {
		t.Fatalf("snippet %q", m.Text)
	}
}

func TestFetchHTMLBody(t *testing.T) {
	t.Parallel()
	plain := base64.RawURLEncoding.EncodeToString([]byte("Hello Ada"))
	page := base64.RawURLEncoding.EncodeToString([]byte("<p>Hello <b>Ada</b></p>"))
	mux := http.NewServeMux()
	mux.HandleFunc("/gmail/v1/users/me/profile", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"historyId": "1"})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages" {
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []map[string]string{{"id": "abc"}}})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages/abc", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "full" {
			t.Errorf("format %s", r.URL.Query().Get("format"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "abc",
			"snippet": "Hello Ada",
			"payload": map[string]any{
				"mimeType": "multipart/alternative",
				"headers": []map[string]string{
					{"name": "From", "value": "pat@x.test"},
					{"name": "Subject", "value": "Hi"},
					{"name": "Message-ID", "value": "<m@x>"},
				},
				"parts": []map[string]any{
					{"mimeType": "text/plain", "body": map[string]string{"data": plain}},
					{"mimeType": "text/html", "body": map[string]string{"data": page}},
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Base: srv.URL}
	res, err := c.Fetch(context.Background(), mailbox.FetchCursor{})
	if err != nil || len(res.Messages) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	m := res.Messages[0]
	if m.Text != "Hello Ada" || m.HTML != "<p>Hello <b>Ada</b></p>" {
		t.Fatalf("%+v", m)
	}
}

func TestFetchHTMLAttachment(t *testing.T) {
	t.Parallel()
	page := base64.RawURLEncoding.EncodeToString([]byte("<p>Full <b>letter</b></p>"))
	mux := http.NewServeMux()
	mux.HandleFunc("/gmail/v1/users/me/profile", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"historyId": "1"})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages" {
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": []map[string]string{{"id": "abc"}}})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages/abc/attachments/att1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"data": page})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages/abc", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "abc",
			"snippet": "Full letter",
			"payload": map[string]any{
				"mimeType": "text/html",
				"headers": []map[string]string{
					{"name": "From", "value": "pat@x.test"},
					{"name": "Message-ID", "value": "<m@x>"},
				},
				"body": map[string]string{"attachmentId": "att1"},
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Base: srv.URL}
	res, err := c.Fetch(context.Background(), mailbox.FetchCursor{})
	if err != nil || len(res.Messages) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	m := res.Messages[0]
	if m.Text != "Full letter" || m.HTML != "<p>Full <b>letter</b></p>" {
		t.Fatalf("%+v", m)
	}
}

func TestListPagesAndInlineImage(t *testing.T) {
	t.Parallel()
	png := base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3})
	page := base64.RawURLEncoding.EncodeToString([]byte(`<img src="cid:logo@x">`))
	mux := http.NewServeMux()
	mux.HandleFunc("/gmail/v1/users/me/profile", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"historyId": "9"})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages" {
			return
		}
		if r.URL.Query().Get("maxResults") == "" {
			t.Errorf("missing maxResults")
		}
		if r.URL.Query().Get("pageToken") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages":      []map[string]string{{"id": "a"}},
				"nextPageToken": "p2",
			})
			return
		}
		if r.URL.Query().Get("pageToken") != "p2" {
			t.Errorf("page %s", r.URL.Query().Get("pageToken"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]string{{"id": "b"}},
		})
	})
	write := func(id, html string, withImage bool) {
		mux.HandleFunc("/gmail/v1/users/me/messages/"+id, func(w http.ResponseWriter, r *http.Request) {
			payload := map[string]any{
				"mimeType": "multipart/related",
				"headers": []map[string]string{
					{"name": "From", "value": "pat@x.test"},
					{"name": "Message-ID", "value": "<" + id + "@x>"},
				},
				"parts": []map[string]any{
					{"mimeType": "text/html", "body": map[string]string{"data": html}},
				},
			}
			if withImage {
				payload["parts"] = []map[string]any{
					{"mimeType": "text/html", "body": map[string]string{"data": html}},
					{
						"mimeType": "image/png",
						"headers":  []map[string]string{{"name": "Content-Id", "value": "<logo@x>"}},
						"body":     map[string]string{"data": png},
					},
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": id, "snippet": id, "payload": payload,
			})
		})
	}
	write("a", page, true)
	write("b", base64.RawURLEncoding.EncodeToString([]byte("<p>second</p>")), false)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Base: srv.URL}
	res, err := c.Fetch(context.Background(), mailbox.FetchCursor{})
	if err != nil || len(res.Messages) != 2 {
		t.Fatalf("%+v %v", res, err)
	}
	if !strings.Contains(res.Messages[0].HTML, "data:image/png;base64,") || strings.Contains(res.Messages[0].HTML, "cid:") {
		t.Fatalf("image not embedded: %s", res.Messages[0].HTML)
	}
	if res.Messages[1].HTML != "<p>second</p>" {
		t.Fatalf("%s", res.Messages[1].HTML)
	}
}

func TestImageSavedAndDateQuery(t *testing.T) {
	t.Parallel()
	png := base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3})
	page := base64.RawURLEncoding.EncodeToString([]byte(`<img src="cid:logo@x">`))
	mux := http.NewServeMux()
	mux.HandleFunc("/gmail/v1/users/me/profile", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"historyId": "3"})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages" {
			return
		}
		q := r.URL.Query().Get("q")
		if !strings.Contains(q, "after:2026/11/13") || !strings.Contains(q, "before:2026/11/16") {
			t.Errorf("query %s", q)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]string{{"id": "a"}},
		})
	})
	mux.HandleFunc("/gmail/v1/users/me/messages/a", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "a", "snippet": "hi",
			"payload": map[string]any{
				"mimeType": "multipart/related",
				"headers": []map[string]string{
					{"name": "From", "value": "pat@x.test"},
					{"name": "Message-ID", "value": "<a@x>"},
				},
				"parts": []map[string]any{
					{"mimeType": "text/html", "body": map[string]string{"data": page}},
					{
						"mimeType": "image/png",
						"headers":  []map[string]string{{"name": "Content-Id", "value": "<logo@x>"}},
						"body":     map[string]string{"data": png},
					},
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	files := storage.NewMemory()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, Files: files, FileBase: "http://api.test"}
	res, err := c.Fetch(context.Background(), mailbox.FetchCursor{
		After:  time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		Before: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil || len(res.Messages) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	htmlBody := res.Messages[0].HTML
	if strings.Contains(htmlBody, "cid:") || !strings.Contains(htmlBody, "http://api.test/api/v1/mail-files/mail-images/") {
		t.Fatalf("html %s", htmlBody)
	}
	key := strings.TrimPrefix(htmlBody, `<img src="http://api.test/api/v1/mail-files/`)
	key = strings.TrimSuffix(key, `">`)
	got, err := files.Get(context.Background(), key)
	if err != nil || string(got) != string([]byte{1, 2, 3}) {
		t.Fatalf("file %q %v %v", key, got, err)
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
