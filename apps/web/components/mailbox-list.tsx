"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Pill } from "@/components/page-header";
import type { Mailbox } from "@/lib/api";
import { authKindLabel, mailboxStatus } from "@/lib/labels";

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
          {` · cap ${mailbox.effective_daily_limit}/day`}
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
          pending={pending}
          onClick={onTest}
        >
          Test
        </Button>
      </div>
    </li>
  );
}
