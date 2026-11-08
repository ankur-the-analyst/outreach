package mailbox

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/domain"
	"outreach/internal/pacing"
)

type Sealer interface {
	Seal(plain []byte) (crypto.Sealed, error)
}

type Mailbox struct {
	ID                  uuid.UUID `json:"id"`
	Email               string    `json:"email"`
	DisplayName         string    `json:"display_name"`
	AuthKind            string    `json:"auth_kind"`
	Provider            string    `json:"provider"`
	Status              string    `json:"status"`
	Timezone            string    `json:"timezone"`
	SMTPHost            string    `json:"smtp_host"`
	SMTPPort            int32     `json:"smtp_port"`
	SendingDomainID     uuid.UUID `json:"sending_domain_id"`
	WarmupProfileID     uuid.UUID `json:"warmup_profile_id"`
	EffectiveDailyLimit int32     `json:"effective_daily_limit"`
	ReplyMode           string    `json:"reply_mode"`
	WarmupKind          string    `json:"warmup_kind"`
	Domain              string    `json:"domain,omitempty"`
	DomainDailyLimit    int32     `json:"domain_daily_limit,omitempty"`
}

type SendingDomain struct {
	ID         uuid.UUID `json:"id"`
	Domain     string    `json:"domain"`
	DailyLimit int32     `json:"daily_limit"`
}

type ConnectInput struct {
	Email       string
	DisplayName string
	AuthKind    string
	Password    string
	Timezone    string
	SMTPHost    string
	SMTPPort    int
	SMTPTLS     string
	IMAPHost    string
	IMAPPort    int
	WarmupKind  string
}

type Postgres struct {
	pool *db.Pool
	keys Sealer
}

func NewPostgres(pool *db.Pool, keys Sealer) *Postgres {
	return &Postgres{pool: pool, keys: keys}
}

func (p *Postgres) Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error) {
	var role string
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID, UserID: userID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMembership(ctx, db.GetMembershipParams{WorkspaceID: workspaceID, UserID: userID})
		if err != nil {
			return err
		}
		role = row.Role
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID) ([]Mailbox, error) {
	var out []Mailbox
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListMailboxes(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]Mailbox, 0, len(rows))
		for _, row := range rows {
			out = append(out, mailboxFromList(row))
		}
		return nil
	})
	return out, err
}

func (p *Postgres) ListDomains(ctx context.Context, workspaceID uuid.UUID) ([]SendingDomain, error) {
	var out []SendingDomain
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListSendingDomains(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]SendingDomain, 0, len(rows))
		for _, row := range rows {
			out = append(out, SendingDomain{ID: row.ID, Domain: row.Domain, DailyLimit: row.DailyLimit})
		}
		return nil
	})
	return out, err
}

