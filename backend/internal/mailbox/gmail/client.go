package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"outreach/internal/mailbox"
	"outreach/internal/mime"
)

const defaultAPIBase = "https://gmail.googleapis.com"

type Config struct {
	ClientID     string
	ClientSecret string
	HTTP         *http.Client
	APIBase      string
}

func NewFactory(cfg Config) mailbox.Factory {
	return func(acc mailbox.Account) (mailbox.Operations, error) {
		if len(acc.Secret) == 0 {
			return nil, mailbox.ErrNotConfigured
		}
		httpClient := cfg.HTTP
		if httpClient == nil {
			oc := &oauth2.Config{
				ClientID:     cfg.ClientID,
				ClientSecret: cfg.ClientSecret,
				Endpoint:     google.Endpoint,
			}
			httpClient = oc.Client(context.Background(), &oauth2.Token{RefreshToken: string(acc.Secret)})
		}
		base := cfg.APIBase
		if base == "" {
			base = defaultAPIBase
		}
		return &Client{HTTP: httpClient, Base: strings.TrimRight(base, "/")}, nil
	}
}

type Client struct {
	HTTP *http.Client
	Base string
}

var _ mailbox.Operations = (*Client)(nil)

func (c *Client) Test(ctx context.Context) error {
	_, err := c.profile(ctx)
	return err
}

func (c *Client) Send(ctx context.Context, m *mime.Built) (mailbox.SendResult, error) {
	if m == nil || len(m.Bytes) == 0 {
		return mailbox.SendResult{}, fmt.Errorf("gmail: empty message")
	}
	body, err := json.Marshal(map[string]string{"raw": base64.RawURLEncoding.EncodeToString(m.Bytes)})
	if err != nil {
		return mailbox.SendResult{}, err
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, "/gmail/v1/users/me/messages/send", body, &out); err != nil {
		return mailbox.SendResult{}, err
	}
	return mailbox.SendResult{ProviderID: out.ID}, nil
}

func (c *Client) AppendSent(context.Context, *mime.Built) error {
	return nil
}

func (c *Client) FindSent(ctx context.Context, messageID string) (bool, error) {
	q := strings.Trim(strings.TrimSpace(messageID), "<>")
	if q == "" {
		return false, fmt.Errorf("gmail: message id required")
	}
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	path := "/gmail/v1/users/me/messages?q=" + url.QueryEscape("rfc822msgid:"+q)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return false, err
	}
	return len(out.Messages) > 0, nil
}

func (c *Client) Fetch(ctx context.Context, cur mailbox.FetchCursor) (mailbox.FetchResult, error) {
	if cur.Token == "" {
		return c.fullSync(ctx)
	}
	res, err := c.history(ctx, cur.Token)
	if err != nil {
		if isNotFound(err) {
			return c.fullSync(ctx)
		}
		return mailbox.FetchResult{}, err
	}
	return res, nil
}

func (c *Client) fullSync(ctx context.Context) (mailbox.FetchResult, error) {
	prof, err := c.profile(ctx)
	if err != nil {
		return mailbox.FetchResult{}, err
	}
	var list struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := c.do(ctx, http.MethodGet, "/gmail/v1/users/me/messages?q="+url.QueryEscape("newer_than:7d")+"&maxResults=50", nil, &list); err != nil {
		return mailbox.FetchResult{}, err
	}
	msgs := make([]mailbox.Message, 0, len(list.Messages))
	for _, raw := range list.Messages {
		m, err := c.getMessage(ctx, raw.ID)
		if err != nil {
			return mailbox.FetchResult{}, err
		}
		msgs = append(msgs, m)
	}
	return mailbox.FetchResult{Messages: msgs, Cursor: mailbox.FetchCursor{Token: prof.HistoryID}}, nil
}

