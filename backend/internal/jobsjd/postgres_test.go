package jobsjd

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/ai"
	"outreach/internal/db"
	"outreach/internal/storage"
)

func TestProfileResumeJobFit(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteFitsForWorkspace(ctx, wsID)
			_ = q.DeleteResumesForWorkspace(ctx, wsID)
			_ = q.DeleteJobsForWorkspace(ctx, wsID)
			_ = q.DeleteProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m54"})
	})
	if err != nil {
		t.Fatal(err)
	}
	blobs := storage.NewMemory()
	store := NewPostgres(pool, blobs, &ai.Fake{JSON: json.RawMessage(`{"score":82,"summary":"deals overlap"}`)})
	prof, err := store.CreateProfile(ctx, wsID, "Ada", "seller")
	if err != nil {
		t.Fatal(err)
	}
	res, err := store.UploadResume(ctx, wsID, prof.ID, "cv.txt", "text/plain", "Closed 12 deals last year.")
	if err != nil || res.Version != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	got, err := blobs.Get(ctx, res.FileKey)
	if err != nil || string(got) != "Closed 12 deals last year." {
		t.Fatalf("%s %v", got, err)
	}
	job, err := store.CreateJob(ctx, wsID, "AE", "", "Need someone who closed enterprise deals.")
	if err != nil || job.AnalysisStatus != "ready" || string(job.Analysis) != `{"score":82,"summary":"deals overlap"}` {
		t.Fatalf("%+v %v", job, err)
	}
	fit, err := store.Fit(ctx, wsID, job.ID, prof.ID)
	if err != nil || fit.Score == nil || *fit.Score != 82 || fit.ResumeID == nil {
		t.Fatalf("%+v %v", fit, err)
	}
}
