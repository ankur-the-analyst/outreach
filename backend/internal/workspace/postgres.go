package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"outreach/internal/db"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrForbidden     = errors.New("forbidden")
	ErrAlreadyMember = errors.New("already a member")
)

type Workspace struct {
	ID          uuid.UUID
	Name        string
	SendsPaused bool
}

type Member struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Name   string    `json:"name"`
	Role   string    `json:"role"`
}

type Postgres struct {
	pool *db.Pool
}

func NewPostgres(pool *db.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) Get(ctx context.Context, workspaceID uuid.UUID) (Workspace, error) {
	var out Workspace
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = Workspace{ID: row.ID, Name: row.Name, SendsPaused: row.SendsPaused}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, ErrNotFound
	}
	return out, err
}

func (p *Postgres) SetName(ctx context.Context, workspaceID uuid.UUID, name string) error {
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.UpdateWorkspaceName(ctx, db.UpdateWorkspaceNameParams{ID: workspaceID, Name: name})
	})
}

func (p *Postgres) SetPaused(ctx context.Context, workspaceID uuid.UUID, paused bool) error {
	return p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.SetSendsPaused(ctx, db.SetSendsPausedParams{ID: workspaceID, SendsPaused: paused})
	})
}

func (p *Postgres) Delete(ctx context.Context, workspaceID uuid.UUID) error {
	return p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.DeleteSessionsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: sessions: %w", err)
		}
		if err := q.DeleteMembershipsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: memberships: %w", err)
		}
		if err := q.DeleteDraftsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: drafts: %w", err)
		}
		if err := q.DeleteMessagesForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: messages: %w", err)
		}
		if err := q.DeleteThreadsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: threads: %w", err)
		}
		if err := q.DeleteSyncStateForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: sync: %w", err)
		}
		if err := q.DeleteMailboxesForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: mailboxes: %w", err)
		}
		if err := q.DeleteGoogleServiceAccountsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: service accounts: %w", err)
		}
		if err := q.DeleteSecretsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: secrets: %w", err)
		}
		if err := q.DeleteWarmupProfilesForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: warmup: %w", err)
		}
		if err := q.DeleteSendingDomainsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: domains: %w", err)
		}
		if err := q.DeleteSuppressionForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: suppression: %w", err)
		}
		if err := q.DeleteEvidenceForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: evidence: %w", err)
		}
		if err := q.DeleteFitsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: fits: %w", err)
		}
		if err := q.DeleteResumesForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: resumes: %w", err)
		}
		if err := q.DeleteJobsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: jobs: %w", err)
		}
		if err := q.DeleteProfilesForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: profiles: %w", err)
		}
		if err := q.DeleteContactsForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: contacts: %w", err)
		}
		if err := q.DeleteCompaniesForWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("workspace: companies: %w", err)
		}
		return q.DeleteWorkspace(ctx, workspaceID)
	})
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

func (p *Postgres) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]Member, error) {
	var out []Member
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		rows, err := q.ListMembers(ctx, workspaceID)
		if err != nil {
			return err
		}
		out = make([]Member, 0, len(rows))
		for _, row := range rows {
			out = append(out, Member{
				UserID: row.UserID,
				Email:  row.Email,
				Name:   row.Name.String,
				Role:   row.Role,
			})
		}
		return nil
	})
	return out, err
}

func (p *Postgres) UserByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	u, err := p.pool.Queries().GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return u.ID, nil
}

func (p *Postgres) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error {
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertMembership(ctx, db.InsertMembershipParams{
			WorkspaceID: workspaceID,
			UserID:      userID,
			Role:        role,
		})
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyMember
	}
	return err
}
