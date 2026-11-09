"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState, Panel } from "@/components/page-header";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { Mailbox, MailboxPool } from "@/lib/api";

export function MailboxGroups({
  mailboxes,
  pools,
}: {
  mailboxes: Mailbox[];
  pools: MailboxPool[];
}) {
  const [open, setOpen] = useState(false);
  const newGroup = (
    <Button
      type="button"
      variant="outline"
      className="rounded-md"
      onClick={() => setOpen(true)}
    >
      New group
    </Button>
  );

  return (
    <>
      <Panel title="Groups" actions={pools.length > 0 ? newGroup : undefined}>
        {pools.length === 0 ? (
          <EmptyState
            title="No groups yet"
            description="A group is several mailboxes that share a campaign."
            action={newGroup}
          />
        ) : (
          <ul className="flex flex-col gap-2">
            {pools.map((p) => (
              <li
                key={p.id}
                className="rounded-md border border-border bg-muted/50 px-3 py-2"
              >
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
      </Panel>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>New group</SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            <GroupForm
              mailboxes={mailboxes}
              onDone={() => setOpen(false)}
            />
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}

function GroupForm({
  mailboxes,
  onDone,
}: {
  mailboxes: Mailbox[];
  onDone: () => void;
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
    onDone();
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
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
      ) : (
        <p className="font-body-md text-muted-foreground">
          Connect a mailbox first, then add it to a group.
        </p>
      )}
      {error ? (
        <p role="alert" className="font-body-md text-destructive">
          {error}
        </p>
      ) : null}
      <Button type="submit" className="w-fit rounded-md" pending={pending}>
        Save group
      </Button>
    </form>
  );
}
