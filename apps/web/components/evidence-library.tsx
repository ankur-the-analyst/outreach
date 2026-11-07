"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { EvidenceItem } from "@/lib/api";

export function EvidenceLibrary({ items }: { items: EvidenceItem[] }) {
  return (
    <div className="flex flex-col gap-8">
      {items.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No evidence yet.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {items.map((item) => (
            <EvidenceRow key={item.id} item={item} />
          ))}
        </ul>
      )}
      <ProposeForm />
    </div>
  );
}

function EvidenceRow({ item }: { item: EvidenceItem }) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);

  async function onConfirm() {
    setMsg("");
    setPending(true);
    const res = await fetch(`/api/v1/evidence/${item.id}/confirm`, {
      method: "POST",
      credentials: "include",
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(data?.error?.message ?? "Confirm failed");
      return;
    }
    router.refresh();
  }

  const status = item.do_not_use
    ? "do not use"
    : item.confirmed
      ? "confirmed"
      : "unconfirmed";

  return (
    <li className="rounded-md border border-border bg-card p-4">
      <p className="whitespace-pre-wrap font-body-md text-foreground">{item.text}</p>
      <p className="mt-1 font-body-md text-muted-foreground">
        {status}
        {item.source_ref ? ` · ${item.source_ref}` : ""}
      </p>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        {!item.confirmed && !item.do_not_use ? (
          <Button
            type="button"
            className="rounded-md"
            disabled={pending}
            onClick={onConfirm}
          >
            {pending ? "…" : "Confirm"}
          </Button>
        ) : null}
        {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      </div>
    </li>
  );
}

function ProposeForm() {
  const router = useRouter();
  const [text, setText] = useState("");
  const [sourceRef, setSourceRef] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/evidence", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text,
        source_ref: sourceRef,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not propose");
      return;
    }
    setText("");
    setSourceRef("");
    router.refresh();
  }

  return (
    <form
      onSubmit={onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">Propose a claim</h2>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="evidence-text">Text</Label>
        <textarea
          id="evidence-text"
          required
          rows={4}
          value={text}
          onChange={(e) => setText(e.target.value)}
          className="w-full rounded-md border border-input bg-card px-2.5 py-2 font-[family-name:var(--font-body)] text-sm text-foreground outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="evidence-source">Source (optional)</Label>
        <Input
          id="evidence-source"
          value={sourceRef}
          onChange={(e) => setSourceRef(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" disabled={pending}>
        {pending ? "Saving…" : "Propose"}
      </Button>
    </form>
  );
}
