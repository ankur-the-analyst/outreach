package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
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
	"outreach/internal/cv"
	"outreach/internal/db"
	"outreach/internal/events"
	"outreach/internal/evidence"
	"outreach/internal/fields"
	"outreach/internal/health"
	"outreach/internal/inbound"
	"outreach/internal/jobsjd"
	"outreach/internal/mailbox"
	"outreach/internal/mailbox/gmail"
	"outreach/internal/mailbox/smtpimap"
	"outreach/internal/sender"
	"outreach/internal/storage"
	"outreach/internal/suppression"
	mailsync "outreach/internal/sync"
	"outreach/internal/templates"
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

type syncJobs struct {
	client *river.Client[pgx.Tx]
}

func (j syncJobs) EnqueueSync(ctx context.Context, workspaceID, mailboxID uuid.UUID) error {
	return worker.EnqueueSync(ctx, j.client, workspaceID, mailboxID)
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
	reg := mailbox.NewRegistry()
	reg.Register("smtp_imap", "", smtpimap.Factory)
	reg.Register("app_password", "", smtpimap.Factory)
	reg.Register("oauth", "gmail", gmail.NewFactory(gmail.Config{
		ClientID:     cfg.GoogleOAuthClientID,
		ClientSecret: cfg.GoogleOAuthClientSecret,
	}))
	dialer := mailbox.NewDialer(pool, keys, reg)
	jobs, err := worker.NewInsertClient(pool)
	if err != nil {
		logger.Error("river client", "err", err)
		os.Exit(1)
	}
	mb := mailbox.New(mailbox.NewPostgres(pool, keys), smtpimap.Probe{}).
		WithOAuth(mailbox.OAuthConfig{
			ClientID:     cfg.GoogleOAuthClientID,
			ClientSecret: cfg.GoogleOAuthClientSecret,
			RedirectURI:  cfg.GoogleOAuthRedirectURI,
			StateKey:     cfg.OAuthStateKey(),
			SuccessPath:  strings.TrimRight(cfg.PublicAppURL, "/") + "/mailboxes",
		}).
		WithDialer(dialer).
		WithSync(syncJobs{client: jobs})
	wrk := worker.New(pool, wsStore, jobs)
	hold := time.Duration(cfg.UndoHoldS) * time.Second
	unknown := time.Duration(cfg.UnknownTimeoutS) * time.Second
	suppStore := suppression.NewPostgres(pool, keys)
	supp := suppression.New(suppStore, keys)
	blobs, err := storage.Open(cfg.StorageDriver, cfg.StoragePath)
	if err != nil {
		logger.Error("open storage", "err", err)
		os.Exit(1)
	}
	cvStore := cv.NewPostgres(pool, blobs)
	led := sender.NewPostgres(pool, time.Now, hold).
		WithDialer(dialer).
		WithUnknownTimeout(unknown).
		WithQueue(reconcileJobs{client: jobs}).
		WithBlocker(suppStore).
		WithUnsub(cfg.PublicBaseURL, keys).
		WithAttach(cvStore)
	snd := sender.New(led)
	writer := &ai.Fake{
		Model: "fake-8", Dim: evidence.Dim,
		Text: "You closed 12 deals last year. Open to a short chat?",
		JSON: json.RawMessage(`{"score":82,"summary":"deals overlap"}`),
	}
	evStore := evidence.NewPostgres(pool, writer)
	ev := evidence.New(evStore)
	aiStore := ai.NewPostgres(pool, keys)
	aiHTTP := ai.NewAPI(aiStore)
	cvStore.WithBinder(aiStore)
	cmp := compose.New(compose.NewPostgres(pool, led).WithBinder(aiStore).WithCite(evStore).WithRetrieve(evStore))
	proc := inbound.New(pool, suppStore, classifyJobs{client: jobs})
	sy := mailsync.New(mailsync.NewPostgres(pool, mailsync.Mailpit{Base: cfg.MailpitAPI}).WithInbound(proc).WithOpener(dialer))
	jd := jobsjd.New(jobsjd.NewPostgres(pool, blobs, writer).WithBinder(aiStore))
	cvHTTP := cv.New(cvStore)
	fieldStore := fields.NewPostgres(pool)
	fd := fields.New(fieldStore)
	tpl := templates.New(templates.NewPostgres(pool))
	ct := contacts.New(contacts.NewPostgres(pool).WithHMAC(keys).WithCatalog(fieldStore)).WithCatalog(fieldStore)
	packs, err := campaign.LoadPacks()
	if err != nil {
		logger.Error("load packs", "err", err)
		os.Exit(1)
	}
	camp := campaign.New(campaign.NewPostgres(pool, packs).WithBinder(aiStore).WithRetrieve(evStore).WithStamp(cvStore))

	mux := http.NewServeMux()
	hs := health.New(health.NewPostgres(pool))
	mux.HandleFunc("GET /health", health.Handler(pool))
	mux.HandleFunc("POST /u/{token}", supp.OneClick)
	mux.HandleFunc("GET /u/{token}", supp.OneClick)
	mux.HandleFunc("POST /api/v1/unsubscribe/{token}", supp.OneClick)
	mux.HandleFunc("GET /api/v1/unsubscribe/{token}", supp.OneClick)
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
	evs := events.New(events.NewPostgres(pool))
	mux.Handle("GET /api/v1/events/stream", a.Require(http.HandlerFunc(evs.Stream)))
	mux.Handle("GET /api/v1/health-samples", a.Require(http.HandlerFunc(hs.List)))
	mux.Handle("GET /api/v1/warmup-profiles", a.Require(http.HandlerFunc(mb.ListWarmups)))
	mux.Handle("POST /api/v1/warmup-profiles", a.Require(http.HandlerFunc(mb.CreateWarmup)))
	mux.Handle("GET /api/v1/mailbox-pools", a.Require(http.HandlerFunc(mb.ListPools)))
	mux.Handle("POST /api/v1/mailbox-pools", a.Require(http.HandlerFunc(mb.CreatePool)))
	mux.Handle("GET /api/v1/mailboxes", a.Require(http.HandlerFunc(mb.List)))
	mux.Handle("GET /api/v1/sending-domains", a.Require(http.HandlerFunc(mb.ListDomains)))
	mux.Handle("GET /api/v1/mailboxes/oauth/google", a.Require(http.HandlerFunc(mb.OAuthEnabled)))
	mux.Handle("GET /api/v1/mailboxes/oauth/google/start", a.Require(http.HandlerFunc(mb.OAuthStart)))
	mux.HandleFunc("GET /api/v1/mailboxes/oauth/callback", mb.OAuthCallback)
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
	mux.Handle("GET /api/v1/suppression", a.Require(http.HandlerFunc(supp.List)))
	mux.Handle("GET /api/v1/drafts", a.Require(http.HandlerFunc(cmp.List)))
	mux.Handle("POST /api/v1/drafts", a.Require(http.HandlerFunc(cmp.Create)))
	mux.Handle("PATCH /api/v1/drafts/{id}", a.Require(http.HandlerFunc(cmp.Patch)))
	mux.Handle("POST /api/v1/drafts/compose", a.Require(http.HandlerFunc(cmp.Compose)))
	mux.Handle("POST /api/v1/drafts/{id}/approve", a.Require(http.HandlerFunc(cmp.Approve)))
	mux.Handle("POST /api/v1/drafts/{id}/reject", a.Require(http.HandlerFunc(cmp.Reject)))
	mux.Handle("POST /api/v1/drafts/{id}/regenerate", a.Require(http.HandlerFunc(cmp.Regenerate)))
	mux.Handle("POST /api/v1/drafts/{id}/undo", a.Require(http.HandlerFunc(cmp.Undo)))
	mux.Handle("POST /api/v1/mailboxes/{id}/sync", a.Require(http.HandlerFunc(sy.Sync)))
	mux.Handle("GET /api/v1/mailboxes/{id}/threads", a.Require(http.HandlerFunc(sy.Threads)))
	mux.Handle("POST /api/v1/threads/{id}/read", a.Require(http.HandlerFunc(sy.MarkRead)))
	mux.Handle("GET /api/v1/evidence", a.Require(http.HandlerFunc(ev.List)))
	mux.Handle("GET /api/v1/evidence/usable", a.Require(http.HandlerFunc(ev.Usable)))
	mux.Handle("POST /api/v1/evidence", a.Require(http.HandlerFunc(ev.Propose)))
	mux.Handle("POST /api/v1/evidence/{id}/confirm", a.Require(http.HandlerFunc(ev.Confirm)))
	mux.Handle("GET /api/v1/profiles", a.Require(http.HandlerFunc(jd.ListProfiles)))
	mux.Handle("POST /api/v1/profiles", a.Require(http.HandlerFunc(jd.CreateProfile)))
	mux.Handle("POST /api/v1/profiles/{id}/resumes", a.Require(http.HandlerFunc(jd.UploadResume)))
	mux.Handle("POST /api/v1/profiles/{id}/cv", a.Require(http.HandlerFunc(cvHTTP.Save)))
	mux.Handle("GET /api/v1/jobs", a.Require(http.HandlerFunc(jd.ListJobs)))
	mux.Handle("POST /api/v1/jobs", a.Require(http.HandlerFunc(jd.CreateJob)))
	mux.Handle("POST /api/v1/jobs/{id}/fit", a.Require(http.HandlerFunc(jd.Fit)))
	mux.Handle("GET /api/v1/fits", a.Require(http.HandlerFunc(jd.ListFits)))
	mux.Handle("GET /api/v1/fields", a.Require(http.HandlerFunc(fd.List)))
	mux.Handle("POST /api/v1/fields", a.Require(http.HandlerFunc(fd.Create)))
	mux.Handle("GET /api/v1/templates", a.Require(http.HandlerFunc(tpl.List)))
	mux.Handle("POST /api/v1/templates", a.Require(http.HandlerFunc(tpl.Create)))
	mux.Handle("PUT /api/v1/templates/{id}", a.Require(http.HandlerFunc(tpl.Put)))
	mux.Handle("GET /api/v1/contacts", a.Require(http.HandlerFunc(ct.List)))
	mux.Handle("POST /api/v1/contacts", a.Require(http.HandlerFunc(ct.Create)))
	mux.Handle("POST /api/v1/contacts/preview", a.Require(http.HandlerFunc(ct.Preview)))
	mux.Handle("POST /api/v1/contacts/import", a.Require(http.HandlerFunc(ct.Import)))
	mux.Handle("GET /api/v1/contacts/{id}", a.Require(http.HandlerFunc(ct.Get)))
	mux.Handle("PATCH /api/v1/contacts/{id}", a.Require(http.HandlerFunc(ct.Patch)))
	mux.Handle("DELETE /api/v1/contacts/{id}", a.Require(http.HandlerFunc(ct.Delete)))
	mux.Handle("POST /api/v1/contacts/{id}/erase", a.Require(http.HandlerFunc(ct.Erase)))
	mux.Handle("GET /api/v1/packs", a.Require(http.HandlerFunc(camp.ListPacks)))
	mux.Handle("GET /api/v1/sender-identity", a.Require(http.HandlerFunc(camp.GetIdentity)))
	mux.Handle("PUT /api/v1/sender-identity", a.Require(http.HandlerFunc(camp.PutIdentity)))
	mux.Handle("GET /api/v1/settings/ai", a.Require(http.HandlerFunc(aiHTTP.Get)))
	mux.Handle("PUT /api/v1/settings/ai", a.Require(http.HandlerFunc(aiHTTP.Put)))
	mux.Handle("GET /api/v1/campaigns", a.Require(http.HandlerFunc(camp.List)))
	mux.Handle("POST /api/v1/campaigns", a.Require(http.HandlerFunc(camp.Create)))
	mux.Handle("GET /api/v1/campaigns/{id}", a.Require(http.HandlerFunc(camp.Get)))
	mux.Handle("POST /api/v1/campaigns/{id}/start", a.Require(http.HandlerFunc(camp.Start)))
	mux.Handle("POST /api/v1/campaigns/{id}/pause", a.Require(http.HandlerFunc(camp.Pause)))
	mux.Handle("POST /api/v1/campaigns/{id}/stop", a.Require(http.HandlerFunc(camp.Stop)))
	mux.Handle("POST /api/v1/campaigns/{id}/enrolments", a.Require(http.HandlerFunc(camp.Enrol)))
	mux.Handle("GET /api/v1/campaigns/{id}/enrolments", a.Require(http.HandlerFunc(camp.ListEnrolments)))
	mux.Handle("GET /api/v1/tracker", a.Require(http.HandlerFunc(camp.ListTracker)))
	mux.Handle("PATCH /api/v1/applications/{id}", a.Require(http.HandlerFunc(camp.SetStage)))

	logger.Info("api listening", "addr", cfg.Addr, "google_oauth", cfg.GoogleOAuthClientID != "")
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
