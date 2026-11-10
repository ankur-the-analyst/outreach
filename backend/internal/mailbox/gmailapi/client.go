package gmailapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"outreach/internal/mailbox"
	"outreach/internal/mime"
)

type Key struct {
	ClientEmail string `json:"client_email"`
	ClientID    string `json:"client_id"`
	PrivateKey  string `json:"private_key"`
	Type        string `json:"type"`
}

func ParseKey(raw []byte) (Key, error) {
	var k Key
	if err := json.Unmarshal(raw, &k); err != nil {
		return Key{}, fmt.Errorf("gmailapi: key json: %w", err)
	}
	if k.ClientEmail == "" || k.PrivateKey == "" {
		return Key{}, fmt.Errorf("gmailapi: client_email and private_key required")
	}
	return k, nil
}

type Client struct {
	domain  string
	subject string
	key     Key
}

func New(domain, subject string, key Key) *Client {
	return &Client{
		domain:  strings.ToLower(strings.TrimSpace(domain)),
		subject: strings.ToLower(strings.TrimSpace(subject)),
		key:     key,
	}
}

var _ mailbox.Transport = (*Client)(nil)

func (c *Client) Test(context.Context) error {
	if mailbox.IsGmail("user@" + c.domain) {
		return mailbox.ErrGmailServiceAccount
	}
	_, subDom, _ := strings.Cut(c.subject, "@")
	if c.domain == "" || subDom != c.domain {
		return fmt.Errorf("gmailapi: subject must be on %s", c.domain)
	}
	if c.key.ClientEmail == "" || c.key.PrivateKey == "" {
		return fmt.Errorf("gmailapi: incomplete key")
	}
	return nil
}

func (c *Client) Send(ctx context.Context, m *mime.Built) (mailbox.SendResult, error) {
	if err := c.Test(ctx); err != nil {
		return mailbox.SendResult{}, err
	}
	return mailbox.SendResult{}, fmt.Errorf("gmailapi: send not wired")
}

func (c *Client) AppendSent(context.Context, *mime.Built) error {
	return nil
}

func (c *Client) FindSent(context.Context, string) (bool, error) {
	return false, fmt.Errorf("gmailapi: find not wired")
}

func (c *Client) Fetch(context.Context, mailbox.FetchCursor) (mailbox.FetchResult, error) {
	return mailbox.FetchResult{}, mailbox.ErrNotSupported
}
