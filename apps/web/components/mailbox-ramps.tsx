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
import { warmupKindLabel } from "@/lib/labels";
import type { WarmupProfile } from "@/lib/api";

export function MailboxRamps({ profiles }: { profiles: WarmupProfile[] }) {
  const [open, setOpen] = useState(false);
  const add = (
    <Button
      type="button"
      variant="outline"
      className="rounded-md"
      onClick={() => setOpen(true)}
    >
      Custom ramp
    </Button>
  );

  return (
    <>
      <Panel title="Send ramps" actions={profiles.length > 0 ? add : undefined}>
        {profiles.length === 0 ? (
          <EmptyState
            title="No ramps yet"
            description="A ramp is how many sends a mailbox may start with each day."
            action={add}
          />
        ) : (
          <ul className="flex flex-col gap-2">
            {profiles.map((p) => (
              <li
                key={p.id}
                className="flex flex-wrap items-baseline justify-between gap-2 rounded-md border border-border bg-muted/50 px-3 py-2"
              >
                <p className="font-label-md text-foreground">
                  {p.name || warmupKindLabel(p.kind)}
                </p>
                <p className="font-body-sm text-muted-foreground">
                  {p.first_day != null
                    ? `Starts at ${p.first_day}/day`
                    : warmupKindLabel(p.kind)}
                </p>
              </li>
            ))}
          </ul>
        )}
      </Panel>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>Custom ramp</SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            <RampForm onDone={() => setOpen(false)} />
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}

function RampForm({ onDone }: { onDone: () => void }) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [start, setStart] = useState("2");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/warmup-profiles", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, start: Number(start) }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not save ramp");
      return;
    }
    setName("");
    setStart("2");
    onDone();
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ramp-name">Name</Label>
        <Input
          id="ramp-name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ramp-start">Starts at (per day)</Label>
        <Input
          id="ramp-start"
          type="number"
          min={1}
          required
          value={start}
          onChange={(e) => setStart(e.target.value)}
        />
      </div>
      {error ? (
        <p role="alert" className="font-body-sm text-destructive">
          {error}
        </p>
      ) : null}
      <Button type="submit" className="w-fit rounded-md" pending={pending}>
        Save ramp
      </Button>
    </form>
  );
}
