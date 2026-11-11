package cv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"outreach/internal/db"
	"outreach/internal/evidence"
	"outreach/internal/mime"
	"outreach/internal/storage"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrNoCV      = errors.New("add a CV to this profile")
	ErrFrom      = errors.New("this file's From does not match the mailbox")
)

type Document struct {
	ID        uuid.UUID `json:"id"`
	ProfileID uuid.UUID `json:"profile_id"`
	Version   int32     `json:"version"`
	Spec      Spec      `json:"spec"`
}

type Render struct {
	ID        uuid.UUID `json:"id"`
	FileKey   string    `json:"file_key"`
	FileName  string    `json:"file_name"`
	FromEmail string    `json:"from_email"`
	Size      int32     `json:"size"`
}

type Postgres struct {
	pool  *db.Pool
	blobs storage.Store
	json  JSONCompleter
}

func NewPostgres(pool *db.Pool, blobs storage.Store) *Postgres {
	return &Postgres{pool: pool, blobs: blobs}
}

func (p *Postgres) WithJSON(c JSONCompleter) *Postgres {
	p.json = c
	return p
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

func (p *Postgres) Save(ctx context.Context, workspaceID, profileID uuid.UUID, spec Spec) (Document, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Name == "" {
		return Document{}, fmt.Errorf("cv: name required")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Document{}, err
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return Document{}, err
	}
	var out Document
	err = p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.GetProfile(ctx, db.GetProfileParams{WorkspaceID: workspaceID, ID: profileID}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		ver, err := q.NextCvVersion(ctx, db.NextCvVersionParams{WorkspaceID: workspaceID, ProfileID: profileID})
		if err != nil {
			return err
		}
		row, err := q.InsertCvDocument(ctx, db.InsertCvDocumentParams{
			WorkspaceID: workspaceID, ID: id, ProfileID: profileID, Version: ver, Spec: raw,
		})
		if err != nil {
			return err
		}
		out = Document{ID: row.ID, ProfileID: row.ProfileID, Version: row.Version, Spec: spec}
		return nil
	})
	return out, err
}

func (p *Postgres) Stamp(ctx context.Context, q *db.Queries, workspaceID, profileID, mailboxID uuid.UUID) (Render, error) {
	cvRow, err := q.LatestCvByProfile(ctx, db.LatestCvByProfileParams{WorkspaceID: workspaceID, ProfileID: profileID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Render{}, ErrNoCV
	}
	if err != nil {
		return Render{}, err
	}
	mb, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: mailboxID})
	if err != nil {
		return Render{}, err
	}
	var spec Spec
	if err := json.Unmarshal(cvRow.Spec, &spec); err != nil {
		return Render{}, err
	}
	from := mb.Email
	existing, err := q.GetUntailoredRender(ctx, db.GetUntailoredRenderParams{
		WorkspaceID: workspaceID, CvID: cvRow.ID, MailboxID: mailboxID,
	})
	if err == nil {
		if _, err := p.bytes(ctx, existing.FileKey, spec, from); err != nil {
			return Render{}, err
		}
		return renderFrom(existing.ID, existing.FileKey, existing.FileName, existing.FromEmail, existing.Size), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Render{}, err
	}
	rid, err := uuid.NewV7()
	if err != nil {
		return Render{}, err
	}
	key := workspaceID.String() + "/cv/" + rid.String() + ".pdf"
	name := FileName(spec)
	pdf, err := p.bytes(ctx, key, spec, from)
	if err != nil {
		return Render{}, err
	}
	row, err := q.InsertCvRender(ctx, db.InsertCvRenderParams{
		WorkspaceID: workspaceID,
		ID:          rid,
		CvID:        cvRow.ID,
		MailboxID:   mailboxID,
		FileKey:     key,
		FileName:    name,
		Mime:        "application/pdf",
		Size:        int32(len(pdf)),
		FromEmail:   from,
	})
	if isUnique(err) {
		got, err := q.GetUntailoredRender(ctx, db.GetUntailoredRenderParams{
			WorkspaceID: workspaceID, CvID: cvRow.ID, MailboxID: mailboxID,
		})
		if err != nil {
			return Render{}, err
		}
		return renderFrom(got.ID, got.FileKey, got.FileName, got.FromEmail, got.Size), nil
	}
	if err != nil {
		return Render{}, err
	}
	return renderFrom(row.ID, row.FileKey, row.FileName, row.FromEmail, row.Size), nil
}

