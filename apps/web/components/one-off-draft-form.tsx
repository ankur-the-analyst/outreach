"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Mailbox, Profile } from "@/lib/api";
import { fieldSelect, fieldTextarea } from "@/lib/labels";

export function OneOffDraftForm({
  mailboxes,
  profiles = [],
  onSaved,
}: {
  mailboxes: Mailbox[];
  profiles?: Profile[];
  onSaved?: () => void;
}) {
  const router = useRouter();
  const [mailboxID, setMailboxID] = useState(mailboxes[0]?.id ?? "");
  const [profileID, setProfileID] = useState("");
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
    onSaved?.();
  }

  async function writeFromClaims() {
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/drafts/compose", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        mailbox_id: mailboxID,
        to,
        subject,
        profile_id: profileID || undefined,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not write from claims");
      return;
    }
    setTo("");
    setSubject("");
    setText("");
    router.refresh();
    onSaved?.();
  }

  if (mailboxes.length === 0) {
    return (
      <p className="font-body-md text-muted-foreground">
        Connect a mailbox first.
      </p>
    );
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="draft-mailbox">Mailbox</Label>
        <select
          id="draft-mailbox"
          required
          value={mailboxID}
          onChange={(e) => setMailboxID(e.target.value)}
          className={fieldSelect}
        >
          {mailboxes.map((mb) => (
            <option key={mb.id} value={mb.id}>
              {mb.email}
            </option>
          ))}
        </select>
      </div>
      {profiles.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="draft-profile">Claims for this profile</Label>
          <select
            id="draft-profile"
            value={profileID}
            onChange={(e) => setProfileID(e.target.value)}
            className={fieldSelect}
          >
            <option value="">Any profile</option>
            {profiles.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>
      ) : null}
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
          className={`${fieldTextarea} max-w-[68ch]`}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <div className="flex justify-end gap-2">
        <Button
          type="button"
          variant="outline"
          className="rounded-md"
          pending={pending}
          disabled={!to || !subject}
          onClick={writeFromClaims}
        >
          Write from claims
        </Button>
        <Button type="submit" className="rounded-md" pending={pending}>
          Save to review
        </Button>
      </div>
    </form>
  );
}
