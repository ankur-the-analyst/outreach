package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"outreach/internal/db"
)

type StepArgs struct {
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	EnrolmentID  uuid.UUID `json:"enrolment_id"`
	StepPosition int32     `json:"step_position"`
}

func (StepArgs) Kind() string { return "campaign.step" }

func (StepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

type StepWorker struct {
	river.WorkerDefaults[StepArgs]
}

func (StepWorker) Work(ctx context.Context, job *river.Job[StepArgs]) error {
	if QueriesFrom(ctx) == nil {
		return fmt.Errorf("worker: guc queries missing")
	}
	if job.Args.EnrolmentID == uuid.Nil {
		return fmt.Errorf("worker: enrolment_id required")
	}
	return nil
}

type ClaimResult struct {
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

func ClaimDue(ctx context.Context, pool *db.Pool, client *river.Client[pgx.Tx], workspaceID uuid.UUID, now time.Time) (ClaimResult, error) {
	var out ClaimResult
	err := pool.ExecTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries, tx pgx.Tx) error {
		ws, err := q.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return fmt.Errorf("worker: claim due: %w", err)
		}
		if ws.SendsPaused {
			return nil
		}
		rows, err := q.ListDueEmailSteps(ctx, db.ListDueEmailStepsParams{
			WorkspaceID: workspaceID,
			Now:         pgtype.Timestamptz{Time: now.UTC(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("worker: list due steps: %w", err)
		}
		for _, row := range rows {
			res, err := client.InsertTx(ctx, tx, StepArgs{
				WorkspaceID:  workspaceID,
				EnrolmentID:  row.EnrolmentID,
				StepPosition: row.StepPosition,
			}, nil)
			if err != nil {
				return fmt.Errorf("worker: insert step: %w", err)
			}
			if res.UniqueSkippedAsDuplicate {
				out.Skipped++
				continue
			}
			out.Inserted++
		}
		return nil
	})
	return out, err
}
