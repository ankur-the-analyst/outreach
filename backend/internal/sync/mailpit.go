package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Item struct {
	RFCMessageID string
	InReplyTo    string
	Subject      string
	From         string
	To           []string
	Text         string
	At           time.Time
}

type Source interface {
	List(ctx context.Context) ([]Item, error)
}

type Mailpit struct {
	Base string
	HTTP *http.Client
}

func (m Mailpit) List(ctx context.Context) ([]Item, error) {
	client := m.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	u := strings.TrimRight(m.Base, "/") + "/api/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("sync: mailpit list %s", res.Status)
	}
	var body struct {
		Messages []struct {
			ID        string `json:"ID"`
			MessageID string `json:"MessageID"`
			Subject   string `json:"Subject"`
			Created   string `json:"Created"`
			From      struct {
				Address string `json:"Address"`
			} `json:"From"`
			To []struct {
				Address string `json:"Address"`
			} `json:"To"`
			Snippet string `json:"Snippet"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(body.Messages))
	for _, raw := range body.Messages {
		to := make([]string, 0, len(raw.To))
		for _, a := range raw.To {
			if a.Address != "" {
				to = append(to, a.Address)
			}
		}
		at, _ := time.Parse(time.RFC3339, raw.Created)
		item := Item{
			RFCMessageID: normalizeID(raw.MessageID),
			Subject:      raw.Subject,
			From:         strings.ToLower(raw.From.Address),
			To:           to,
			Text:         raw.Snippet,
			At:           at,
		}
		if raw.ID != "" {
			if inReply, text, err := m.detail(ctx, client, raw.ID); err == nil {
				item.InReplyTo = normalizeID(inReply)
				if text != "" {
					item.Text = text
				}
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (m Mailpit) detail(ctx context.Context, client *http.Client, id string) (inReply, text string, err error) {
	u := strings.TrimRight(m.Base, "/") + "/api/v1/message/" + id
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", "", fmt.Errorf("sync: mailpit message %s", res.Status)
	}
	var body struct {
		InReplyTo []string `json:"InReplyTo"`
		Text      string   `json:"Text"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", "", err
	}
	if len(body.InReplyTo) > 0 {
		inReply = body.InReplyTo[0]
	}
	return inReply, body.Text, nil
}

func normalizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if !strings.HasPrefix(id, "<") {
		id = "<" + id
	}
	if !strings.HasSuffix(id, ">") {
		id += ">"
	}
	return id
}
