"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { SquarePen, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { OneOffDraftForm } from "@/components/one-off-draft-form";
import { PageBody, PageHeader, Pill } from "@/components/page-header";

import type { Draft, Mailbox } from "@/lib/api";
import {
  reviewBucket,
  sendStatus,
  type ReviewFilter,
} from "@/lib/labels";

export function ReviewQueue({
  drafts,
  mailboxes = [],
  filters = false,
}: {
  drafts: Draft[];
  mailboxes?: Mailbox[];
  filters?: boolean;
}) {
  const [filter, setFilter] = useState<ReviewFilter>("pending");
  const [compose, setCompose] = useState(false);
  const counts = useMemo(() => {
    const c = { pending: 0, undo: 0, checking: 0, sent: 0 };
    for (const d of drafts) {
      const b = reviewBucket(d);
      if (b !== "other") {
        c[b] += 1;
      }
    }
    return c;
  }, [drafts]);

  const shown = filters
    ? drafts.filter((d) => reviewBucket(d) === filter)
    : drafts;

  const list =
    shown.length === 0 ? (
      <p className="font-body-md text-muted-foreground">
        {drafts.length === 0 ? "No drafts in review." : "Nothing in this filter."}
      </p>
    ) : (
      <ul className="flex flex-col gap-3">
        {shown.map((d) => (
          <ReviewRow key={d.id} draft={d} mailboxes={mailboxes} />
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
            <SquarePen className="size-4" strokeWidth={1.5} />
            Compose one-off
          </Button>
        }
      />
      <PageBody>
      <div className="mb-4 flex flex-wrap gap-1.5">
        {(
          [
            ["pending", "Needs approval"],
            ["undo", "You can still undo"],
            ["checking", "Checking if it sent"],
            ["sent", "Sent"],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            onClick={() => setFilter(id)}
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
      </PageBody>
      {compose ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4"
          onClick={() => setCompose(false)}
        >
          <div
            className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-md border border-border bg-card shadow-[var(--shadow-modal)]"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-border px-6 py-4">
              <h2 className="font-headline-md text-foreground">
                Compose one-off
              </h2>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => setCompose(false)}
              >
                <X className="size-4" />
              </Button>
            </div>
            <div className="p-6">
              <OneOffDraftForm
                mailboxes={mailboxes}
                onSaved={() => setCompose(false)}
              />
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function ReviewRow({
  draft,
  mailboxes,
}: {
  draft: Draft;
  mailboxes: Mailbox[];
}) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);
  const mailbox = mailboxes.find((m) => m.id === draft.mailbox_id);
  const bucket = reviewBucket(draft);
  const accent = bucket === "undo" || bucket === "pending";

  async function post(path: string) {
    setMsg("");
    setPending(true);
    const res = await fetch(path, { method: "POST", credentials: "include" });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(data?.error?.message ?? "Request failed");
      return;
    }
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
        <div>
          <p className="font-headline-sm text-foreground">{draft.subject}</p>
          <p className="mt-1 font-body-sm text-muted-foreground">
            To {draft.to}
            {mailbox ? ` · via ${mailbox.email}` : ""}
          </p>
        </div>
        <Pill
          tone={bucket === "undo" || bucket === "pending" ? "primary" : "muted"}
        >
          {sendStatus(draft)}
        </Pill>
      </div>
      <p className="mt-3 max-w-[68ch] whitespace-pre-wrap font-body-md text-foreground">
        {draft.body_text}
      </p>
      <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-border pt-3">
        {draft.status === "pending" ? (
          <Button
            type="button"
            className="rounded-md"
            disabled={pending}
            onClick={() => post(`/api/v1/drafts/${draft.id}/approve`)}
          >
            {pending ? "…" : "Approve"}
          </Button>
        ) : null}
        {draft.status === "approved" && draft.ledger_state === "held" ? (
          <Button
            type="button"
            variant="outline"
            className="rounded-md border-destructive text-destructive hover:bg-destructive/10"
            disabled={pending}
            onClick={() => post(`/api/v1/drafts/${draft.id}/undo`)}
          >
            {pending ? "…" : "Undo send"}
          </Button>
        ) : null}
        {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      </div>
    </li>
  );
}

