"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Mailbox } from "@/lib/api";

export function OneOffDraftForm({ mailboxes }: { mailboxes: Mailbox[] }) {
  const router = useRouter();
  const [mailboxID, setMailboxID] = useState(mailboxes[0]?.id ?? "");
  const [to, setTo] = useState("");
  const [subject, setSubject] = useState("");
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/drafts", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        mailbox_id: mailboxID,
        to,
        subject,
        text,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not save draft");
      return;
    }
    setTo("");
    setSubject("");
    setText("");
    router.refresh();
  }

  if (mailboxes.length === 0) {
    return (
      <p className="mt-8 font-body-md text-muted-foreground">
        Connect a mailbox first.
      </p>
    );
  }

  return (
    <form
      onSubmit={onSubmit}
      className="mt-8 flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">One-off draft</h2>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="draft-mailbox">Mailbox</Label>
        <select
          id="draft-mailbox"
          required
          value={mailboxID}
          onChange={(e) => setMailboxID(e.target.value)}
          className="h-8 w-full rounded-md border border-input bg-card px-2.5 font-[family-name:var(--font-body)] text-sm text-foreground outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
        >
          {mailboxes.map((mb) => (
            <option key={mb.id} value={mb.id}>
              {mb.email}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="draft-to">To</Label>
        <Input
          id="draft-to"
          type="email"
          required
          value={to}
          onChange={(e) => setTo(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="draft-subject">Subject</Label>
        <Input
          id="draft-subject"
          required
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="draft-body">Body</Label>
        <textarea
          id="draft-body"
          required
          rows={6}
          value={text}
          onChange={(e) => setText(e.target.value)}
          className="w-full rounded-md border border-input bg-card px-2.5 py-2 font-[family-name:var(--font-body)] text-sm text-foreground outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" disabled={pending}>
        {pending ? "Saving…" : "Save to review"}
      </Button>
    </form>
  );
}
