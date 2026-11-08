"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Mailbox, MailboxPool } from "@/lib/api";

export function MailboxGroups({
  mailboxes,
  pools,
}: {
  mailboxes: Mailbox[];
  pools: MailboxPool[];
}) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [picked, setPicked] = useState<Record<string, boolean>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/mailbox-pools", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name,
        mailbox_ids: Object.keys(picked).filter((id) => picked[id]),
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not save group");
      return;
    }
    setName("");
    setPicked({});
    router.refresh();
  }

  return (
    <div className="flex flex-col gap-4 rounded-md border border-border bg-card p-4">
      <h2 className="font-headline-sm text-foreground">Groups</h2>
      {pools.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No groups yet.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {pools.map((p) => (
            <li key={p.id}>
              <p className="font-label-md text-foreground">{p.name}</p>
              <p className="font-body-sm text-muted-foreground">
                {p.mailboxes.length === 0
                  ? "No mailboxes"
                  : p.mailboxes.map((m) => m.email).join(", ")}
              </p>
            </li>
          ))}
        </ul>
      )}
      <form
        onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
        className="flex flex-col gap-3"
      >
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pool-name">Group name</Label>
          <Input
            id="pool-name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        {mailboxes.length > 0 ? (
          <fieldset className="flex flex-col gap-2">
            <legend className="font-label-sm text-muted-foreground">
              Mailboxes
            </legend>
            {mailboxes.map((mb) => (
              <label
                key={mb.id}
                className="flex items-center gap-2 font-body-md text-foreground"
              >
                <input
                  type="checkbox"
                  checked={!!picked[mb.id]}
                  onChange={(e) =>
                    setPicked((cur) => ({ ...cur, [mb.id]: e.target.checked }))
                  }
                />
                {mb.email}
              </label>
            ))}
          </fieldset>
        ) : null}
        {error ? <p className="font-body-md text-destructive">{error}</p> : null}
        <Button type="submit" className="w-fit rounded-md" pending={pending}>
          Save group
        </Button>
      </form>
    </div>
  );
}
