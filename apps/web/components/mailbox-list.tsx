"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Pill } from "@/components/page-header";
import type { Mailbox } from "@/lib/api";
import {
  authKindLabel,
  fieldSelect,
  mailboxStatus,
  mailboxUsageLabel,
  replyModeLabel,
  warmupKindLabel,
} from "@/lib/labels";

function mailboxError(message: string): string {
  switch (message) {
    case "smtp host not set":
      return "This mailbox has no SMTP server to test";
    case "reply_mode must be draft or off":
      return "Pick draft replies or don't draft";
    case "not found":
      return "Unknown send ramp";
    default:
      return message;
  }
}

export function MailboxList({
  mailboxes,
  usageByMailbox,
}: {
  mailboxes: Mailbox[];
  usageByMailbox: Record<string, number>;
}) {
  if (mailboxes.length === 0) {
    return <p className="font-body-md text-muted-foreground">No mailboxes yet.</p>;
  }
  return (
    <ul className="flex flex-col gap-3">
      {mailboxes.map((mb) => (
        <MailboxRow
          key={mb.id}
          mailbox={mb}
          used={usageByMailbox[mb.id] ?? 0}
        />
      ))}
    </ul>
  );
}

function MailboxRow({
  mailbox,
  used,
}: {
  mailbox: Mailbox;
  used: number;
}) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);
  const [replyMode, setReplyModeState] = useState(
    mailbox.reply_mode === "off" ? "off" : "draft",
  );
  const [warmupKind, setWarmupKind] = useState(
    mailbox.warmup_kind || "standard",
  );

  async function post(path: string, opts?: { okMsg?: string; refresh?: boolean }) {
    setMsg("");
    setPending(true);
    const res = await fetch(path, { method: "POST", credentials: "include" });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(mailboxError(data?.error?.message ?? "Request failed"));
      return;
    }
    if (opts?.okMsg) {
      setMsg(opts.okMsg);
    }
    if (opts?.refresh) {
      router.refresh();
    }
  }

  async function setReplyMode(mode: string) {
    const prev = replyMode;
    setReplyModeState(mode);
    setMsg("");
    setPending(true);
    const res = await fetch(`/api/v1/mailboxes/${mailbox.id}`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reply_mode: mode }),
    });
    setPending(false);
    if (!res.ok) {
      setReplyModeState(prev);
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(mailboxError(data?.error?.message ?? "Could not update replies"));
      return;
    }
    router.refresh();
  }

  async function setRamp(kind: string) {
    const prev = warmupKind;
    setWarmupKind(kind);
    setMsg("");
    setPending(true);
    const res = await fetch(`/api/v1/mailboxes/${mailbox.id}`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ warmup_kind: kind }),
    });
    setPending(false);
    if (!res.ok) {
      setWarmupKind(prev);
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(mailboxError(data?.error?.message ?? "Could not update send ramp"));
      return;
    }
    router.refresh();
  }

  return (
    <li className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-card p-4">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <p className="font-label-md text-foreground">{mailbox.email}</p>
          <Pill
            tone={
              mailbox.status === "paused" || mailbox.status === "disconnected"
                ? "danger"
                : mailbox.status === "warming"
                  ? "primary"
                  : "success"
            }
          >
            {mailboxStatus(mailbox.status)}
          </Pill>
        </div>
        <p className="mt-1 font-body-sm tabular-nums text-muted-foreground">
          {authKindLabel(mailbox.auth_kind)}
          {mailbox.smtp_host
            ? ` · ${mailbox.smtp_host}:${mailbox.smtp_port}`
            : ""}
          {` · ${mailboxUsageLabel(used, mailbox.effective_daily_limit)}`}
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        {msg ? (
          <p className="font-body-md text-muted-foreground">{msg}</p>
        ) : null}
        <select
          aria-label="Send ramp"
          disabled={pending}
          value={warmupKind}
          onChange={(e) => setRamp(e.target.value)}
          className={`${fieldSelect} w-auto min-w-[11rem]`}
        >
          <option value="standard">{warmupKindLabel("standard")}</option>
          <option value="conservative">{warmupKindLabel("conservative")}</option>
          <option value="fast">{warmupKindLabel("fast")}</option>
          <option value="off">{warmupKindLabel("off")}</option>
        </select>
        <select
          aria-label="When they reply"
          disabled={pending}
          value={replyMode}
          onChange={(e) => setReplyMode(e.target.value)}
          className={`${fieldSelect} w-auto min-w-[12rem]`}
        >
          <option value="draft">{replyModeLabel("draft")}</option>
          <option value="off">{replyModeLabel("off")}</option>
        </select>
        {mailbox.status === "paused" ? (
          <Button
            type="button"
            className="rounded-md"
            pending={pending}
            onClick={() =>
              post(`/api/v1/mailboxes/${mailbox.id}/reconnect`, {
                refresh: true,
              })
            }
          >
            Resume
          </Button>
        ) : mailbox.status === "disconnected" ? (
          <Button
            type="button"
            className="rounded-md"
            pending={pending}
            onClick={() =>
              post(`/api/v1/mailboxes/${mailbox.id}/reconnect`, {
                refresh: true,
              })
            }
          >
            Reconnect
          </Button>
        ) : (
          <Button
            type="button"
            variant="outline"
            className="rounded-md"
            pending={pending}
            onClick={() =>
              post(`/api/v1/mailboxes/${mailbox.id}/pause`, { refresh: true })
            }
          >
            Pause
          </Button>
        )}
        <Button
          type="button"
          variant="outline"
          className="rounded-md"
          pending={pending}
          onClick={() =>
            post(`/api/v1/mailboxes/${mailbox.id}/test`, {
              okMsg: "Server answered",
            })
          }
        >
          Test
        </Button>
      </div>
    </li>
  );
}
