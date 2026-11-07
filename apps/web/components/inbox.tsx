"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import type { Mailbox, Thread } from "@/lib/api";

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
  const selected = threads.find((t) => t.id === threadID);

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

  if (mailboxes.length === 0) {
    return (
      <p className="font-body-md text-muted-foreground">
        Connect a mailbox first.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <label className="font-label-md text-muted-foreground" htmlFor="inbox-mailbox">
          Mailbox
        </label>
        <select
          id="inbox-mailbox"
          className="rounded-md border border-input bg-card px-2 py-1.5 font-body-md text-foreground"
          value={mailboxID}
          onChange={(e) => {
            window.location.assign(href(e.target.value, ""));
          }}
        >
          {mailboxes.map((m) => (
            <option key={m.id} value={m.id}>
              {m.email}
            </option>
          ))}
        </select>
        <Button
          type="button"
          variant="outline"
          className="rounded-md"
          disabled={pending || !mailboxID}
          onClick={onSync}
        >
          {pending ? "Syncing…" : "Sync"}
        </Button>
        {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      </div>
      <div className="grid min-h-[24rem] grid-cols-1 gap-4 md:grid-cols-[minmax(16rem,20rem)_1fr]">
        <ul className="flex flex-col gap-1 rounded-md border border-border bg-card p-2">
          {threads.length === 0 ? (
            <li className="p-3 font-body-md text-muted-foreground">
              No threads. Sync to pull Mailpit.
            </li>
          ) : (
            threads.map((t) => {
              const active = t.id === threadID;
              return (
                <li key={t.id}>
                  <Link
                    href={href(mailboxID, t.id)}
                    className={
                      active
                        ? "block w-full rounded-md bg-accent px-3 py-2 text-left"
                        : "block w-full rounded-md px-3 py-2 text-left hover:bg-muted"
                    }
                    onClick={() => {
                      void onOpen(t);
                    }}
                  >
                    <p
                      className={
                        t.unread
                          ? "font-label-md text-foreground"
                          : "font-body-md text-foreground"
                      }
                    >
                      {t.subject || "(no subject)"}
                    </p>
                    <p className="mt-0.5 font-body-md text-muted-foreground">
                      {t.unread ? "Unread" : "Read"}
                    </p>
                  </Link>
                </li>
              );
            })
          )}
        </ul>
        <div className="rounded-md border border-border bg-card p-4">
          {!selected ? (
            <p className="font-body-md text-muted-foreground">
              Open a thread to read it.
            </p>
          ) : (
            <div className="flex flex-col gap-4">
              <h2 className="font-headline-md text-foreground">
                {selected.subject || "(no subject)"}
              </h2>
              {(selected.messages ?? []).map((m) => (
                <article
                  key={m.id}
                  className="border-t border-border pt-3 first:border-t-0 first:pt-0"
                >
                  <p className="font-label-md text-foreground">
                    {m.direction === "in" ? "From" : "To"} {m.from || m.to.join(", ")}
                  </p>
                  {m.classification?.class ? (
                    <p className="mt-1 font-body-md text-muted-foreground">
                      {m.classification.class}
                      {m.classification.reply_draft ? " · reply draft" : ""}
                    </p>
                  ) : null}
                  <p className="mt-2 whitespace-pre-wrap font-body-md text-foreground">
                    {m.body_text}
                  </p>
                </article>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
