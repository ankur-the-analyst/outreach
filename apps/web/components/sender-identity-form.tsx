"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4 rounded-md border border-border bg-muted p-4"
    >
      <div>
        <h2 className="font-headline-sm text-foreground">Sender identity</h2>
        <p className="mt-1 font-body-sm text-muted-foreground">
          Needed before you start cold or freelance campaigns (unsubscribe
          footer).
        </p>
      </div>
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
      {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Save identity
      </Button>
    </form>
  );
}
