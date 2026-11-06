"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { Mailbox } from "@/lib/api";

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
      setMsg("SMTP reachable");
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
          {mailbox.status} · {mailbox.auth_kind}
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
