"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { EmptyState, PageHeader, Pill } from "@/components/page-header";
import type { Mailbox, Thread, ThreadMessage } from "@/lib/api";
import { fieldSelect, readableMail } from "@/lib/labels";

const classLabel: Record<string, string> = {
  interested: "Interested",
  interview: "Interview",
  question: "Question",
  not_now: "Not now",
  rejection: "Not a fit",
  referral: "Referral",
  out_of_office: "Out of office",
  unsubscribe: "Unsubscribed",
  complaint: "Complaint",
  dsn: "Bounced",
};

function messageClass(message?: ThreadMessage): string {
  const raw = message?.classification?.class;
  return typeof raw === "string" ? raw : "";
}

function threadClass(thread: Thread): string {
  const inbound = [...(thread.messages ?? [])]
    .reverse()
    .find((m) => m.direction === "in");
  return messageClass(inbound ?? thread.messages?.at(-1));
}

function displayName(raw: string): string {
  const trimmed = raw.trim();
  const named = trimmed.match(/^(.*?)\s*<([^>]+)>\s*$/);
  if (!named) {
    return trimmed;
  }
  const name = named[1].replace(/^"|"$/g, "").trim();
  return name || named[2];
}

function shortWhen(iso?: string): string {
  if (!iso) {
    return "";
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  const now = new Date();
  if (d.toDateString() === now.toDateString()) {
    return new Intl.DateTimeFormat(undefined, {
      hour: "numeric",
      minute: "2-digit",
    }).format(d);
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
  }).format(d);
}

