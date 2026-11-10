package mime

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	stdmime "mime"
	"mime/multipart"
	netmail "net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MaxBytes = 25 << 20

var ErrTooLarge = errors.New("mime: message too large")

type File struct {
	Name        string
	ContentType string
	Bytes       []byte
}

type Message struct {
	From            string
	To              []string
	Subject         string
	Text            string
	HTML            string
	MessageID       string
	ListUnsubscribe string
	Files           []File
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
	raw := len(m.Text) + len(m.HTML)
	for _, f := range m.Files {
		raw += len(f.Bytes)
	}
	if raw > MaxBytes {
		return nil, ErrTooLarge
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

	altBody, altBoundary, err := alternative(m.Text, m.HTML)
	if err != nil {
		return nil, err
	}
	ctype := "multipart/alternative; boundary=" + altBoundary
	payload := altBody
	if len(m.Files) > 0 {
		payload, ctype, err = mixed(altBody, altBoundary, m.Files)
		if err != nil {
			return nil, err
		}
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
	fmt.Fprintf(&msg, "Content-Type: %s\r\n\r\n", ctype)
	msg.Write(payload)
	if msg.Len() > MaxBytes {
		return nil, ErrTooLarge
	}
	return &Built{MessageID: msgid, From: from.Address, To: toAddrs, Bytes: msg.Bytes()}, nil
}

func alternative(text, html string) ([]byte, string, error) {
	var body bytes.Buffer
	mp := multipart.NewWriter(&body)
	plain, err := mp.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err := plain.Write([]byte(text)); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(html) != "" {
		h, err := mp.CreatePart(textproto.MIMEHeader{
			"Content-Type": {"text/html; charset=utf-8"},
		})
		if err != nil {
			return nil, "", err
		}
		if _, err := h.Write([]byte(html)); err != nil {
			return nil, "", err
		}
	}
	if err := mp.Close(); err != nil {
		return nil, "", err
	}
	return body.Bytes(), mp.Boundary(), nil
}

func mixed(alt []byte, altBoundary string, files []File) ([]byte, string, error) {
	var body bytes.Buffer
	mp := multipart.NewWriter(&body)
	inner, err := mp.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"multipart/alternative; boundary=" + altBoundary},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err := inner.Write(alt); err != nil {
		return nil, "", err
	}
	for _, f := range files {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			name = "file"
		}
		ctype := strings.TrimSpace(f.ContentType)
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		disp := stdmime.FormatMediaType("attachment", map[string]string{"filename": name})
		part, err := mp.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {ctype},
			"Content-Disposition":       {disp},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return nil, "", err
		}
		enc := base64.NewEncoder(base64.StdEncoding, part)
		if _, err := enc.Write(f.Bytes); err != nil {
			return nil, "", err
		}
		if err := enc.Close(); err != nil {
			return nil, "", err
		}
	}
	if err := mp.Close(); err != nil {
		return nil, "", err
	}
	return body.Bytes(), "multipart/mixed; boundary=" + mp.Boundary(), nil
}
