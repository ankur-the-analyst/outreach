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
	"outreach/internal/storage"

	"github.com/google/uuid"
)

const defaultAPIBase = "https://gmail.googleapis.com"

// pageSize is one Gmail list request. maxMessages is how many messages one sync
// will open. syncWindow is the search used when a mailbox has no cursor yet.
// A picture larger than maxImage is skipped.
const (
	pageSize    = 50
	maxMessages = 100
	maxImage    = 1 << 20
	syncWindow  = "newer_than:2d"
)

type Config struct {
	ClientID     string
	ClientSecret string
	HTTP         *http.Client
	APIBase      string
	Files        storage.Store
	FileBase     string
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
		return &Client{
			HTTP:     httpClient,
			Base:     strings.TrimRight(base, "/"),
			Files:    cfg.Files,
			FileBase: strings.TrimRight(cfg.FileBase, "/"),
		}, nil
	}
}

type Client struct {
	HTTP     *http.Client
	Base     string
	Files    storage.Store
	FileBase string
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
	if !cur.After.IsZero() || !cur.Before.IsZero() {
		return c.fullSyncQuery(ctx, dateQuery(cur))
	}
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
	return c.fullSyncQuery(ctx, syncWindow)
}

func (c *Client) fullSyncQuery(ctx context.Context, query string) (mailbox.FetchResult, error) {
	prof, err := c.profile(ctx)
	if err != nil {
		return mailbox.FetchResult{}, err
	}
	ids, err := c.listMessageIDs(ctx, query, maxMessages)
	if err != nil {
		return mailbox.FetchResult{}, err
	}
	msgs := make([]mailbox.Message, 0, len(ids))
	for _, id := range ids {
		m, err := c.getMessage(ctx, id)
		if err != nil {
			return mailbox.FetchResult{}, err
		}
		msgs = append(msgs, m)
	}
	return mailbox.FetchResult{Messages: msgs, Cursor: mailbox.FetchCursor{Token: prof.HistoryID}}, nil
}

func dateQuery(cur mailbox.FetchCursor) string {
	var parts []string
	if !cur.After.IsZero() {
		parts = append(parts, "after:"+cur.After.UTC().Format("2006/01/02"))
	}
	if !cur.Before.IsZero() {
		parts = append(parts, "before:"+cur.Before.UTC().AddDate(0, 0, 1).Format("2006/01/02"))
	}
	if len(parts) == 0 {
		return syncWindow
	}
	return strings.Join(parts, " ")
}

func (c *Client) listMessageIDs(ctx context.Context, query string, limit int) ([]string, error) {
	if limit <= 0 || limit > maxMessages {
		limit = maxMessages
	}
	var ids []string
	page := ""
	for len(ids) < limit {
		n := pageSize
		if remain := limit - len(ids); remain < n {
			n = remain
		}
		q := url.Values{}
		q.Set("q", query)
		q.Set("maxResults", strconv.Itoa(n))
		if page != "" {
			q.Set("pageToken", page)
		}
		var list struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := c.do(ctx, http.MethodGet, "/gmail/v1/users/me/messages?"+q.Encode(), nil, &list); err != nil {
			return nil, err
		}
		for _, m := range list.Messages {
			if m.ID == "" {
				continue
			}
			ids = append(ids, m.ID)
			if len(ids) >= limit {
				break
			}
		}
		if list.NextPageToken == "" || len(ids) >= limit {
			break
		}
		page = list.NextPageToken
	}
	return ids, nil
}