func (c *Client) history(ctx context.Context, start string) (mailbox.FetchResult, error) {
	path := "/gmail/v1/users/me/history?startHistoryId=" + url.QueryEscape(start) + "&historyTypes=messageAdded&maxResults=50"
	var body struct {
		History []struct {
			MessagesAdded []struct {
				Message struct {
					ID string `json:"id"`
				} `json:"message"`
			} `json:"messagesAdded"`
		} `json:"history"`
		HistoryID string `json:"historyId"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &body); err != nil {
		return mailbox.FetchResult{}, err
	}
	seen := map[string]bool{}
	var msgs []mailbox.Message
	for _, h := range body.History {
		for _, added := range h.MessagesAdded {
			id := added.Message.ID
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			m, err := c.getMessage(ctx, id)
			if err != nil {
				if isNotFound(err) {
					continue
				}
				return mailbox.FetchResult{}, err
			}
			msgs = append(msgs, m)
		}
	}
	token := body.HistoryID
	if token == "" {
		token = start
	}
	return mailbox.FetchResult{Messages: msgs, Cursor: mailbox.FetchCursor{Token: token}}, nil
}

type profile struct {
	Email     string `json:"emailAddress"`
	HistoryID string `json:"historyId"`
}

func (c *Client) profile(ctx context.Context) (profile, error) {
	var p profile
	err := c.do(ctx, http.MethodGet, "/gmail/v1/users/me/profile", nil, &p)
	if p.HistoryID == "" {
		p.HistoryID = "0"
	}
	return p, err
}

func (c *Client) getMessage(ctx context.Context, id string) (mailbox.Message, error) {
	q := url.Values{}
	q.Set("format", "full")
	var raw struct {
		ID           string    `json:"id"`
		InternalDate string    `json:"internalDate"`
		Snippet      string    `json:"snippet"`
		Payload      gmailPart `json:"payload"`
	}
	if err := c.do(ctx, http.MethodGet, "/gmail/v1/users/me/messages/"+url.PathEscape(id)+"?"+q.Encode(), nil, &raw); err != nil {
		return mailbox.Message{}, err
	}
	headers := map[string]string{}
	for _, h := range raw.Payload.Headers {
		headers[http.CanonicalHeaderKey(h.Name)] = h.Value
	}
	from := emailAddr(headers["From"])
	to := splitAddrs(headers["To"])
	at := time.Time{}
	if ms, err := strconv.ParseInt(raw.InternalDate, 10, 64); err == nil && ms > 0 {
		at = time.UnixMilli(ms).UTC()
	}
	plain, htmlBody := partBodies(raw.Payload)
	text := plain
	if text == "" {
		text = html.UnescapeString(raw.Snippet)
	}
	return mailbox.Message{
		RFCMessageID: normalizeID(headers["Message-Id"]),
		InReplyTo:    normalizeID(headers["In-Reply-To"]),
		Subject:      html.UnescapeString(headers["Subject"]),
		From:         from,
		To:           to,
		Text:         text,
		HTML:         htmlBody,
		At:           at,
		ProviderID:   raw.ID,
	}, nil
}

type gmailPart struct {
	MimeType string `json:"mimeType"`
	Headers  []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"headers"`
	Body struct {
		Data string `json:"data"`
	} `json:"body"`
	Parts []gmailPart `json:"parts"`
}

func partBodies(p gmailPart) (plain, htmlBody string) {
	switch p.MimeType {
	case "text/plain":
		plain = decodeB64(p.Body.Data)
	case "text/html":
		htmlBody = decodeB64(p.Body.Data)
	}
	for _, child := range p.Parts {
		cp, ch := partBodies(child)
		if plain == "" {
			plain = cp
		}
		if htmlBody == "" {
			htmlBody = ch
		}
	}
	return plain, htmlBody
}

func decodeB64(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.StdEncoding} {
		b, err := enc.DecodeString(s)
		if err == nil {
			return string(b)
		}
	}
	return ""
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	base := c.Base
	if base == "" {
		base = defaultAPIBase
	}
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("gmail: %w", errNotFound)
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("gmail: %s %s", res.Status, strings.TrimSpace(string(b)))
	}
	if out == nil || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

var errNotFound = fmt.Errorf("not found")

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func emailAddr(raw string) string {
	a, err := mail.ParseAddress(raw)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(raw))
	}
	return strings.ToLower(a.Address)
}

func splitAddrs(raw string) []string {
	list, err := mail.ParseAddressList(raw)
	if err != nil || len(list) == 0 {
		if s := emailAddr(raw); s != "" {
			return []string{s}
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, a := range list {
		out = append(out, strings.ToLower(a.Address))
	}
	return out
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
