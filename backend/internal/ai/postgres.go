package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/crypto"
	"outreach/internal/db"
)

type Keys interface {
	Seal(plain []byte) (crypto.Sealed, error)
	Open(s crypto.Sealed) ([]byte, error)
}

type Postgres struct {
	pool *db.Pool
	keys Keys
}

func NewPostgres(pool *db.Pool, keys Keys) *Postgres {
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

func (p *Postgres) Get(ctx context.Context, workspaceID uuid.UUID) (Settings, error) {
	var out Settings
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetWorkspaceAI(ctx, workspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		out = settingsFromJSON(row.Providers, row.Models)
		return nil
	})
	return out, err
}

func (p *Postgres) Put(ctx context.Context, workspaceID uuid.UUID, in PutInput) (Settings, error) {
	existing, err := p.Get(ctx, workspaceID)
	if err != nil {
		return Settings{}, err
	}
	if err := validatePut(in, existing); err != nil {
		return Settings{}, err
	}
	in.APIKey = strings.TrimSpace(in.APIKey)
	in.BaseURL = strings.TrimSpace(in.BaseURL)
	in.ComposeModel = strings.TrimSpace(in.ComposeModel)
	in.EmbedModel = strings.TrimSpace(in.EmbedModel)

	var current []storedProvider
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetWorkspaceAI(ctx, workspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(row.Providers) > 0 {
			_ = json.Unmarshal(row.Providers, &current)
		}
		return nil
	})
	if err != nil {
		return Settings{}, err
	}

	var out Settings
	err = p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		prov := storedProvider{Kind: in.Kind, BaseURL: in.BaseURL}
		if len(current) > 0 {
			prov.ID = current[0].ID
			prov.SecretID = current[0].SecretID
			prov.Hint = current[0].Hint
		}
		if prov.ID == uuid.Nil {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			prov.ID = id
		}
		if in.APIKey != "" {
			secretID, err := uuid.NewV7()
			if err != nil {
				return err
			}
			sealed, err := p.keys.Seal([]byte(in.APIKey))
			if err != nil {
				return fmt.Errorf("ai: seal key: %w", err)
			}
			hint := keyHint(in.APIKey)
			if err := q.InsertSecret(ctx, db.InsertSecretParams{
				WorkspaceID: workspaceID,
				ID:          secretID,
				Kind:        "llm_key",
				KeyVersion:  int32(sealed.KeyVersion),
				WrappedDek:  sealed.WrappedDEK,
				Nonce:       sealed.Nonce,
				Ciphertext:  sealed.Ciphertext,
				Hint:        pgtype.Text{String: hint, Valid: true},
			}); err != nil {
				return err
			}
			prov.SecretID = secretID
			prov.Hint = hint
		}
		providers, err := json.Marshal([]storedProvider{prov})
		if err != nil {
			return err
		}
		tag := func(model string) string {
			return string(in.Kind) + ":" + model
		}
		models, err := json.Marshal(map[string]string{
			"compose":  tag(in.ComposeModel),
			"analyze":  tag(in.ComposeModel),
			"classify": tag(in.ComposeModel),
			"reply":    tag(in.ComposeModel),
			"embed":    tag(in.EmbedModel),
		})
		if err != nil {
			return err
		}
		fb, err := json.Marshal(storedFallback{Kind: in.Kind, Model: in.ComposeModel})
		if err != nil {
			return err
		}
		row, err := q.UpsertWorkspaceAI(ctx, db.UpsertWorkspaceAIParams{
			WorkspaceID: workspaceID,
			Providers:   providers,
			Models:      models,
			Fallback:    fb,
		})
		if err != nil {
			return err
		}
		out = settingsFromJSON(row.Providers, row.Models)
		return nil
	})
	return out, err
}

func (p *Postgres) Provider(ctx context.Context, workspaceID uuid.UUID, role Role) (Provider, error) {
	b, err := p.Binding(ctx, workspaceID, role)
	if err != nil {
		return nil, err
	}
	return New(b)
}

func (p *Postgres) Binding(ctx context.Context, workspaceID uuid.UUID, role Role) (Binding, error) {
	var out Binding
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetWorkspaceAI(ctx, workspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoBinding
		}
		if err != nil {
			return err
		}
		var ps []storedProvider
		if len(row.Providers) > 0 {
			if err := json.Unmarshal(row.Providers, &ps); err != nil {
				return err
			}
		}
		if len(ps) == 0 {
			return ErrNoBinding
		}
		var mm map[string]string
		if len(row.Models) > 0 {
			_ = json.Unmarshal(row.Models, &mm)
		}
		kind, model, ok := splitKindModel(mm[string(role)])
		if !ok {
			kind, model, ok = splitKindModel(mm["compose"])
		}
		if !ok {
			kind = ps[0].Kind
		}
		out = Binding{Role: role, Kind: kind, Model: model, BaseURL: ps[0].BaseURL}
		if ps[0].SecretID == uuid.Nil {
			return ErrNoKey
		}
		sec, err := q.GetSecret(ctx, db.GetSecretParams{WorkspaceID: workspaceID, ID: ps[0].SecretID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoKey
		}
		if err != nil {
			return err
		}
		if p.keys == nil {
			return ErrNoKey
		}
		plain, err := p.keys.Open(crypto.Sealed{
			KeyVersion: int(sec.KeyVersion),
			WrappedDEK: sec.WrappedDek,
			Nonce:      sec.Nonce,
			Ciphertext: sec.Ciphertext,
		})
		if err != nil {
			return fmt.Errorf("ai: secret: %w", err)
		}
		out.APIKey = string(plain)
		return nil
	})
	return out, err
}
