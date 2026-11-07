"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import type { Draft } from "@/lib/api";

function sendStatus(draft: Draft): string {
  if (draft.status === "pending") {
    return "Needs approval";
  }
  switch (draft.ledger_state) {
    case "held":
      return undoLabel(draft.release_at);
    case "reserved":
    case "sending":
      return "Sending";
    case "sent":
      return "Sent";
    case "unknown":
      return "Checking if it sent";
    case "cancelled":
      return "Cancelled";
    case "released":
      return "Not sent";
    case "failed":
      return "Could not send";
    default:
      return draft.status === "approved" ? "Approved" : draft.status;
  }
}

function undoLabel(releaseAt?: string): string {
  if (!releaseAt) {
    return "You can still undo";
  }
  const ms = new Date(releaseAt).getTime() - Date.now();
  if (ms <= 0) {
    return "You can still undo";
  }
  const mins = Math.max(1, Math.ceil(ms / 60_000));
  return `You can still undo · ${mins} min left`;
}

export function ReviewQueue({ drafts }: { drafts: Draft[] }) {
  if (drafts.length === 0) {
    return (
      <p className="font-body-md text-muted-foreground">No drafts in review.</p>
    );
  }
  return (
    <ul className="flex flex-col gap-3">
      {drafts.map((d) => (
        <ReviewRow key={d.id} draft={d} />
      ))}
    </ul>
  );
}

function ReviewRow({ draft }: { draft: Draft }) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);

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
    <li className="rounded-md border border-border bg-card p-4">
      <p className="font-label-md text-foreground">{draft.subject}</p>
      <p className="mt-1 font-body-md tabular-nums text-muted-foreground">
        To {draft.to} · {sendStatus(draft)}
      </p>
      <p className="mt-3 whitespace-pre-wrap font-body-md text-foreground">
        {draft.body_text}
      </p>
      <div className="mt-4 flex flex-wrap items-center gap-3">
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
            className="rounded-md"
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