func (c *Client) history(ctx context.Context, start string) (mailbox.FetchResult, error) {
	seen := map[string]bool{}
	var msgs []mailbox.Message
	page := ""
	cursor := start
	for {
		q := url.Values{}
		q.Set("startHistoryId", start)
		q.Set("historyTypes", "messageAdded")
		q.Set("maxResults", strconv.Itoa(pageSize))
		if page != "" {
			q.Set("pageToken", page)
		}
		var body struct {
			History []struct {
				ID            string `json:"id"`
				MessagesAdded []struct {
					Message struct {
						ID string `json:"id"`
					} `json:"message"`
				} `json:"messagesAdded"`
			} `json:"history"`
			HistoryID     string `json:"historyId"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := c.do(ctx, http.MethodGet, "/gmail/v1/users/me/history?"+q.Encode(), nil, &body); err != nil {
			return mailbox.FetchResult{}, err
		}
		stop := false
		for _, h := range body.History {
			for _, added := range h.MessagesAdded {
				if len(msgs) >= maxMessages {
					stop = true
					break
				}
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
			if stop {
				break
			}
			if h.ID != "" {
				cursor = h.ID
			}
		}
		if stop {
			break
		}
		if body.NextPageToken == "" {
			if body.HistoryID != "" {
				cursor = body.HistoryID
			}
			break
		}
		page = body.NextPageToken
	}
	if cursor == "" {
		cursor = start
	}
	return mailbox.FetchResult{Messages: msgs, Cursor: mailbox.FetchCursor{Token: cursor}}, nil
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
	plain, htmlBody := c.partBodies(ctx, raw.ID, raw.Payload)
	htmlBody = c.embedImages(ctx, raw.ID, raw.Payload, htmlBody)
	text := html.UnescapeString(raw.Snippet)
	if strings.TrimSpace(text) == "" {
		text = plain
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
		Data         string `json:"data"`
		AttachmentID string `json:"attachmentId"`
	} `json:"body"`
	Parts []gmailPart `json:"parts"`
}

func (c *Client) partBodies(ctx context.Context, messageID string, p gmailPart) (plain, htmlBody string) {
	body := c.partText(ctx, messageID, p)
	switch {
	case strings.HasPrefix(p.MimeType, "text/plain"):
		plain = body
	case strings.HasPrefix(p.MimeType, "text/html"):
		htmlBody = body
	}
	for _, child := range p.Parts {
		cp, ch := c.partBodies(ctx, messageID, child)
		if plain == "" {
			plain = cp
		}
		if htmlBody == "" {
			htmlBody = ch
		}
	}
	return plain, htmlBody
}

func (c *Client) partText(ctx context.Context, messageID string, p gmailPart) string {
	if !strings.HasPrefix(p.MimeType, "text/plain") && !strings.HasPrefix(p.MimeType, "text/html") {
		return ""
	}
	return string(c.partBytes(ctx, messageID, p))
}

func (c *Client) embedImages(ctx context.Context, messageID string, p gmailPart, htmlBody string) string {
	if htmlBody == "" || !strings.Contains(strings.ToLower(htmlBody), "cid:") {
		return htmlBody
	}
	images := map[string]string{}
	c.collectImages(ctx, messageID, p, images)
	for id, uri := range images {
		htmlBody = strings.ReplaceAll(htmlBody, "cid:"+id, uri)
		htmlBody = strings.ReplaceAll(htmlBody, "CID:"+id, uri)
		htmlBody = strings.ReplaceAll(htmlBody, "cid:<"+id+">", uri)
	}
	return htmlBody
}

func (c *Client) collectImages(ctx context.Context, messageID string, p gmailPart, images map[string]string) {
	if strings.HasPrefix(p.MimeType, "image/") {
		if id := contentID(p); id != "" {
			raw := c.partBytes(ctx, messageID, p)
			if len(raw) > 0 && len(raw) <= maxImage {
				images[id] = c.imageURL(ctx, p.MimeType, raw)
			}
		}
	}
	for _, child := range p.Parts {
		c.collectImages(ctx, messageID, child, images)
	}
}

func (c *Client) imageURL(ctx context.Context, mimeType string, raw []byte) string {
	if c.Files != nil && c.FileBase != "" {
		key := "mail-images/" + uuid.NewString() + imageExt(mimeType)
		if err := c.Files.Put(ctx, key, mimeType, raw); err == nil {
			return c.FileBase + "/api/v1/mail-files/" + key
		}
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw)
}

func imageExt(mimeType string) string {
	switch {
	case strings.Contains(mimeType, "jpeg"), strings.Contains(mimeType, "jpg"):
		return ".jpg"
	case strings.Contains(mimeType, "gif"):
		return ".gif"
	case strings.Contains(mimeType, "webp"):
		return ".webp"
	default:
		return ".png"
	}
}

func contentID(p gmailPart) string {
	for _, h := range p.Headers {
		if !strings.EqualFold(h.Name, "Content-Id") {
			continue
		}
		id := strings.Trim(strings.TrimSpace(h.Value), "<>")
		return id
	}
	return ""
}

func (c *Client) partBytes(ctx context.Context, messageID string, p gmailPart) []byte {
	if p.Body.Data != "" {
		return decodeB64(p.Body.Data)
	}
	if p.Body.AttachmentID == "" || messageID == "" {
		return nil
	}
	var out struct {
		Data string `json:"data"`
	}
	path := "/gmail/v1/users/me/messages/" + url.PathEscape(messageID) + "/attachments/" + url.PathEscape(p.Body.AttachmentID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil
	}
	return decodeB64(out.Data)
}

func decodeB64(s string) []byte {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.StdEncoding} {
		b, err := enc.DecodeString(s)
		if err == nil {
			return b
		}
	}
	return nil
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
