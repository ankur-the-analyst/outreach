package tracking

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/net/html"
)

var ErrHost = errors.New("Set a public tracking URL before sending this campaign")

type InjectInput struct {
	HTML        string
	Host        string
	HMAC        Hasher
	WorkspaceID uuid.UUID
	DraftID     uuid.UUID
	Opens       bool
	Clicks      bool
}

func Inject(in InjectInput) (string, error) {
	body := strings.TrimSpace(in.HTML)
	if body == "" || (!in.Opens && !in.Clicks) {
		return in.HTML, nil
	}
	host := strings.TrimRight(strings.TrimSpace(in.Host), "/")
	if host == "" {
		return "", ErrHost
	}
	if in.HMAC == nil || in.WorkspaceID == uuid.Nil || in.DraftID == uuid.Nil {
		return "", fmt.Errorf("tracking: token required")
	}
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("tracking: html: %w", err)
	}
	bodyNode := findBody(doc)
	if bodyNode == nil {
		return in.HTML, nil
	}
	if in.Clicks {
		if err := wrapHrefs(bodyNode, in, host); err != nil {
			return "", err
		}
	}
	var b bytes.Buffer
	for c := bodyNode.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&b, c); err != nil {
			return "", err
		}
	}
	out := b.String()
	if in.Opens && !strings.Contains(out, "/t/o/") {
		tok, err := Token(in.HMAC, TokenData{
			WorkspaceID: in.WorkspaceID, DraftID: in.DraftID, Kind: KindOpen,
		})
		if err != nil {
			return "", err
		}
		out += `<img src="` + host + `/t/o/` + tok + `" width="1" height="1" alt="">`
	}
	return out, nil
}

func wrapHrefs(n *html.Node, in InjectInput, host string) error {
	if n.Type == html.ElementNode && n.Data == "a" {
		for i, a := range n.Attr {
			if !strings.EqualFold(a.Key, "href") {
				continue
			}
			href := strings.TrimSpace(a.Val)
			if skipHref(href) {
				continue
			}
			tok, err := Token(in.HMAC, TokenData{
				WorkspaceID: in.WorkspaceID, DraftID: in.DraftID, Kind: KindClick, URL: href,
			})
			if err != nil {
				return err
			}
			n.Attr[i].Val = host + "/t/c/" + tok
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := wrapHrefs(c, in, host); err != nil {
			return err
		}
	}
	return nil
}

func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if got := findBody(c); got != nil {
			return got
		}
	}
	return nil
}

func skipHref(href string) bool {
	if href == "" || strings.HasPrefix(href, "#") {
		return true
	}
	lower := strings.ToLower(href)
	if strings.HasPrefix(lower, "mailto:") || strings.HasPrefix(lower, "javascript:") {
		return true
	}
	return strings.Contains(href, "/u/") || strings.Contains(href, "/t/o/") || strings.Contains(href, "/t/c/")
}
