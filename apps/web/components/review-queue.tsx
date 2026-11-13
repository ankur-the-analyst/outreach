"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { SquarePen } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { OneOffDraftForm } from "@/components/one-off-draft-form";
import { EmptyState, PageBody, PageHeader, Pill } from "@/components/page-header";

import type { Draft, Mailbox, Profile } from "@/lib/api";
import {
  fieldTextarea,
  mailboxIdle,
  reviewBucket,
  sendStatus,
  type ReviewFilter,
} from "@/lib/labels";

function reviewActionError(message: string): string {
  switch (message) {
    case "invalid draft transition":
      return "This draft is no longer waiting for approval";
    case "Fix leftover placeholders before sending":
      return "This email still has placeholders. Fix them before sending.";
    default:
      return message;
  }
}

function hasLeftover(draft: Draft, subject: string, body: string): boolean {
  return [subject, body, draft.body_html ?? ""].some((s) => s.includes("{{"));
}

export function ReviewQueue({
  drafts,
  nextCursor,
  mailboxes = [],
  profiles = [],
  filters = false,
}: {
  drafts: Draft[];
  nextCursor?: string;
  mailboxes?: Mailbox[];
  profiles?: Profile[];
  filters?: boolean;
}) {
  const router = useRouter();
  const [filter, setFilter] = useState<ReviewFilter>("pending");
  const [compose, setCompose] = useState(false);
  const [rows, setRows] = useState(drafts);
  const [cursor, setCursor] = useState(nextCursor);
  const [morePending, setMorePending] = useState(false);
  const [nowMs, setNowMs] = useState<number>();
  useEffect(() => {
    setRows(drafts);
    setCursor(nextCursor);
  }, [drafts, nextCursor]);
  useEffect(() => {
    setNowMs(Date.now());
  }, []);
  const byMailbox = useMemo(
    () => new Map(mailboxes.map((m) => [m.id, m])),
    [mailboxes],
  );
  const counts = useMemo(() => {
    const c = { pending: 0, tomorrow: 0, undo: 0, checking: 0, sent: 0 };
    for (const d of rows) {
      const b = reviewBucket(d, byMailbox.get(d.mailbox_id));
      if (b !== "other") {
        c[b] += 1;
      }
    }
    return c;
  }, [rows, byMailbox]);

  const shown = filters
    ? rows.filter((d) => reviewBucket(d, byMailbox.get(d.mailbox_id)) === filter)
    : rows;

  async function loadMore() {
    if (!cursor) {
      return;
    }
    setMorePending(true);
    const res = await fetch(
      `/api/v1/drafts?cursor=${encodeURIComponent(cursor)}`,
      { credentials: "include" },
    );
    setMorePending(false);
    if (!res.ok) {
      return;
    }
    const data = (await res.json()) as {
      drafts?: Draft[];
      next_cursor?: string;
    };
    setRows((cur) => [...cur, ...(data.drafts ?? [])]);
    setCursor(data.next_cursor);
  }

  useEffect(() => {
    if (!filters) {
      return;
    }
    const es = new EventSource("/api/v1/events/stream", { withCredentials: true });
    let last = -1;
    es.addEventListener("review", (ev) => {
      const n = Number((JSON.parse(ev.data) as { n?: number }).n);
      if (last < 0) {
        last = n;
        return;
      }
      if (n !== last) {
        last = n;
        router.refresh();
      }
    });
    return () => es.close();
  }, [filters, router]);

  const list =
    shown.length === 0 ? (
      <EmptyState
        title={rows.length === 0 ? "No drafts in review" : "Nothing in this filter"}
        description={
          rows.length === 0
            ? mailboxes.length === 0
              ? "Connect a mailbox, then compose or run a campaign."
              : mailboxes.every((m) => mailboxIdle(m))
                ? "People stay queued until the mailbox is sending. Then drafts show up here."
                : "Approve, undo, and one-off sends show up here."
            : "Try another filter, or compose a one-off."
        }
        action={
          rows.length === 0 && mailboxes.length === 0 ? (
            <Link
              href="/mailboxes"
              className="inline-flex h-8 items-center rounded-md bg-primary px-2.5 font-label-md text-sm text-primary-foreground hover:bg-primary/80"
            >
              Mailboxes
            </Link>
          ) : filters ? (
            <Button
              type="button"
              className="rounded-md"
              onClick={() => setCompose(true)}
              disabled={mailboxes.length === 0}
            >
              Compose one-off
            </Button>
          ) : undefined
        }
      />
    ) : (
      <ul className="flex flex-col gap-3">
        {shown.map((d) => (
          <ReviewRow key={d.id} draft={d} mailboxes={mailboxes} nowMs={nowMs} />
        ))}
      </ul>
    );

  if (!filters) {
    return list;
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader
        title="Review"
        actions={
          <Button
            type="button"
            variant="outline"
            className="rounded-md"
            onClick={() => setCompose(true)}
            disabled={mailboxes.length === 0}
          >
            <SquarePen className="size-4" strokeWidth={1.5} aria-hidden="true" />
            Compose one-off
          </Button>
        }
      />
      <PageBody>
      <div className="mb-4 flex flex-wrap gap-1.5">
        {(
          [
            ["pending", "Needs approval"],
            ["tomorrow", "Waiting to send"],
            ["undo", "You can still undo"],
            ["checking", "Checking if it sent"],
            ["sent", "Sent"],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            onClick={() => setFilter(id)}
            aria-pressed={filter === id}
            className={
              filter === id
                ? "inline-flex items-center gap-1.5 rounded-full border border-primary bg-accent px-3 py-1.5 font-label-md text-primary"
                : "inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-3 py-1.5 font-label-md text-muted-foreground hover:bg-muted hover:text-foreground"
            }
          >
            {label}
            <span className="tabular-nums">{counts[id]}</span>
          </button>
        ))}
      </div>
      {list}
      {cursor ? (
        <Button
          type="button"
          variant="outline"
          className="mt-3 w-fit rounded-md"
          pending={morePending}
          onClick={loadMore}
        >
          Show more
        </Button>
      ) : null}
      </PageBody>
      <Dialog open={compose} onOpenChange={setCompose}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Compose one-off</DialogTitle>
          </DialogHeader>
          <OneOffDraftForm
            mailboxes={mailboxes}
            profiles={profiles}
            onSaved={() => setCompose(false)}
          />
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ReviewRow({
  draft,
  mailboxes,
  nowMs,
}: {
  draft: Draft;
  mailboxes: Mailbox[];
  nowMs?: number;
}) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);
  const [subject, setSubject] = useState(draft.subject);
  const [body, setBody] = useState(draft.body_text);
  const mailbox = mailboxes.find((m) => m.id === draft.mailbox_id);
  const bucket = reviewBucket(draft, mailbox);
  const accent = bucket === "undo" || bucket === "pending" || bucket === "tomorrow";

  async function post(path: string) {
    setMsg("");
    setPending(true);
    const res = await fetch(path, { method: "POST", credentials: "include" });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(reviewActionError(data?.error?.message ?? "Request failed"));
      return;
    }
    router.refresh();
  }

  async function save() {
    setMsg("");
    setPending(true);
    const res = await fetch(`/api/v1/drafts/${draft.id}`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ subject, text: body }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(reviewActionError(data?.error?.message ?? "Could not save"));
      return;
    }
    router.refresh();
  }

  async function rewrite() {
    setMsg("");
    setPending(true);
    const res = await fetch(`/api/v1/drafts/${draft.id}/regenerate`, {
      method: "POST",
      credentials: "include",
    });
    setPending(false);
    const data = (await res.json().catch(() => null)) as
      | {
          error?: { message?: string };
          draft?: { body_text?: string; subject?: string };
          failures?: { message?: string }[];
        }
      | null;
    if (!res.ok) {
      setMsg(reviewActionError(data?.error?.message ?? "Could not rewrite"));
      return;
    }
    if (data?.draft?.body_text) {
      setBody(data.draft.body_text);
    }
    if (data?.draft?.subject) {
      setSubject(data.draft.subject);
    }
    const fails = (data?.failures ?? [])
      .map((f) => f.message)
      .filter((m): m is string => Boolean(m));
    setMsg(fails.length > 0 ? fails.join(" · ") : "");
    router.refresh();
  }

  return (
    <li
      className={
        accent
          ? "relative overflow-hidden rounded-md border border-border bg-card p-5"
          : "rounded-md border border-border bg-card p-5"
      }
    >
      {accent ? (
        <span className="absolute top-0 bottom-0 left-0 w-[3px] bg-primary" />
      ) : null}
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          {draft.status === "pending" ? (
            <Input
              aria-label="Subject"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
            />
          ) : (
            <p className="font-headline-sm text-foreground">{draft.subject}</p>
          )}
          <p className="mt-1 font-body-sm text-muted-foreground">
            To {draft.to}
            {mailbox ? ` · via ${mailbox.email}` : ""}
          </p>
          {draft.attachment_name ? (
            <p className="mt-1 font-body-sm text-muted-foreground">
              File {draft.attachment_name}
              {draft.attachment_from ? ` · From ${draft.attachment_from}` : ""}
            </p>
          ) : null}
        </div>
        <Pill
          tone={bucket === "undo" || bucket === "pending" ? "primary" : "muted"}
        >
          {sendStatus(draft, mailbox, nowMs)}
        </Pill>
      </div>
      {(draft.facts_used ?? []).length > 0 ? (
        <p className="mt-2 font-body-sm text-muted-foreground">
          Claims: {(draft.facts_used ?? []).map((f) => f.claim).join(" · ")}
        </p>
      ) : null}
      {hasLeftover(draft, subject, body) ? (
        <p className="mt-2 font-body-sm text-muted-foreground">
          This email still has placeholders. Fix them before sending.
        </p>
      ) : null}
      {draft.body_html ? (
        <iframe
          title="Email preview"
          sandbox=""
          srcDoc={draft.body_html}
          className="mt-3 h-[280px] w-full max-w-[68ch] rounded-md border border-border bg-card"
        />
      ) : null}
      {draft.status === "pending" ? (
        <textarea
          aria-label={draft.body_html ? "Plain text" : "Body"}
          rows={draft.body_html ? 4 : 8}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          className={`mt-3 max-w-[68ch] ${fieldTextarea}`}
        />
      ) : draft.body_html ? null : (
        <p className="mt-3 max-w-[68ch] whitespace-pre-wrap font-body-md text-foreground">
          {body}
        </p>
      )}
      <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-border pt-3">
        {draft.status === "pending" ? (
          <>
            <Button
              type="button"
              variant="outline"
              className="rounded-md"
              pending={pending}
              onClick={() => save()}
            >
              Save
            </Button>
            <Button
              type="button"
              className="rounded-md"
              pending={pending}
              onClick={() => post(`/api/v1/drafts/${draft.id}/approve`)}
            >
              Approve
            </Button>
            <Button
              type="button"
              variant="outline"
              className="rounded-md"
              pending={pending}
              onClick={() => rewrite()}
            >
              Rewrite
            </Button>
            <Button
              type="button"
              variant="outline"
              className="rounded-md border-destructive text-destructive hover:bg-destructive/10"
              pending={pending}
              onClick={() => post(`/api/v1/drafts/${draft.id}/reject`)}
            >
              Don&apos;t send
            </Button>
          </>
        ) : null}
        {draft.status === "approved" && draft.ledger_state === "held" ? (
          <Button
            type="button"
            variant="outline"
            className="rounded-md border-destructive text-destructive hover:bg-destructive/10"
            pending={pending}
            onClick={() => post(`/api/v1/drafts/${draft.id}/undo`)}
          >
            Undo send
          </Button>
        ) : null}
        {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      </div>
    </li>
  );
}

