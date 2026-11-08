package contacts

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/db"
	"outreach/internal/suppression"
)

func (p *Postgres) WithHMAC(h suppression.Hasher) *Postgres {
	p.hmac = h
	return p
}

func (p *Postgres) Erase(ctx context.Context, workspaceID, id uuid.UUID) error {
	if p.hmac == nil {
		return fmt.Errorf("contacts: hasher required")
	}
	return p.pool.InTx(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetContact(ctx, db.GetContactParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		sum, email, err := suppression.Hash(p.hmac, suppression.KindEmail, row.Email)
		if err != nil {
			return err
		}
		if err := q.InsertSuppression(ctx, db.InsertSuppressionParams{
			WorkspaceID: workspaceID,
			Kind:        suppression.KindEmail,
			ValueHash:   sum,
			ValueCitext: pgtype.Text{String: email, Valid: true},
			Reason:      "erasure",
		}); err != nil {
			return err
		}
		args := db.DeleteAttemptsForContactParams{WorkspaceID: workspaceID, ContactID: id, Email: email}
		if err := q.DeleteAttemptsForContact(ctx, args); err != nil {
			return err
		}
		if err := q.DeleteLedgerForContact(ctx, db.DeleteLedgerForContactParams{
			WorkspaceID: workspaceID, ContactID: id, Email: email,
		}); err != nil {
			return err
		}
		if err := q.DeleteMessagesForContact(ctx, db.DeleteMessagesForContactParams{
			WorkspaceID: workspaceID, Email: text(email), ContactID: id,
		}); err != nil {
			return err
		}
		cid := id
		if err := q.DeleteThreadsForContact(ctx, db.DeleteThreadsForContactParams{
			WorkspaceID: workspaceID, ContactID: &cid,
		}); err != nil {
			return err
		}
		if err := q.DeleteDraftsForContact(ctx, db.DeleteDraftsForContactParams{
			WorkspaceID: workspaceID, Email: email, ContactID: id,
		}); err != nil {
			return err
		}
		if err := q.DeleteEnrolmentsForContact(ctx, db.DeleteEnrolmentsForContactParams{
			WorkspaceID: workspaceID, ContactID: id,
		}); err != nil {
			return err
		}
		if err := q.DeleteContact(ctx, db.DeleteContactParams{WorkspaceID: workspaceID, ID: id}); err != nil {
			return err
		}
		return q.ClearSuppressionPlaintext(ctx, db.ClearSuppressionPlaintextParams{
			WorkspaceID: workspaceID,
			Kind:        suppression.KindEmail,
			ValueHash:   sum,
		})
	})
}
