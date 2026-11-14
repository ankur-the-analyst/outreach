package fields

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
)

type Def struct {
	ID       uuid.UUID       `json:"id"`
	Object   string          `json:"object"`
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Kind     string          `json:"kind"`
	Required bool            `json:"required"`
	Options  json.RawMessage `json:"options,omitempty"`
	Position int32           `json:"position"`
}

type CreateInput struct {
	Object   string
	Key      string
	Label    string
	Kind     string
	Required bool
}

type Hint struct {
	Key   string
	Label string
}

type Postgres struct {
	pool *db.Pool
}

func NewPostgres(pool *db.Pool) *Postgres {
	return &Postgres{pool: pool}
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

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID, object string) ([]Def, error) {
	var out []Def
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		arg := db.ListFieldDefsParams{WorkspaceID: workspaceID}
		if object != "" {
			arg.Object = pgtype.Text{String: object, Valid: true}
		}
		rows, err := q.ListFieldDefs(ctx, arg)
		if err != nil {
			return err
		}
		out = make([]Def, 0, len(rows))
		for _, row := range rows {
			out = append(out, defFrom(row))
		}
		return nil
	})
	return out, err
}

func (p *Postgres) AllowedCustom(ctx context.Context, workspaceID uuid.UUID, object string) (map[string]struct{}, error) {
	defs, err := p.List(ctx, workspaceID, object)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(defs))
	for _, d := range defs {
		out[d.Key] = struct{}{}
	}
	return out, nil
}

func (p *Postgres) Hints(ctx context.Context, workspaceID uuid.UUID, object string) ([]Hint, error) {
	defs, err := p.List(ctx, workspaceID, object)
	if err != nil {
		return nil, err
	}
	out := make([]Hint, 0, len(defs))
	for _, d := range defs {
		out = append(out, Hint{Key: d.Key, Label: d.Label})
	}
	return out, nil
}

func (p *Postgres) Mine(ctx context.Context, workspaceID, userID uuid.UUID) (map[string]string, error) {
	var out map[string]string
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID, UserID: userID}, func(ctx context.Context, q *db.Queries) error {
		raw, err := q.GetMembershipAttrs(ctx, db.GetMembershipAttrsParams{WorkspaceID: workspaceID, UserID: userID})
		if err != nil {
			return err
		}
		if len(raw) == 0 {
			out = map[string]string{}
			return nil
		}
		return json.Unmarshal(raw, &out)
	})
	if out == nil {
		out = map[string]string{}
	}
	return out, err
}

func (p *Postgres) SetMine(ctx context.Context, workspaceID, userID uuid.UUID, values map[string]string) error {
	allowed, err := p.AllowedCustom(ctx, workspaceID, "member")
	if err != nil {
		return err
	}
	clean := map[string]string{}
	for k, v := range values {
		if _, ok := allowed[k]; !ok {
			continue
		}
		clean[k] = strings.TrimSpace(v)
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return err
	}
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID, UserID: userID}, func(ctx context.Context, q *db.Queries) error {
		return q.UpdateMembershipAttrs(ctx, db.UpdateMembershipAttrsParams{
			WorkspaceID: workspaceID, UserID: userID, Attrs: raw,
		})
	})
}

func (p *Postgres) Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Def, error) {
	object := strings.TrimSpace(in.Object)
	key := strings.TrimSpace(in.Key)
	label := strings.TrimSpace(in.Label)
	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		kind = "text"
	}
	if !ValidObject(object) {
		return Def{}, ErrObject
	}
	if !ValidKey(key) {
		return Def{}, ErrKey
	}
	if IsBuiltin(object, key) {
		return Def{}, ErrBuiltin
	}
	if !ValidKind(kind) {
		return Def{}, ErrKind
	}
	if label == "" {
		return Def{}, ErrLabel
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Def{}, err
	}
	var out Def
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		n, err := q.CountFieldDefs(ctx, db.CountFieldDefsParams{WorkspaceID: workspaceID, Object: object})
		if err != nil {
			return err
		}
		if n >= MaxPerObject {
			return ErrLimit
		}
		row, err := q.InsertFieldDef(ctx, db.InsertFieldDefParams{
			WorkspaceID: workspaceID,
			ID:          id,
			Object:      object,
			Key:         key,
			Label:       label,
			Kind:        kind,
			Required:    in.Required,
			Options:     json.RawMessage("[]"),
			Position:    n,
		})
		if err != nil {
			return err
		}
		out = defFrom(row)
		return nil
	})
	if isUnique(err) {
		return Def{}, ErrDuplicate
	}
	return out, err
}

func defFrom(row db.FieldDef) Def {
	return Def{
		ID: row.ID, Object: row.Object, Key: row.Key, Label: row.Label,
		Kind: row.Kind, Required: row.Required, Options: row.Options, Position: row.Position,
	}
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
