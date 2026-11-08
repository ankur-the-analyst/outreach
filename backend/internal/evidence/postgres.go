package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"outreach/internal/db"
)

const (
	Dim           = 8
	IncludeAllMax = 30
	embedModel    = "fake-8"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrForbidden   = errors.New("forbidden")
	ErrUnconfirmed = errors.New("unconfirmed evidence cannot be cited")
	ErrClaim       = errors.New("number not in cited evidence")
	ErrEmbedDim    = errors.New("embedding dimension mismatch")
)

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type Item struct {
	ID        uuid.UUID  `json:"id"`
	ProfileID *uuid.UUID `json:"profile_id,omitempty"`
	Text      string     `json:"text"`
	SourceRef string     `json:"source_ref,omitempty"`
	Tags      []string   `json:"tags"`
	DoNotUse  bool       `json:"do_not_use"`
	Confirmed bool       `json:"confirmed"`
	Model     string     `json:"embedding_model,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type ProposeInput struct {
	Text      string
	SourceRef string
	Tags      []string
	ProfileID *uuid.UUID
}

type Postgres struct {
	pool       *db.Pool
	embed      Embedder
	model      string
	includeAll int
}

func NewPostgres(pool *db.Pool, embed Embedder) *Postgres {
	return &Postgres{pool: pool, embed: embed, model: embedModel, includeAll: IncludeAllMax}
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

func (p *Postgres) Propose(ctx context.Context, workspaceID uuid.UUID, in ProposeInput) (Item, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return Item{}, fmt.Errorf("evidence: text required")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Item{}, err
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}
	var out Item
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.InsertEvidence(ctx, db.InsertEvidenceParams{
			WorkspaceID: workspaceID,
			ID:          id,
			ProfileID:   in.ProfileID,
			Text:        text,
			SourceRef:   pgtype.Text{String: in.SourceRef, Valid: in.SourceRef != ""},
			Tags:        tags,
		})
		if err != nil {
			return err
		}
		out = itemFrom(row.ID, row.ProfileID, row.Text, row.SourceRef, row.Tags, row.DoNotUse, row.ConfirmedAt, row.EmbeddingModel, row.CreatedAt)
		return nil
	})
	return out, err
}

func (p *Postgres) List(ctx context.Context, workspaceID uuid.UUID) ([]Item, error) {
	var out []Item
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListEvidence(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]Item, 0, len(rows))
		for _, row := range rows {
			out = append(out, itemFrom(row.ID, row.ProfileID, row.Text, row.SourceRef, row.Tags, row.DoNotUse, row.ConfirmedAt, row.EmbeddingModel, row.CreatedAt))
		}
		return nil
	})
	return out, err
}

func (p *Postgres) Confirm(ctx context.Context, workspaceID, id uuid.UUID) (Item, error) {
	if p.embed == nil {
		return Item{}, fmt.Errorf("evidence: embedder required")
	}
	var text string
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetEvidence(ctx, db.GetEvidenceParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			return err
		}
		text = row.Text
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	vecs, err := p.embed.Embed(ctx, []string{text})
	if err != nil {
		return Item{}, fmt.Errorf("evidence: embed: %w", err)
	}
	if len(vecs) != 1 || len(vecs[0]) != Dim {
		return Item{}, ErrEmbedDim
	}
	var out Item
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.ConfirmEvidence(ctx, db.ConfirmEvidenceParams{
			WorkspaceID:    workspaceID,
			ID:             id,
			Embedding:      pgvector.NewVector(vecs[0]),
			EmbeddingModel: pgtype.Text{String: p.model, Valid: true},
		})
		if err != nil {
			return err
		}
		out = itemFrom(row.ID, row.ProfileID, row.Text, row.SourceRef, row.Tags, row.DoNotUse, row.ConfirmedAt, row.EmbeddingModel, row.CreatedAt)
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) Retrieve(ctx context.Context, workspaceID uuid.UUID, query string) ([]Item, error) {
	var out []Item
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		n, err := q.CountConfirmedEvidence(ctx, db.CountConfirmedEvidenceParams{WorkspaceID: workspaceID, EmbeddingModel: pgtype.Text{String: p.model, Valid: true}})
		if err != nil {
			return err
		}
		if n <= int64(p.includeAll) {
			rows, err := q.ListConfirmedEvidence(ctx, db.ListConfirmedEvidenceParams{WorkspaceID: workspaceID, EmbeddingModel: pgtype.Text{String: p.model, Valid: true}})
			if err != nil {
				return err
			}
			out = itemsFromConfirmed(rows)
			return nil
		}
		if p.embed == nil {
			return fmt.Errorf("evidence: embedder required")
		}
		vecs, err := p.embed.Embed(ctx, []string{query})
		if err != nil {
			return err
		}
		if len(vecs) != 1 || len(vecs[0]) != Dim {
			return ErrEmbedDim
		}
		rows, err := q.RetrieveEvidence(ctx, db.RetrieveEvidenceParams{
			WorkspaceID:    workspaceID,
			EmbeddingModel: pgtype.Text{String: p.model, Valid: true},
			Query:          pgvector.NewVector(vecs[0]),
		})
		if err != nil {
			return err
		}
		out = itemsFromRetrieve(rows)
		return nil
	})
	return out, err
}

func (p *Postgres) Cite(ctx context.Context, workspaceID uuid.UUID, ids []uuid.UUID) ([]Item, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []db.GetEvidenceByIDsRow
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		var err error
		rows, err = q.GetEvidenceByIDs(ctx, db.GetEvidenceByIDsParams{WorkspaceID: workspaceID, Ids: ids})
		return err
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]db.GetEvidenceByIDsRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	out := make([]Item, 0, len(ids))
	for _, id := range ids {
		row, ok := byID[id]
		if !ok || !row.ConfirmedAt.Valid || row.DoNotUse {
			return nil, ErrUnconfirmed
		}
		out = append(out, itemFrom(row.ID, row.ProfileID, row.Text, row.SourceRef, row.Tags, row.DoNotUse, row.ConfirmedAt, row.EmbeddingModel, row.CreatedAt))
	}
	return out, nil
}

func itemFrom(id uuid.UUID, profileID *uuid.UUID, text string, source pgtype.Text, tags []string, doNotUse bool, confirmed pgtype.Timestamptz, model pgtype.Text, created pgtype.Timestamptz) Item {
	if tags == nil {
		tags = []string{}
	}
	it := Item{ID: id, ProfileID: profileID, Text: text, Tags: tags, DoNotUse: doNotUse, Confirmed: confirmed.Valid}
	if source.Valid {
		it.SourceRef = source.String
	}
	if model.Valid {
		it.Model = model.String
	}
	if created.Valid {
		it.CreatedAt = created.Time.UTC()
	}
	return it
}

func itemsFromConfirmed(rows []db.ListConfirmedEvidenceRow) []Item {
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemFrom(row.ID, row.ProfileID, row.Text, row.SourceRef, row.Tags, row.DoNotUse, row.ConfirmedAt, row.EmbeddingModel, row.CreatedAt))
	}
	return out
}

func itemsFromRetrieve(rows []db.RetrieveEvidenceRow) []Item {
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemFrom(row.ID, row.ProfileID, row.Text, row.SourceRef, row.Tags, row.DoNotUse, row.ConfirmedAt, row.EmbeddingModel, row.CreatedAt))
	}
	return out
}
