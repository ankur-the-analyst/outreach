package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"outreach/internal/ai"
	"outreach/internal/auth"
	"outreach/internal/campaign"
	"outreach/internal/compose"
	"outreach/internal/config"
	"outreach/internal/contacts"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/evidence"
	"outreach/internal/health"
	"outreach/internal/inbound"
	"outreach/internal/jobsjd"
	"outreach/internal/mailbox"
	"outreach/internal/mailbox/smtpimap"
	"outreach/internal/sender"
	"outreach/internal/storage"
	"outreach/internal/suppression"
	mailsync "outreach/internal/sync"
	"outreach/internal/worker"
	"outreach/internal/workspace"
)

type reconcileJobs struct {
	client *river.Client[pgx.Tx]
}

type classifyJobs struct {
	client *river.Client[pgx.Tx]
}

func (j classifyJobs) Enqueue(ctx context.Context, workspaceID, messageID uuid.UUID) error {
	return worker.EnqueueClassify(ctx, j.client, workspaceID, messageID)
}

func (j reconcileJobs) EnqueueReconcile(ctx context.Context, workspaceID, ledgerID uuid.UUID, in sender.SendInput) error {
	return worker.EnqueueReconcile(ctx, j.client, worker.ReconcileArgs{
		WorkspaceID: workspaceID,
		LedgerID:    ledgerID,
		To:          in.To,
		Subject:     in.Subject,
		Text:        in.Text,
		HTML:        in.HTML,
	})
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := db.Open(ctx, cfg.DatabaseAppURL)
	cancel()
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	keys, err := crypto.NewKeyring(cfg.MasterKeys, cfg.MasterKeyActive)
	if err != nil {
		logger.Error("load master keys", "err", err)
		os.Exit(1)
	}

	a := auth.New(auth.NewPostgres(pool))
	wsStore := workspace.NewPostgres(pool)
	ws := workspace.New(wsStore)
	mb := mailbox.New(mailbox.NewPostgres(pool, keys), smtpimap.Probe{})
	jobs, err := worker.NewInsertClient(pool)
	if err != nil {
		logger.Error("river client", "err", err)
		os.Exit(1)
	}
	wrk := worker.New(pool, wsStore, jobs)
	hold := time.Duration(cfg.UndoHoldS) * time.Second
	unknown := time.Duration(cfg.UnknownTimeoutS) * time.Second
	suppStore := suppression.NewPostgres(pool, keys)
	supp := suppression.New(suppStore)
	led := sender.NewPostgres(pool, time.Now, hold).
		WithDialer(sender.NewMailboxDialer(pool, keys)).
		WithUnknownTimeout(unknown).
		WithQueue(reconcileJobs{client: jobs}).
		WithBlocker(suppStore)
	snd := sender.New(led)
	writer := &ai.Fake{
		Model: "fake-8", Dim: evidence.Dim,
		Text: "You closed 12 deals last year. Open to a short chat?",
		JSON: json.RawMessage(`{"score":82,"summary":"deals overlap"}`),
	}
	evStore := evidence.NewPostgres(pool, writer)
	ev := evidence.New(evStore)
	cmp := compose.New(compose.NewPostgres(pool, led).WithWriter(writer).WithCite(evStore))
	proc := inbound.New(pool, suppStore, classifyJobs{client: jobs})
	sy := mailsync.New(mailsync.NewPostgres(pool, mailsync.Mailpit{Base: cfg.MailpitAPI}).WithInbound(proc))
	jd := jobsjd.New(jobsjd.NewPostgres(pool, storage.NewMemory(), writer))
	ct := contacts.New(contacts.NewPostgres(pool))
	packs, err := campaign.LoadPacks()
	if err != nil {
		logger.Error("load packs", "err", err)
		os.Exit(1)
	}
	camp := campaign.New(campaign.NewPostgres(pool, packs))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Handler(pool))
	mux.HandleFunc("POST /api/v1/auth/register", a.Register)
	mux.HandleFunc("POST /api/v1/auth/login", a.Login)
	mux.Handle("POST /api/v1/auth/logout", a.Require(http.HandlerFunc(a.Logout)))
	mux.Handle("GET /api/v1/me", a.Require(http.HandlerFunc(a.Me)))
	mux.Handle("GET /api/v1/workspace", a.Require(http.HandlerFunc(ws.Get)))
	mux.Handle("PATCH /api/v1/workspace", a.Require(http.HandlerFunc(ws.Patch)))
	mux.Handle("DELETE /api/v1/workspace", a.Require(http.HandlerFunc(ws.Delete)))
	mux.Handle("PUT /api/v1/workspace/kill-switch", a.Require(http.HandlerFunc(ws.PutKillSwitch)))
	mux.Handle("GET /api/v1/workspace/members", a.Require(http.HandlerFunc(ws.ListMembers)))
	mux.Handle("POST /api/v1/workspace/members", a.Require(http.HandlerFunc(ws.Invite)))
	mux.Handle("GET /api/v1/mailboxes", a.Require(http.HandlerFunc(mb.List)))
	mux.Handle("POST /api/v1/mailboxes", a.Require(http.HandlerFunc(mb.Create)))
	mux.Handle("PATCH /api/v1/mailboxes/{id}", a.Require(http.HandlerFunc(mb.Patch)))
	mux.Handle("POST /api/v1/mailboxes/{id}/pause", a.Require(http.HandlerFunc(mb.Pause)))
	mux.Handle("POST /api/v1/mailboxes/{id}/reconnect", a.Require(http.HandlerFunc(mb.Reconnect)))
	mux.Handle("POST /api/v1/mailboxes/{id}/test", a.Require(http.HandlerFunc(mb.Test)))
	mux.Handle("GET /api/v1/google-service-accounts", a.Require(http.HandlerFunc(mb.ListSA)))
	mux.Handle("POST /api/v1/google-service-accounts", a.Require(http.HandlerFunc(mb.CreateSA)))
	mux.Handle("POST /api/v1/jobs/ping", a.Require(http.HandlerFunc(wrk.Ping)))
	mux.Handle("POST /api/v1/jobs/schedule", a.Require(http.HandlerFunc(wrk.Schedule)))
	mux.Handle("POST /api/v1/mailboxes/{id}/holds", a.Require(http.HandlerFunc(snd.Hold)))
	mux.Handle("GET /api/v1/mailboxes/{id}/usage", a.Require(http.HandlerFunc(snd.Usage)))
	mux.Handle("POST /api/v1/ledger/{id}/cancel", a.Require(http.HandlerFunc(snd.Cancel)))
	mux.Handle("POST /api/v1/reserve", a.Require(http.HandlerFunc(snd.Reserve)))
	mux.Handle("POST /api/v1/ledger/{id}/send", a.Require(http.HandlerFunc(snd.Send)))
	mux.Handle("POST /api/v1/ledger/{id}/reconcile", a.Require(http.HandlerFunc(snd.Reconcile)))
	mux.Handle("POST /api/v1/suppression", a.Require(http.HandlerFunc(supp.Create)))
	mux.Handle("GET /api/v1/drafts", a.Require(http.HandlerFunc(cmp.List)))
	mux.Handle("POST /api/v1/drafts", a.Require(http.HandlerFunc(cmp.Create)))
	mux.Handle("POST /api/v1/drafts/compose", a.Require(http.HandlerFunc(cmp.Compose)))
	mux.Handle("POST /api/v1/drafts/{id}/approve", a.Require(http.HandlerFunc(cmp.Approve)))
	mux.Handle("POST /api/v1/drafts/{id}/undo", a.Require(http.HandlerFunc(cmp.Undo)))
	mux.Handle("POST /api/v1/mailboxes/{id}/sync", a.Require(http.HandlerFunc(sy.Sync)))
	mux.Handle("GET /api/v1/mailboxes/{id}/threads", a.Require(http.HandlerFunc(sy.Threads)))
	mux.Handle("POST /api/v1/threads/{id}/read", a.Require(http.HandlerFunc(sy.MarkRead)))
	mux.Handle("GET /api/v1/evidence", a.Require(http.HandlerFunc(ev.List)))
	mux.Handle("POST /api/v1/evidence", a.Require(http.HandlerFunc(ev.Propose)))
	mux.Handle("POST /api/v1/evidence/{id}/confirm", a.Require(http.HandlerFunc(ev.Confirm)))
	mux.Handle("GET /api/v1/profiles", a.Require(http.HandlerFunc(jd.ListProfiles)))
	mux.Handle("POST /api/v1/profiles", a.Require(http.HandlerFunc(jd.CreateProfile)))
	mux.Handle("POST /api/v1/profiles/{id}/resumes", a.Require(http.HandlerFunc(jd.UploadResume)))
	mux.Handle("GET /api/v1/jobs", a.Require(http.HandlerFunc(jd.ListJobs)))
	mux.Handle("POST /api/v1/jobs", a.Require(http.HandlerFunc(jd.CreateJob)))
	mux.Handle("POST /api/v1/jobs/{id}/fit", a.Require(http.HandlerFunc(jd.Fit)))
	mux.Handle("GET /api/v1/contacts", a.Require(http.HandlerFunc(ct.List)))
	mux.Handle("POST /api/v1/contacts", a.Require(http.HandlerFunc(ct.Create)))
	mux.Handle("POST /api/v1/contacts/preview", a.Require(http.HandlerFunc(ct.Preview)))
	mux.Handle("GET /api/v1/contacts/{id}", a.Require(http.HandlerFunc(ct.Get)))
	mux.Handle("PATCH /api/v1/contacts/{id}", a.Require(http.HandlerFunc(ct.Patch)))
	mux.Handle("DELETE /api/v1/contacts/{id}", a.Require(http.HandlerFunc(ct.Delete)))
	mux.Handle("GET /api/v1/packs", a.Require(http.HandlerFunc(camp.ListPacks)))
	mux.Handle("GET /api/v1/sender-identity", a.Require(http.HandlerFunc(camp.GetIdentity)))
	mux.Handle("PUT /api/v1/sender-identity", a.Require(http.HandlerFunc(camp.PutIdentity)))
	mux.Handle("GET /api/v1/campaigns", a.Require(http.HandlerFunc(camp.List)))
	mux.Handle("POST /api/v1/campaigns", a.Require(http.HandlerFunc(camp.Create)))
	mux.Handle("GET /api/v1/campaigns/{id}", a.Require(http.HandlerFunc(camp.Get)))
	mux.Handle("POST /api/v1/campaigns/{id}/start", a.Require(http.HandlerFunc(camp.Start)))
	mux.Handle("POST /api/v1/campaigns/{id}/pause", a.Require(http.HandlerFunc(camp.Pause)))
	mux.Handle("POST /api/v1/campaigns/{id}/stop", a.Require(http.HandlerFunc(camp.Stop)))
	mux.Handle("POST /api/v1/campaigns/{id}/enrolments", a.Require(http.HandlerFunc(camp.Enrol)))
	mux.Handle("GET /api/v1/campaigns/{id}/enrolments", a.Require(http.HandlerFunc(camp.ListEnrolments)))

	logger.Info("api listening", "addr", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
