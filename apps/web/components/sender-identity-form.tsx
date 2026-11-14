"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Panel } from "@/components/page-header";
import type { SenderIdentity } from "@/lib/api";

export function SenderIdentityForm({
  identity,
}: {
  identity?: SenderIdentity | null;
}) {
  const [legalName, setLegalName] = useState(identity?.legal_name ?? "");
  const [reply, setReply] = useState(identity?.reply_address ?? "");
  const [postal, setPostal] = useState(identity?.postal_address ?? "");
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    setPending(true);
    const res = await fetch("/api/v1/sender-identity", {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        legal_name: legalName,
        reply_address: reply,
        postal_address: postal,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(data?.error?.message ?? "Could not save identity");
      return;
    }
    setMsg("Saved");
  }

  return (
    <Panel
      title="Sender identity"
      description="Required only for cold and freelance mail. Those messages end with your legal name, a reply address, and a postal address, plus an unsubscribe link. Job and referral mail do not use this."
    >
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="id-name">Legal name</Label>
        <Input
          id="id-name"
          required
          value={legalName}
          onChange={(e) => setLegalName(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="id-reply">Reply address</Label>
        <Input
          id="id-reply"
          type="email"
          required
          value={reply}
          onChange={(e) => setReply(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="id-postal">Postal address</Label>
        <Input
          id="id-postal"
          required
          value={postal}
          onChange={(e) => setPostal(e.target.value)}
        />
      </div>
      {msg ? (
        <p role="status" className="font-body-md text-muted-foreground">
          {msg}
        </p>
      ) : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Save identity
      </Button>
    </form>
    </Panel>
  );
}
