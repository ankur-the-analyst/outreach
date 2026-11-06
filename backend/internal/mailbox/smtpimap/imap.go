package smtpimap

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"outreach/internal/mime"
)

func (c *Client) AppendSent(ctx context.Context, m *mime.Built) error {
	if c.cfg.IMAPHost == "" {
		return nil
	}
	if m == nil || len(m.Bytes) == 0 {
		return fmt.Errorf("smtpimap: empty message")
	}
	cl, err := c.imapLogin(ctx)
	if err != nil {
		return err
	}
	defer cl.Logout()
	return cl.Append("Sent", nil, time.Time{}, bytes.NewReader(m.Bytes))
}

func (c *Client) FindSent(ctx context.Context, messageID string) (bool, error) {
	if c.cfg.IMAPHost == "" {
		return false, nil
	}
	if messageID == "" {
		return false, fmt.Errorf("smtpimap: message id required")
	}
	cl, err := c.imapLogin(ctx)
	if err != nil {
		return false, err
	}
	defer cl.Logout()
	if _, err := cl.Select("INBOX", true); err != nil {
		if _, err2 := cl.Select("Sent", true); err2 != nil {
			return false, fmt.Errorf("smtpimap: select: %w", err)
		}
	}
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Message-ID", messageID)
	ids, err := cl.Search(criteria)
	if err != nil {
		return false, fmt.Errorf("smtpimap: search: %w", err)
	}
	return len(ids) > 0, nil
}

func (c *Client) testIMAP(ctx context.Context) error {
	cl, err := c.imapLogin(ctx)
	if err != nil {
		return err
	}
	return cl.Logout()
}

func (c *Client) imapLogin(ctx context.Context) (*imapclient.Client, error) {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", c.imapAddr())
	if err != nil {
		return nil, fmt.Errorf("smtpimap: dial imap: %w", err)
	}
	cl, err := imapclient.New(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("smtpimap: imap: %w", err)
	}
	if c.cfg.Username != "" {
		if err := cl.Login(c.cfg.Username, c.cfg.Password); err != nil {
			_ = cl.Logout()
			return nil, fmt.Errorf("smtpimap: imap login: %w", err)
		}
	}
	return cl, nil
}

func (c *Client) imapAddr() string {
	port := c.cfg.IMAPPort
	if port == 0 {
		port = 143
	}
	return net.JoinHostPort(c.cfg.IMAPHost, strconv.Itoa(port))
}
