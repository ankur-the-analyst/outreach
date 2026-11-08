package mime

import (
	"bytes"
	"fmt"
	stdmime "mime/multipart"
	netmail "net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	From            string
	To              []string
	Subject         string
	Text            string
	HTML            string
	MessageID       string
	ListUnsubscribe string
}

func NewMessageID(host string) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "outreach.local"
	}
	return "<" + id.String() + "@" + host + ">", nil
}

func Build(m Message) (*Built, error) {
	from, err := netmail.ParseAddress(m.From)
	if err != nil {
		return nil, fmt.Errorf("mime: from: %w", err)
	}
	if len(m.To) == 0 {
		return nil, fmt.Errorf("mime: to required")
	}
	toAddrs := make([]string, 0, len(m.To))
	var toHeader []string
	for _, raw := range m.To {
		a, err := netmail.ParseAddress(raw)
		if err != nil {
			return nil, fmt.Errorf("mime: to: %w", err)
		}
		toAddrs = append(toAddrs, a.Address)
		toHeader = append(toHeader, a.String())
	}
	msgid := strings.TrimSpace(m.MessageID)
	if msgid == "" {
		host := from.Address
		if i := strings.LastIndex(host, "@"); i >= 0 {
			host = host[i+1:]
		}
		msgid, err = NewMessageID(host)
		if err != nil {
			return nil, err
		}
	}

	var body bytes.Buffer
	mp := stdmime.NewWriter(&body)
	plain, err := mp.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := plain.Write([]byte(m.Text)); err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.HTML) != "" {
		html, err := mp.CreatePart(textproto.MIMEHeader{
			"Content-Type": {"text/html; charset=utf-8"},
		})
		if err != nil {
			return nil, err
		}
		if _, err := html.Write([]byte(m.HTML)); err != nil {
			return nil, err
		}
	}
	if err := mp.Close(); err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from.String())
	fmt.Fprintf(&msg, "To: %s\r\n", strings.Join(toHeader, ", "))
	fmt.Fprintf(&msg, "Subject: %s\r\n", m.Subject)
	fmt.Fprintf(&msg, "Message-ID: %s\r\n", msgid)
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	if u := strings.TrimSpace(m.ListUnsubscribe); u != "" {
		fmt.Fprintf(&msg, "List-Unsubscribe: <%s>\r\n", u)
		fmt.Fprintf(&msg, "List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n")
	}
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", mp.Boundary())
	msg.Write(body.Bytes())

	return &Built{MessageID: msgid, From: from.Address, To: toAddrs, Bytes: msg.Bytes()}, nil
}
