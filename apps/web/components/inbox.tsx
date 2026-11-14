"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { EmptyState, PageHeader } from "@/components/page-header";
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
  const sameYear = d.getFullYear() === now.getFullYear();
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    year: sameYear ? undefined : "numeric",
    hour: "numeric",
    minute: "2-digit",
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
  const [rows, setRows] = useState(threads);
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [q, setQ] = useState("");
  const [filter, setFilter] = useState("all");
  const [openThread, setOpenThread] = useState<Thread | null>(null);
  const [loadingMail, setLoadingMail] = useState(Boolean(threadID));
  const [mailError, setMailError] = useState("");
  const unreadCount = rows.filter((t) => t.unread).length;
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
  const shown = rows.filter((t) => {
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

  useEffect(() => {
    setRows(threads);
  }, [threads]);

  useEffect(() => {
    if (!threadID) {
      setOpenThread(null);
      setLoadingMail(false);
      setMailError("");
      return;
    }
    let cancel = false;
    setLoadingMail(true);
    setMailError("");
    setOpenThread(null);
    void (async () => {
      const res = await fetch(`/api/v1/threads/${threadID}`, { credentials: "include" });
      if (cancel) {
        return;
      }
      setLoadingMail(false);
      if (!res.ok) {
        if (res.status === 404) {
          const known = threads.find((t) => t.id === threadID);
          if (known) {
            setOpenThread(known);
            return;
          }
        }
        setMailError("Could not open this email");
        return;
      }
      setOpenThread((await res.json()) as Thread);
    })();
    return () => {
      cancel = true;
    };
  }, [threadID, threads]);

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
    const params = new URLSearchParams();
    if (from) {
      params.set("from", from);
    }
    if (to) {
      params.set("to", to);
    }
    const qs = params.toString();
    const res = await fetch(
      `/api/v1/mailboxes/${mailboxID}/sync${qs ? `?${qs}` : ""}`,
      {
        method: "POST",
        credentials: "include",
      },
    );
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
    setRows((cur) => cur.map((row) => (row.id === t.id ? { ...row, unread: false } : row)));
    await fetch(`/api/v1/threads/${t.id}/read`, {
      method: "POST",
      credentials: "include",
    });
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Inbox" />
      <div
        className={
          threadID
            ? "flex min-h-0 flex-1 flex-col overflow-hidden bg-background p-[var(--margin-desktop)]"
            : "min-h-0 flex-1 overflow-y-auto bg-background p-[var(--margin-desktop)]"
        }
      >
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
          <div className={threadID ? "flex min-h-0 flex-1 flex-col gap-4" : "flex flex-col gap-4"}>
            {msg ? (
              <p role="alert" className="font-body-md text-destructive">
                {msg}
              </p>
            ) : null}
            {threadID ? null : (
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
              <label className="flex items-center gap-1.5 font-label-sm text-muted-foreground">
                From
                <input
                  type="date"
                  aria-label="From date"
                  value={from}
                  onChange={(e) => setFrom(e.target.value)}
                  className={`${fieldSelect} !w-auto`}
                />
              </label>
              <label className="flex items-center gap-1.5 font-label-sm text-muted-foreground">
                To
                <input
                  type="date"
                  aria-label="To date"
                  value={to}
                  onChange={(e) => setTo(e.target.value)}
                  className={`${fieldSelect} !w-auto`}
                />
              </label>
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
            )}
            {threadID ? (
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border border-border bg-card">
                <div className="flex shrink-0 items-center gap-3 border-b border-border px-4 py-3">
                  <Link
                    href={href(mailboxID, "")}
                    className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 font-label-md text-foreground hover:bg-muted"
                  >
                    <ArrowLeft className="size-4" />
                    Back
                  </Link>
                </div>
                {loadingMail ? (
                  <div className="flex h-64 items-center justify-center">
                    <Spinner className="size-6 text-muted-foreground" />
                  </div>
                ) : mailError ? (
                  <p className="p-6 font-body-md text-destructive" role="alert">
                    {mailError}
                  </p>
                ) : (
                  <div className="min-h-0 flex-1 overflow-y-auto p-5">
                    <h2 className="font-headline-sm text-foreground">
                      {readableMail(openThread?.subject) || "No subject"}
                    </h2>
                    {(openThread?.messages ?? []).map((m) => (
                      <MessageCard key={m.id} message={m} />
                    ))}
                  </div>
                )}
              </div>
            ) : (
            <div className="overflow-hidden rounded-md border border-border bg-card">
                <div className="flex items-center justify-between bg-muted px-4 py-2 font-label-sm text-muted-foreground">
                  <span>{shown.length}</span>
                  <span className="tabular-nums">{unreadCount} unread</span>
                </div>
                {shown.length === 0 ? (
                  <div className="p-4">
                    <EmptyState
                      compact
                      title={
                        rows.length === 0
                          ? "No conversations"
                          : "Nothing in this filter"
                      }
                      description={
                        rows.length === 0
                          ? "Sync loads the last 2 days for a new mailbox. Pick dates to load that range."
                          : "Clear search or pick All."
                      }
                      action={
                        rows.length === 0 ? (
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
                      const last = (t.messages ?? []).at(-1);
                      const who = displayName(last?.from || "") || "Unknown";
                      const preview = readableMail(last?.body_text);
                      const subject = readableMail(t.subject) || "No subject";
                      return (
                        <li key={t.id}>
                          <Link
                            href={href(mailboxID, t.id)}
                            className="flex items-center gap-4 px-4 py-2.5 text-left hover:bg-muted"
                            onClick={() => {
                              void onOpen(t);
                            }}
                          >
                            <span
                              className={
                                t.unread
                                  ? "w-44 shrink-0 truncate font-label-md text-foreground"
                                  : "w-44 shrink-0 truncate font-body-md text-muted-foreground"
                              }
                            >
                              {who}
                            </span>
                            <span className="min-w-0 flex-1 truncate">
                              <span
                                className={
                                  t.unread
                                    ? "font-label-md text-foreground"
                                    : "font-body-md text-foreground"
                                }
                              >
                                {subject}
                              </span>
                              {preview ? (
                                <span className="font-body-md text-muted-foreground">
                                  {" "}
                                  — {preview}
                                </span>
                              ) : null}
                            </span>
                            <When iso={t.last_message_at} />
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </div>
            )}
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
    <article className="mt-4 border-t border-border pt-4">
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
        <When iso={message.at} />
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
        <MailFrame html={message.body_html} />
      ) : (
        <p className="mt-3 max-w-[68ch] whitespace-pre-wrap font-body-md text-foreground">
          {readableMail(message.body_text)}
        </p>
      )}
    </article>
  );
}

function MailFrame({ html }: { html: string }) {
  const frame = useRef<HTMLIFrameElement>(null);
  function fit() {
    const doc = frame.current?.contentDocument;
    const height = doc?.documentElement.scrollHeight;
    if (frame.current && height) {
      frame.current.style.height = `${height}px`;
    }
  }
  return (
    <iframe
      ref={frame}
      title="Message"
      sandbox="allow-same-origin"
      scrolling="no"
      srcDoc={html}
      onLoad={fit}
      className="mt-3 w-full overflow-hidden rounded-md border border-border bg-card"
    />
  );
}