func (p *Postgres) Connect(ctx context.Context, workspaceID uuid.UUID, in ConnectInput) (Mailbox, error) {
	if err := RejectGmailServiceAccount(in.Email, in.AuthKind); err != nil {
		return Mailbox{}, err
	}
	if in.AuthKind == "oauth" {
		return Mailbox{}, ErrNotConfigured
	}
	var out Mailbox
	err := p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		n, err := q.CountWarmupProfiles(ctx)
		if err != nil {
			return err
		}
		if n == 0 {
			if err := pacing.SeedPresets(ctx, q, workspaceID); err != nil {
				return err
			}
		}
		kind := in.WarmupKind
		if kind == "" {
			kind = pacing.KindStandard
		}
		profile, err := q.GetWarmupProfileByKind(ctx, db.GetWarmupProfileByKindParams{WorkspaceID: workspaceID, Kind: kind})
		if err != nil {
			return fmt.Errorf("mailbox: warmup: %w", err)
		}
		spec, err := pacing.ParseSpec(profile.Spec)
		if err != nil {
			return err
		}
		domName := EmailDomain(in.Email)
		if domName == "" {
			return fmt.Errorf("mailbox: email domain required")
		}
		dom, err := ensureDomain(ctx, q, workspaceID, domName)
		if err != nil {
			return err
		}
		var secretID *uuid.UUID
		var saID *uuid.UUID
		if in.AuthKind == "service_account" {
			id, err := lookupSA(ctx, q, workspaceID, domName)
			if err != nil {
				return err
			}
			saID = &id
		}
		if in.Password != "" {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			sealed, err := p.keys.Seal([]byte(in.Password))
			if err != nil {
				return fmt.Errorf("mailbox: seal: %w", err)
			}
			if err := q.InsertSecret(ctx, db.InsertSecretParams{
				WorkspaceID: workspaceID,
				ID:          id,
				Kind:        "app_password",
				KeyVersion:  int32(sealed.KeyVersion),
				WrappedDek:  sealed.WrappedDEK,
				Nonce:       sealed.Nonce,
				Ciphertext:  sealed.Ciphertext,
				Hint:        text(in.AuthKind),
			}); err != nil {
				return fmt.Errorf("mailbox: secret: %w", err)
			}
			secretID = &id
		}
		mbID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		provider := ProviderFor(in.Email)
		if in.AuthKind == "service_account" {
			provider = "workspace"
		}
		host, port, tlsMode, imapHost, imapPort := smtpDefaults(provider, in)
		daily := int32(40)
		hourly := int32(15)
		ceiling := int32(2000)
		if provider == "gmail" {
			hourly = 10
			ceiling = 500
		}
		eff := pacing.Ramp(spec, 1, pacing.Health{Status: pacing.StatusWarming})
		if eff > int(daily) || spec.Uncapped {
			eff = int(daily)
		}
		if eff <= 0 {
			eff = 2
		}
		row, err := q.InsertMailbox(ctx, db.InsertMailboxParams{
			WorkspaceID:         workspaceID,
			ID:                  mbID,
			Email:               strings.ToLower(strings.TrimSpace(in.Email)),
			DisplayName:         text(in.DisplayName),
			AuthKind:            in.AuthKind,
			Provider:            provider,
			SmtpHost:            text(host),
			SmtpPort:            int4(port),
			SmtpTls:             text(tlsMode),
			ImapHost:            text(imapHost),
			ImapPort:            int4(imapPort),
			Username:            text(strings.ToLower(strings.TrimSpace(in.Email))),
			SecretID:            secretID,
			ServiceAccountID:    saID,
			SendingDomainID:     dom.ID,
			Timezone:            in.Timezone,
			WindowDays:          []int32{1, 2, 3, 4, 5},
			WindowStart:         clock(9, 0),
			WindowEnd:           clock(17, 0),
			DailyLimit:          daily,
			HourlyLimit:         hourly,
			MinGapS:             240,
			MaxGapS:             540,
			HardDailyCeiling:    ceiling,
			WarmupProfileID:     profile.ID,
			EffectiveDailyLimit: int32(eff),
			Status:              pacing.StatusWarming,
		})
		if err != nil {
			return err
		}
		out = mailboxFromInsert(row)
		return nil
	})
	if isUnique(err) {
		return Mailbox{}, ErrEmailTaken
	}
	return out, err
}

func (p *Postgres) Pause(ctx context.Context, workspaceID, id uuid.UUID) error {
	return p.setStatus(ctx, workspaceID, id, pacing.StatusPaused, "paused")
}

func (p *Postgres) Reconnect(ctx context.Context, workspaceID, id uuid.UUID) error {
	return p.setStatus(ctx, workspaceID, id, pacing.StatusWarming, "")
}

func (p *Postgres) AssignWarmup(ctx context.Context, workspaceID, id uuid.UUID, kind string) error {
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: id}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		profile, err := q.GetWarmupProfileByKind(ctx, db.GetWarmupProfileByKindParams{WorkspaceID: workspaceID, Kind: kind})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		spec, err := pacing.ParseSpec(profile.Spec)
		if err != nil {
			return err
		}
		eff := pacing.Ramp(spec, 1, pacing.Health{Status: pacing.StatusWarming})
		if spec.Uncapped {
			eff = 40
		}
		if eff <= 0 {
			eff = 2
		}
		return q.SetMailboxWarmup(ctx, db.SetMailboxWarmupParams{
			WorkspaceID:         workspaceID,
			ID:                  id,
			WarmupProfileID:     profile.ID,
			EffectiveDailyLimit: int32(eff),
		})
	})
}

func (p *Postgres) SetReplyMode(ctx context.Context, workspaceID, id uuid.UUID, mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "draft" && mode != "off" {
		return ErrReplyMode
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: id}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		return q.SetMailboxReplyMode(ctx, db.SetMailboxReplyModeParams{
			WorkspaceID: workspaceID, ID: id, ReplyMode: mode,
		})
	})
}

func (p *Postgres) setStatus(ctx context.Context, workspaceID, id uuid.UUID, status, reason string) error {
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: id}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		return q.SetMailboxStatus(ctx, db.SetMailboxStatusParams{
			WorkspaceID:  workspaceID,
			ID:           id,
			Status:       status,
			StatusReason: text(reason),
		})
	})
}

