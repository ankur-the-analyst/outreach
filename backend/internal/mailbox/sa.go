package mailbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"outreach/internal/db"
)

var ErrSADomainTaken = errors.New("service account already exists for domain")

type ServiceAccount struct {
	ID          uuid.UUID `json:"id"`
	Domain      string    `json:"domain"`
	ClientEmail string    `json:"client_email"`
	ClientID    string    `json:"client_id"`
}

type SAInput struct {
	Domain string
	JSON   []byte
}

var defaultSAScopes = []string{
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.readonly",
}

func (p *Postgres) CreateSA(ctx context.Context, workspaceID uuid.UUID, in SAInput) (ServiceAccount, error) {
	domain := strings.ToLower(strings.TrimSpace(in.Domain))
	if domain == "" {
		return ServiceAccount{}, fmt.Errorf("mailbox: domain required")
	}
	if IsGmail("user@" + domain) {
		return ServiceAccount{}, ErrGmailServiceAccount
	}
	var parsed struct {
		ClientEmail string `json:"client_email"`
		ClientID    string `json:"client_id"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal(in.JSON, &parsed); err != nil || parsed.ClientEmail == "" || parsed.PrivateKey == "" {
		return ServiceAccount{}, fmt.Errorf("mailbox: invalid service account json")
	}
	var out ServiceAccount
	err := p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		secretID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		sealed, err := p.keys.Seal(in.JSON)
		if err != nil {
			return fmt.Errorf("mailbox: seal sa: %w", err)
		}
		if err := q.InsertSecret(ctx, db.InsertSecretParams{
			WorkspaceID: workspaceID,
			ID:          secretID,
			Kind:        "service_account_key",
			KeyVersion:  int32(sealed.KeyVersion),
			WrappedDek:  sealed.WrappedDEK,
			Nonce:       sealed.Nonce,
			Ciphertext:  sealed.Ciphertext,
			Hint:        text(parsed.ClientEmail),
		}); err != nil {
			return err
		}
		row, err := q.InsertGoogleServiceAccount(ctx, db.InsertGoogleServiceAccountParams{
			WorkspaceID: workspaceID,
			ID:          id,
			Domain:      domain,
			ClientEmail: parsed.ClientEmail,
			ClientID:    parsed.ClientID,
			SecretID:    secretID,
			Scopes:      defaultSAScopes,
		})
		if err != nil {
			return err
		}
		out = ServiceAccount{ID: row.ID, Domain: row.Domain, ClientEmail: row.ClientEmail, ClientID: row.ClientID}
		return nil
	})
	if isUnique(err) {
		return ServiceAccount{}, ErrSADomainTaken
	}
	return out, err
}

func (p *Postgres) ListSA(ctx context.Context, workspaceID uuid.UUID) ([]ServiceAccount, error) {
	var out []ServiceAccount
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListGoogleServiceAccounts(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]ServiceAccount, 0, len(rows))
		for _, row := range rows {
			out = append(out, ServiceAccount{ID: row.ID, Domain: row.Domain, ClientEmail: row.ClientEmail, ClientID: row.ClientID})
		}
		return nil
	})
	return out, err
}

func lookupSA(ctx context.Context, q *db.Queries, workspaceID uuid.UUID, domain string) (uuid.UUID, error) {
	row, err := q.GetGoogleServiceAccountByDomain(ctx, db.GetGoogleServiceAccountByDomainParams{
		WorkspaceID: workspaceID,
		Domain:      domain,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}