func (p *Postgres) Attach(ctx context.Context, q *db.Queries, workspaceID, profileID, mailboxID uuid.UUID, jobID *uuid.UUID, cited []evidence.Item, jd string) (Render, error) {
	if jobID == nil || *jobID == uuid.Nil || p.json == nil {
		return p.Stamp(ctx, q, workspaceID, profileID, mailboxID)
	}
	cvRow, err := q.LatestCvByProfile(ctx, db.LatestCvByProfileParams{WorkspaceID: workspaceID, ProfileID: profileID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Render{}, ErrNoCV
	}
	if err != nil {
		return Render{}, err
	}
	var base Spec
	if err := json.Unmarshal(cvRow.Spec, &base); err != nil {
		return Render{}, err
	}
	spec, _, err := Propose(ctx, p.json, base, jd, cited)
	if err != nil {
		return Render{}, err
	}
	return p.stampSpec(ctx, q, workspaceID, cvRow.ID, mailboxID, jobID, spec)
}

func (p *Postgres) stampSpec(ctx context.Context, q *db.Queries, workspaceID, cvID, mailboxID uuid.UUID, jobID *uuid.UUID, spec Spec) (Render, error) {
	mb, err := q.GetMailbox(ctx, db.GetMailboxParams{WorkspaceID: workspaceID, ID: mailboxID})
	if err != nil {
		return Render{}, err
	}
	from := mb.Email
	existing, err := q.GetTailoredRender(ctx, db.GetTailoredRenderParams{
		WorkspaceID: workspaceID, CvID: cvID, MailboxID: mailboxID, JobID: jobID,
	})
	if err == nil {
		if _, err := p.bytes(ctx, existing.FileKey, spec, from); err != nil {
			return Render{}, err
		}
		return renderFrom(existing.ID, existing.FileKey, existing.FileName, existing.FromEmail, existing.Size), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Render{}, err
	}
	rid, err := uuid.NewV7()
	if err != nil {
		return Render{}, err
	}
	key := workspaceID.String() + "/cv/" + rid.String() + ".pdf"
	name := FileName(spec)
	pdf, err := p.bytes(ctx, key, spec, from)
	if err != nil {
		return Render{}, err
	}
	row, err := q.InsertCvRender(ctx, db.InsertCvRenderParams{
		WorkspaceID: workspaceID,
		ID:          rid,
		CvID:        cvID,
		MailboxID:   mailboxID,
		JobID:       jobID,
		FileKey:     key,
		FileName:    name,
		Mime:        "application/pdf",
		Size:        int32(len(pdf)),
		FromEmail:   from,
	})
	if isUnique(err) {
		got, err := q.GetTailoredRender(ctx, db.GetTailoredRenderParams{
			WorkspaceID: workspaceID, CvID: cvID, MailboxID: mailboxID, JobID: jobID,
		})
		if err != nil {
			return Render{}, err
		}
		return renderFrom(got.ID, got.FileKey, got.FileName, got.FromEmail, got.Size), nil
	}
	if err != nil {
		return Render{}, err
	}
	return renderFrom(row.ID, row.FileKey, row.FileName, row.FromEmail, row.Size), nil
}

func (p *Postgres) StampFor(ctx context.Context, workspaceID, profileID, mailboxID uuid.UUID) (Render, error) {
	var out Render
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		r, err := p.Stamp(ctx, q, workspaceID, profileID, mailboxID)
		out = r
		return err
	})
	return out, err
}

func (p *Postgres) Files(ctx context.Context, workspaceID, ledgerID, mailboxID uuid.UUID, from string) ([]mime.File, error) {
	var out []mime.File
	err := p.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		led, err := q.GetLedger(ctx, db.GetLedgerParams{WorkspaceID: workspaceID, ID: ledgerID})
		if err != nil {
			return err
		}
		if led.DraftID == uuid.Nil {
			return nil
		}
		d, err := q.GetDraftAttach(ctx, db.GetDraftAttachParams{WorkspaceID: workspaceID, ID: led.DraftID})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.EnrolmentID == nil {
			return nil
		}
		en, err := q.GetEnrolment(ctx, db.GetEnrolmentParams{WorkspaceID: workspaceID, ID: *d.EnrolmentID})
		if err != nil {
			return err
		}
		if en.Intent != "job" {
			return nil
		}
		if d.AttachmentRenderID != nil {
			row, err := q.GetCvRender(ctx, db.GetCvRenderParams{WorkspaceID: workspaceID, ID: *d.AttachmentRenderID})
			if err == nil && row.MailboxID == mailboxID {
				if !strings.EqualFold(row.FromEmail, from) {
					return ErrFrom
				}
				b, err := p.blobs.Get(ctx, row.FileKey)
				if err == nil {
					out = []mime.File{{Name: row.FileName, ContentType: "application/pdf", Bytes: b}}
					return nil
				}
				if !errors.Is(err, storage.ErrNotFound) {
					return err
				}
			}
		}
		profileID := en.ProfileID
		if profileID == nil {
			camp, err := q.GetCampaign(ctx, db.GetCampaignParams{WorkspaceID: workspaceID, ID: en.CampaignID})
			if err != nil {
				return err
			}
			profileID = camp.ProfileID
		}
		if profileID == nil {
			return ErrNoCV
		}
		jd := ""
		if en.JobID != nil {
			job, err := q.GetJob(ctx, db.GetJobParams{WorkspaceID: workspaceID, ID: *en.JobID})
			if err == nil {
				jd = job.RawText
			}
		}
		r, err := p.Attach(ctx, q, workspaceID, *profileID, mailboxID, en.JobID, nil, jd)
		if err != nil {
			return err
		}
		if !strings.EqualFold(r.FromEmail, from) {
			return ErrFrom
		}
		if err := q.UpdateDraftAttachment(ctx, db.UpdateDraftAttachmentParams{
			WorkspaceID: workspaceID, ID: led.DraftID, AttachmentRenderID: &r.ID,
		}); err != nil {
			return err
		}
		b, err := p.blobs.Get(ctx, r.FileKey)
		if err != nil {
			return err
		}
		out = []mime.File{{Name: r.FileName, ContentType: "application/pdf", Bytes: b}}
		return nil
	})
	return out, err
}

func (p *Postgres) bytes(ctx context.Context, key string, spec Spec, from string) ([]byte, error) {
	if key != "" {
		b, err := p.blobs.Get(ctx, key)
		if err == nil {
			return b, nil
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
	}
	pdf, err := PDF(spec, from)
	if err != nil {
		return nil, err
	}
	if key != "" {
		if err := p.blobs.Put(ctx, key, "application/pdf", pdf); err != nil {
			return nil, err
		}
	}
	return pdf, nil
}

func renderFrom(id uuid.UUID, key, name, from string, size int32) Render {
	return Render{ID: id, FileKey: key, FileName: name, FromEmail: from, Size: size}
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