func ensureDomain(ctx context.Context, q *db.Queries, workspaceID uuid.UUID, name string) (db.SendingDomain, error) {
	dom, err := q.GetSendingDomainByName(ctx, db.GetSendingDomainByNameParams{WorkspaceID: workspaceID, Domain: name})
	if err == nil {
		return dom, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.SendingDomain{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return db.SendingDomain{}, err
	}
	if err := q.InsertSendingDomain(ctx, db.InsertSendingDomainParams{
		WorkspaceID: workspaceID,
		ID:          id,
		Domain:      name,
		DailyLimit:  int32(domain.DefaultDailyLimit(0)),
	}); err != nil {
		return db.SendingDomain{}, fmt.Errorf("mailbox: domain: %w", err)
	}
	return q.GetSendingDomainByName(ctx, db.GetSendingDomainByNameParams{WorkspaceID: workspaceID, Domain: name})
}

func smtpDefaults(provider string, in ConnectInput) (host string, port int, tlsMode, imapHost string, imapPort int) {
	host, port, tlsMode = in.SMTPHost, in.SMTPPort, in.SMTPTLS
	imapHost, imapPort = in.IMAPHost, in.IMAPPort
	if provider == "gmail" && in.AuthKind == "app_password" {
		if host == "" {
			host, port, tlsMode = "smtp.gmail.com", 587, "starttls"
		}
		if imapHost == "" {
			imapHost, imapPort = "imap.gmail.com", 993
		}
	}
	return host, port, tlsMode, imapHost, imapPort
}

func mailboxFromInsert(row db.InsertMailboxRow) Mailbox {
	return Mailbox{
		ID: row.ID, Email: row.Email, DisplayName: row.DisplayName.String,
		AuthKind: row.AuthKind, Provider: row.Provider, Status: row.Status,
		Timezone: row.Timezone, SMTPHost: row.SmtpHost.String, SMTPPort: row.SmtpPort.Int32,
		SendingDomainID: row.SendingDomainID,
		WarmupProfileID: row.WarmupProfileID, EffectiveDailyLimit: row.EffectiveDailyLimit,
	}
}

func mailboxFromList(row db.ListMailboxesRow) Mailbox {
	return Mailbox{
		ID: row.ID, Email: row.Email, DisplayName: row.DisplayName.String,
		AuthKind: row.AuthKind, Provider: row.Provider, Status: row.Status,
		Timezone: row.Timezone, SMTPHost: row.SmtpHost.String, SMTPPort: row.SmtpPort.Int32,
		SendingDomainID: row.SendingDomainID,
		WarmupProfileID: row.WarmupProfileID, EffectiveDailyLimit: row.EffectiveDailyLimit,
		ReplyMode: row.ReplyMode, WarmupKind: row.WarmupKind,
		Domain: row.Domain, DomainDailyLimit: row.DomainDailyLimit,
	}
}

func (p *Postgres) SMTPEndpoint(ctx context.Context, workspaceID, id uuid.UUID) (string, int, error) {
	var host string
	var port int
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		host = row.SmtpHost.String
		if row.SmtpPort.Valid {
			port = int(row.SmtpPort.Int32)
		}
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	if host == "" {
		return "", 0, ErrNoSMTP
	}
	return host, port, nil
}

func (p *Postgres) ListPools(ctx context.Context, workspaceID uuid.UUID) ([]Pool, error) {
	var out []Pool
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListMailboxPools(ctx, workspaceID)
		if err != nil {
			return err
		}
		members, err := q.ListMailboxPoolMembers(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = assemblePools(rows, members)
		return nil
	})
	return out, err
}

func (p *Postgres) CreatePool(ctx context.Context, workspaceID uuid.UUID, name string, mailboxIDs []uuid.UUID) (Pool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Pool{}, fmt.Errorf("mailbox: name required")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Pool{}, err
	}
	var out Pool
	err = p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.InsertMailboxPool(ctx, db.InsertMailboxPoolParams{
			WorkspaceID: workspaceID, ID: id, Name: name,
		})
		if err != nil {
			return err
		}
		out = Pool{ID: row.ID, Name: row.Name, Mailboxes: []PoolMailbox{}}
		for _, mailboxID := range mailboxIDs {
			if mailboxID == uuid.Nil {
				continue
			}
			if err := q.InsertMailboxPoolMember(ctx, db.InsertMailboxPoolMemberParams{
				WorkspaceID: workspaceID, PoolID: id, MailboxID: mailboxID,
			}); err != nil {
				if isFK(err) {
					return ErrNotFound
				}
				if isUnique(err) {
					continue
				}
				return err
			}
		}
		members, err := q.ListMailboxPoolMembers(ctx, workspaceID)
		if err != nil {
			return err
		}
		assembled := assemblePools([]db.MailboxPool{row}, members)
		if len(assembled) == 1 {
			out = assembled[0]
		}
		return nil
	})
	return out, err
}

func isFK(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func text(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func int4(n int) pgtype.Int4 {
	if n == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(n), Valid: true}
}

func clock(h, m int) pgtype.Time {
	return pgtype.Time{Microseconds: int64(h*3600+m*60) * 1_000_000, Valid: true}
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
