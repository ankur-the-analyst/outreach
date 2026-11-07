"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { Mailbox } from "@/lib/api";

function mailboxStatus(status: string): string {
  switch (status) {
    case "warming":
      return "Warming up";
    case "healthy":
      return "Ready";
    case "paused":
      return "Paused";
    case "disconnected":
      return "Disconnected";
    default:
      return status;
  }
}

function authKindLabel(kind: string): string {
  switch (kind) {
    case "app_password":
      return "Gmail app password";
    case "smtp_imap":
      return "SMTP";
    case "service_account":
      return "Google Workspace";
    case "oauth":
      return "OAuth";
    default:
      return kind;
  }
}

export function MailboxList({ mailboxes }: { mailboxes: Mailbox[] }) {
  if (mailboxes.length === 0) {
    return <p className="font-body-md text-muted-foreground">No mailboxes yet.</p>;
  }
  return (
    <ul className="flex flex-col gap-3">
      {mailboxes.map((mb) => (
        <MailboxRow key={mb.id} mailbox={mb} />
      ))}
    </ul>
  );
}

function MailboxRow({ mailbox }: { mailbox: Mailbox }) {
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);

  async function onTest() {
    setMsg("");
    setPending(true);
    const res = await fetch(`/api/v1/mailboxes/${mailbox.id}/test`, {
      method: "POST",
      credentials: "include",
    });
    setPending(false);
    if (res.ok) {
      setMsg("Server answered");
      return;
    }
    const data = (await res.json().catch(() => null)) as
      | { error?: { message?: string } }
      | null;
    setMsg(data?.error?.message ?? "Test failed");
  }

  return (
    <li className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-card p-4">
      <div>
        <p className="font-label-md text-foreground">{mailbox.email}</p>
        <p className="mt-1 font-body-md tabular-nums text-muted-foreground">
          {mailboxStatus(mailbox.status)} · {authKindLabel(mailbox.auth_kind)}
          {mailbox.smtp_host
            ? ` · ${mailbox.smtp_host}:${mailbox.smtp_port}`
            : ""}
        </p>
      </div>
      <div className="flex items-center gap-3">
        {msg ? (
          <p className="font-body-md text-muted-foreground">{msg}</p>
        ) : null}
        <Button
          type="button"
          variant="outline"
          className="rounded-md"
          disabled={pending}
          onClick={onTest}
        >
          {pending ? "Testing…" : "Test"}
        </Button>
      </div>
    </li>
  );
}
