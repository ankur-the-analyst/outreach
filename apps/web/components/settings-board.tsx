"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PageBody, PageHeader, Pill } from "@/components/page-header";
import { SenderIdentityForm } from "@/components/sender-identity-form";
import type { Member, SenderIdentity, Workspace } from "@/lib/api";

function roleLabel(role: string): string {
  if (role === "owner") {
    return "Owner";
  }
  if (role === "viewer") {
    return "Viewer";
  }
  return "Member";
}

export function SettingsBoard({
  workspace,
  identity,
  members,
}: {
  workspace: Workspace | null;
  identity: SenderIdentity | null;
  members: Member[];
}) {
  const owner = workspace?.role === "owner";

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Settings" />
      <PageBody>
        <div className="grid max-w-5xl grid-cols-1 gap-6 lg:grid-cols-2">
          <PauseSends
            paused={workspace?.sends_paused ?? false}
            canEdit={owner}
          />
          <SenderIdentityForm identity={identity} />
          <div className="lg:col-span-2">
            <MembersList members={members} canInvite={owner} />
          </div>
        </div>
      </PageBody>
    </div>
  );
}

function PauseSends({ paused, canEdit }: { paused: boolean; canEdit: boolean }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  async function setPaused(next: boolean) {
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/workspace/kill-switch", {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ paused: next }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not update sending");
      return;
    }
    router.refresh();
  }

  return (
    <div className="flex flex-col gap-4 rounded-md border border-border bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 className="font-headline-sm text-foreground">Sending</h2>
          <p className="mt-1 font-body-sm text-muted-foreground">
            Pause stops new sends. Mail already going out may still finish.
          </p>
        </div>
        <Pill tone={paused ? "danger" : "success"}>
          {paused ? "Paused" : "Sending"}
        </Pill>
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      {canEdit ? (
        <Button
          type="button"
          variant={paused ? "default" : "outline"}
          className="w-fit rounded-md"
          pending={pending}
          onClick={() => setPaused(!paused)}
        >
          {paused ? "Resume sending" : "Pause all sending"}
        </Button>
      ) : (
        <p className="font-body-sm text-muted-foreground">
          Only an owner can pause or resume.
        </p>
      )}
    </div>
  );
}

function MembersList({
  members,
  canInvite,
}: {
  members: Member[];
  canInvite: boolean;
}) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  async function onInvite(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/workspace/members", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, role: "member" }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not add person");
      return;
    }
    setEmail("");
    router.refresh();
  }

  return (
    <div className="flex flex-col gap-4 rounded-md border border-border bg-card p-4">
      <h2 className="font-headline-sm text-foreground">People</h2>
      {members.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No people yet.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {members.map((m) => (
            <li
              key={m.user_id}
              className="flex flex-wrap items-center justify-between gap-2"
            >
              <div>
                <p className="font-label-md text-foreground">
                  {m.name || m.email}
                </p>
                {m.name ? (
                  <p className="font-body-sm text-muted-foreground">{m.email}</p>
                ) : null}
              </div>
              <Pill tone="muted">{roleLabel(m.role)}</Pill>
            </li>
          ))}
        </ul>
      )}
      {canInvite ? (
        <form
          onSubmit={pending ? (e) => e.preventDefault() : onInvite}
          className="flex flex-col gap-3 sm:flex-row sm:items-end"
        >
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Label htmlFor="invite-email">Add someone who already has an account</Label>
            <Input
              id="invite-email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <Button type="submit" className="rounded-md" pending={pending}>
            Add
          </Button>
        </form>
      ) : null}
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
    </div>
  );
}
