package smtpimap

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strconv"

	"outreach/internal/mailbox"
	"outreach/internal/mime"
)

type Config struct {
	SMTPHost string
	SMTPPort int
	IMAPHost string
	IMAPPort int
	Username string
	Password string
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

var _ mailbox.Transport = (*Client)(nil)

func (c *Client) Test(ctx context.Context) error {
	if err := c.testSMTP(ctx); err != nil {
		return err
	}
	if c.cfg.IMAPHost == "" {
		return nil
	}
	return c.testIMAP(ctx)
}

func (c *Client) Send(ctx context.Context, m *mime.Built) (mailbox.SendResult, error) {
	if m == nil || m.From == "" || len(m.To) == 0 || len(m.Bytes) == 0 {
		return mailbox.SendResult{}, fmt.Errorf("smtpimap: message from, to, and body required")
	}
	client, err := c.smtpClient(ctx)
	if err != nil {
		return mailbox.SendResult{}, err
	}
	defer client.Close()
	if err := c.maybeAuth(client); err != nil {
		return mailbox.SendResult{}, err
	}
	if err := client.Mail(m.From); err != nil {
		return mailbox.SendResult{}, fmt.Errorf("smtpimap: MAIL: %w", err)
	}
	for _, to := range m.To {
		if err := client.Rcpt(to); err != nil {
			return mailbox.SendResult{}, fmt.Errorf("smtpimap: RCPT %s: %w", to, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return mailbox.SendResult{}, fmt.Errorf("smtpimap: DATA: %w", err)
	}
	if _, err := w.Write(m.Bytes); err != nil {
		_ = w.Close()
		return mailbox.SendResult{}, fmt.Errorf("smtpimap: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return mailbox.SendResult{}, fmt.Errorf("smtpimap: data close: %w", err)
	}
	if err := client.Quit(); err != nil {
		return mailbox.SendResult{}, fmt.Errorf("smtpimap: quit: %w", err)
	}
	return mailbox.SendResult{}, nil
}

func (c *Client) testSMTP(ctx context.Context) error {
	client, err := c.smtpClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := c.maybeAuth(client); err != nil {
		return err
	}
	return client.Quit()
}

func (c *Client) smtpClient(ctx context.Context) (*smtp.Client, error) {
	if c.cfg.SMTPHost == "" {
		return nil, fmt.Errorf("smtpimap: smtp host required")
	}
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", c.smtpAddr())
	if err != nil {
		return nil, fmt.Errorf("smtpimap: dial smtp: %w", err)
	}
	client, err := smtp.NewClient(conn, c.cfg.SMTPHost)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("smtpimap: smtp: %w", err)
	}
	if err := client.Hello("outreach.local"); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("smtpimap: ehlo: %w", err)
	}
	return client, nil
}

func (c *Client) maybeAuth(client *smtp.Client) error {
	if c.cfg.Username == "" {
		return nil
	}
	ok, _ := client.Extension("AUTH")
	if !ok {
		return fmt.Errorf("smtpimap: AUTH not advertised")
	}
	auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtpimap: auth: %w", err)
	}
	return nil
}

func (c *Client) smtpAddr() string {
	port := c.cfg.SMTPPort
	if port == 0 {
		port = 587
	}
	return net.JoinHostPort(c.cfg.SMTPHost, strconv.Itoa(port))
}