export function Inbox({
  mailboxes,
  mailboxID,
  threads,
  threadID,
}: {
  mailboxes: Mailbox[];
  mailboxID: string;
  threads: Thread[];
  threadID: string;
}) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);
  const [q, setQ] = useState("");
  const [filter, setFilter] = useState("all");
  const selected = threads.find((t) => t.id === threadID);
  const unreadCount = threads.filter((t) => t.unread).length;
  const classes = useMemo(() => {
    const seen = new Set<string>();
    for (const t of threads) {
      const c = threadClass(t);
      if (c && classLabel[c]) {
        seen.add(c);
      }
    }
    return [...seen];
  }, [threads]);
  const shown = threads.filter((t) => {
    if (filter === "unread" && !t.unread) {
      return false;
    }
    if (filter !== "all" && filter !== "unread" && threadClass(t) !== filter) {
      return false;
    }
    const last = (t.messages ?? []).at(-1);
    const hay = [t.subject, last?.from, readableMail(last?.body_text)]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    const needle = q.trim().toLowerCase();
    return !needle || hay.includes(needle);
  });

  function href(nextMailbox: string, nextThread: string) {
    const q = new URLSearchParams();
    if (nextMailbox) {
      q.set("mailbox", nextMailbox);
    }
    if (nextThread) {
      q.set("thread", nextThread);
    }
    return `/inbox?${q.toString()}`;
  }

  async function onSync() {
    if (!mailboxID) {
      return;
    }
    setMsg("");
    setPending(true);
    const res = await fetch(`/api/v1/mailboxes/${mailboxID}/sync`, {
      method: "POST",
      credentials: "include",
    });
    setPending(false);
    if (!res.ok) {
      setMsg("Sync failed");
      return;
    }
    router.refresh();
  }

  async function onOpen(t: Thread) {
    if (!t.unread) {
      return;
    }
    const res = await fetch(`/api/v1/threads/${t.id}/read`, {
      method: "POST",
      credentials: "include",
    });
    if (res.ok) {
      router.refresh();
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Inbox" />
      <div className="min-h-0 flex-1 overflow-y-auto bg-background p-[var(--margin-desktop)]">
        {mailboxes.length === 0 ? (
          <EmptyState
            title="Connect a mailbox first"
            description="Inbox shows conversations from a connected mailbox."
            action={
              <Link
                href="/mailboxes"
                className="inline-flex h-8 items-center rounded-md bg-primary px-2.5 font-label-md text-sm text-primary-foreground hover:bg-primary/80"
              >
                Mailboxes
              </Link>
            }
          />
        ) : (
          <div className="flex flex-col gap-4">
            {msg ? (
              <p role="alert" className="font-body-md text-destructive">
                {msg}
              </p>
            ) : null}
            <div className="flex flex-wrap items-center gap-3 rounded-md border border-border bg-card p-3">
              <label className="sr-only" htmlFor="inbox-mailbox">
                Mailbox
              </label>
              <select
                id="inbox-mailbox"
                className={`${fieldSelect} !w-64 shrink-0`}
                value={mailboxID}
                onChange={(e) => {
                  setFilter("all");
                  setQ("");
                  router.push(href(e.target.value, ""));
                }}
              >
                {mailboxes.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.email}
                  </option>
                ))}
              </select>
              <div className="flex flex-wrap items-center gap-1.5">
                {(
                  [
                    ["all", "All"],
                    ["unread", `Unread (${unreadCount})`],
                    ...classes.map((c) => [c, classLabel[c] ?? c] as const),
                  ] as const
                ).map(([id, label]) => (
                  <button
                    key={id}
                    type="button"
                    aria-pressed={filter === id}
                    onClick={() => setFilter(id)}
                    className={
                      filter === id
                        ? "inline-flex items-center rounded-full border border-primary bg-accent px-3 py-1 font-label-md text-primary"
                        : "inline-flex items-center rounded-full border border-border bg-card px-3 py-1 font-label-md text-muted-foreground hover:bg-muted hover:text-foreground"
                    }
                  >
                    {label}
                  </button>
                ))}
              </div>
              <input
                aria-label="Search conversations"
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder="Search sender or subject"
                className={`${fieldSelect} !w-56 shrink-0`}
              />
              <Button
                type="button"
                variant="outline"
                className="ml-auto rounded-md"
                pending={pending}
                disabled={!mailboxID}
                onClick={onSync}
              >
                Sync
              </Button>
            </div>
            <div className="grid items-start gap-4 lg:grid-cols-12">
              <div className="overflow-hidden rounded-md border border-border bg-card lg:col-span-5">
                <div className="flex items-center justify-between bg-muted px-4 py-2 font-label-sm text-muted-foreground">
                  <span>Conversations ({shown.length})</span>
                  <span className="tabular-nums">{unreadCount} unread</span>
                </div>
                {shown.length === 0 ? (
                  <div className="p-4">
                    <EmptyState
                      compact
                      title={
                        threads.length === 0
                          ? "No conversations"
                          : "Nothing in this filter"
                      }
                      description={
                        threads.length === 0
                          ? "Sync this mailbox to pull mail."
                          : "Clear search or pick All."
                      }
                      action={
                        threads.length === 0 ? (
                          <Button
                            type="button"
                            className="rounded-md"
                            pending={pending}
                            disabled={!mailboxID}
                            onClick={onSync}
                          >
                            Sync
                          </Button>
                        ) : (
                          <button
                            type="button"
                            className="font-label-md text-primary hover:underline"
                            onClick={() => {
                              setFilter("all");
                              setQ("");
                            }}
                          >
                            Clear filters
                          </button>
                        )
                      }
                    />
                  </div>
                ) : (
                  <ul className="divide-y divide-border">
                    {shown.map((t) => {
                      const active = t.id === threadID;
                      const last = (t.messages ?? []).at(-1);
                      const who = displayName(last?.from || "") || "Unknown";
                      const preview = readableMail(last?.body_text);
                      const kind = threadClass(t);
                      return (
                        <li key={t.id}>
                          <Link
                            href={href(mailboxID, t.id)}
                            aria-current={active ? "page" : undefined}
                            className={
                              active
                                ? "relative block border-l-[3px] border-primary bg-accent px-4 py-3 text-left"
                                : "relative block border-l-[3px] border-transparent px-4 py-3 text-left hover:bg-muted"
                            }
                            onClick={() => {
                              void onOpen(t);
                            }}
                          >
                            <div className="flex items-baseline justify-between gap-2">
                              <p className="flex min-w-0 items-center gap-2">
                                {t.unread ? (
                                  <span className="size-2 shrink-0 rounded-full bg-primary" />
                                ) : null}
                                <span
                                  className={
                                    t.unread
                                      ? "truncate font-label-md text-foreground"
                                      : "truncate font-body-md text-muted-foreground"
                                  }
                                >
                                  {who}
                                </span>
                              </p>
                              <When iso={t.last_message_at} />
                            </div>
                            <p
                              className={
                                t.unread
                                  ? "mt-1 truncate font-label-md text-foreground"
                                  : "mt-1 truncate font-body-md text-foreground"
                              }
                            >
                              {readableMail(t.subject) || "No subject"}
                            </p>
                            {preview ? (
                              <p className="mt-1 line-clamp-2 font-body-sm text-muted-foreground">
                                {preview}
                              </p>
                            ) : null}
                            {kind && classLabel[kind] ? (
                              <div className="mt-2">
                                <Pill
                                  tone={
                                    kind === "dsn" || kind === "complaint"
                                      ? "danger"
                                      : kind === "interview" ||
                                          kind === "interested"
                                        ? "success"
                                        : "muted"
                                  }
                                >
                                  {classLabel[kind]}
                                </Pill>
                              </div>
                            ) : null}
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </div>
              <div className="rounded-md border border-border bg-card p-5 lg:col-span-7">
                {!selected ? (
                  <EmptyState
                    compact
                    title="Pick a conversation"
                    description="Open one on the left to read it."
                  />
                ) : (
                  <div className="flex flex-col gap-4">
                    <div>
                      <h2 className="font-headline-sm text-foreground">
                        {readableMail(selected.subject) || "No subject"}
                      </h2>
                      {classLabel[threadClass(selected)] ? (
                        <div className="mt-2">
                          <Pill tone="muted">
                            {classLabel[threadClass(selected)]}
                          </Pill>
                        </div>
                      ) : null}
                    </div>
                    {(selected.messages ?? []).map((m) => (
                      <MessageCard key={m.id} message={m} />
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function When({ iso }: { iso?: string }) {
  const [text, setText] = useState("");
  useEffect(() => {
    setText(shortWhen(iso));
  }, [iso]);
  return (
    <p className="shrink-0 font-body-sm tabular-nums text-muted-foreground">
      {text}
    </p>
  );
}

function MessageCard({ message }: { message: ThreadMessage }) {
  const [open, setOpen] = useState(false);
  const to = message.to.filter(Boolean).join(", ");
  return (
    <article className="border-t border-border pt-4 first:border-t-0 first:pt-0">
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-baseline justify-between gap-3 rounded-md text-left hover:bg-muted/60"
      >
        <span className="min-w-0">
          <span className="font-label-md text-foreground">
            {displayName(message.from) || "Unknown"}
          </span>
          <span className="ml-2 font-body-sm text-muted-foreground">
            {open ? "Hide details" : to ? `to ${to}` : "Show details"}
          </span>
        </span>
      </button>
      {open ? (
        <dl className="mt-2 grid grid-cols-[4.5rem_1fr] gap-x-3 gap-y-1 font-body-sm">
          <dt className="text-muted-foreground">From</dt>
          <dd className="text-foreground">{message.from || "—"}</dd>
          <dt className="text-muted-foreground">To</dt>
          <dd className="text-foreground">{to || "—"}</dd>
        </dl>
      ) : null}
      {message.body_html ? (
        <iframe
          title="Message"
          sandbox=""
          srcDoc={message.body_html}
          className="mt-3 h-[420px] w-full rounded-md border border-border bg-card"
        />
      ) : (
        <p className="mt-3 max-w-[68ch] whitespace-pre-wrap font-body-md text-foreground">
          {readableMail(message.body_text)}
        </p>
      )}
    </article>
  );
}
