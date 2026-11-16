"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState, PageBody, PageHeader, Panel, Pill } from "@/components/page-header";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { AISettingsForm } from "@/components/ai-settings";
import { FieldCatalog } from "@/components/field-catalog";
import { SenderIdentityForm } from "@/components/sender-identity-form";
import { fieldSelect } from "@/lib/labels";
import type { Member, ModelSettings, SenderIdentity, Workspace } from "@/lib/api";

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
  models,
}: {
  workspace: Workspace | null;
  identity: SenderIdentity | null;
  members: Member[];
  models?: ModelSettings | null;
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
          <DuplicateRule canEdit={owner} />
          <SenderIdentityForm identity={identity} />
          <AISettingsForm settings={models} />
          <FieldCatalog
            object="member"
            title="Your fields"
            description="About you in this workspace. Built-in fields stay fixed. A template fills {{user.key}} from the values you save here."
            values
          />
          <div className="lg:col-span-2">
            <MembersList members={members} canInvite={owner} />
          </div>
        </div>
      </PageBody>
    </div>
  );
}

function DuplicateRule({ canEdit }: { canEdit: boolean }) {
  const [action, setAction] = useState("prevent");
  const [overwrite, setOverwrite] = useState(false);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    void (async () => {
      const res = await fetch("/api/v1/settings/contact-duplicates", { credentials: "include" });
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as { action?: string; overwrite?: boolean };
      if (data.action === "update") {
        setAction("update");
        setOverwrite(Boolean(data.overwrite));
      }
    })();
  }, []);

  const value = action === "update" && overwrite ? "replace" : action;

  async function onSave(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const next = value === "replace" ? "update" : value;
    const res = await fetch("/api/v1/settings/contact-duplicates", {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: next, overwrite: value === "replace" }),
    });
    setPending(false);
    if (!res.ok) {
      setError("Could not save");
      return;
    }
    setAction(next);
    setOverwrite(value === "replace");
  }

  return (
    <Panel
      title="Repeated email"
      description="One address stays one person in this workspace. Choose what happens when that address is added again."
    >
      <form onSubmit={onSave} className="flex flex-col gap-3">
        <select
          aria-label="When this email already exists"
          value={value}
          disabled={!canEdit}
          onChange={(e) => {
            const v = e.target.value;
            setAction(v === "replace" ? "update" : v);
            setOverwrite(v === "replace");
          }}
          className={fieldSelect}
        >
          <option value="prevent">Keep the person already saved</option>
          <option value="update">Fill blanks on that person</option>
          <option value="replace">Replace filled values</option>
        </select>
        {canEdit ? (
          <Button type="submit" className="w-fit rounded-md" pending={pending}>
            Save
          </Button>
        ) : (
          <p className="font-body-sm text-muted-foreground">Only an owner can change this.</p>
        )}
        {error ? (
          <p role="alert" className="font-body-sm text-destructive">
            {error}
          </p>
        ) : null}
      </form>
    </Panel>
  );
}

function PauseSends({ paused, canEdit }: { paused: boolean; canEdit: boolean }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [pauseOpen, setPauseOpen] = useState(false);

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
    <Panel
      title="Sending"
      description="Pause stops new sends. Mail already going out may still finish."
      actions={
        <Pill tone={paused ? "danger" : "success"}>
          {paused ? "Paused" : "Sending"}
        </Pill>
      }
    >
      {error ? (
        <p role="alert" className="font-body-md text-destructive">
          {error}
        </p>
      ) : null}
      {canEdit ? (
        <Button
          type="button"
          variant={paused ? "default" : "outline"}
          className="w-fit rounded-md"
          pending={pending}
          onClick={() => {
            if (paused) {
              void setPaused(false);
              return;
            }
            setPauseOpen(true);
          }}
        >
          {paused ? "Resume sending" : "Pause all sending"}
        </Button>
      ) : (
        <p className="font-body-sm text-muted-foreground">
          Only an owner can pause or resume.
        </p>
      )}
      <ConfirmDialog
        open={pauseOpen}
        onOpenChange={setPauseOpen}
        title="Pause all sending?"
        description="Nothing new goes out. Mail already sending may still finish."
        confirmLabel="Pause all sending"
        confirmTone="danger"
        loading={pending}
        onConfirm={async () => {
          await setPaused(true);
          setPauseOpen(false);
        }}
      />
    </Panel>
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
  const [inviteOpen, setInviteOpen] = useState(false);
  const [once, setOnce] = useState<{ email: string; password: string } | null>(
    null,
  );

  async function onInvite(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setOnce(null);
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
    const data = (await res.json()) as { email?: string; password?: string };
    if (data.password) {
      setOnce({ email: data.email ?? email, password: data.password });
    }
    setEmail("");
    setInviteOpen(false);
    router.refresh();
  }

  const invite = canInvite ? (
    <Button
      type="button"
      variant="outline"
      className="rounded-md"
      onClick={() => setInviteOpen(true)}
    >
      Add a person
    </Button>
  ) : undefined;

  return (
    <>
    <Panel title="People" actions={members.length > 0 ? invite : undefined}>
      {members.length === 0 ? (
        <EmptyState
          compact
          title="No people yet"
          description="Add someone so they can sign in to this workspace."
          action={invite}
        />
      ) : (
        <ul className="flex flex-col gap-2">
          {members.map((m) => (
            <li
              key={m.user_id}
              className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border bg-muted/50 px-3 py-2"
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
      {once ? (
        <p role="status" className="font-body-md text-foreground">
          Share this sign-in password with {once.email} once:{" "}
          <span className="font-label-md">{once.password}</span>
        </p>
      ) : null}
      {error ? (
        <p role="alert" className="font-body-md text-destructive">
          {error}
        </p>
      ) : null}
    </Panel>
    <Sheet open={inviteOpen} onOpenChange={setInviteOpen}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Add a person</SheetTitle>
        </SheetHeader>
        <form
          onSubmit={pending ? (e) => e.preventDefault() : onInvite}
          className="flex flex-col gap-4 px-4 pb-4"
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="invite-email">Email</Label>
            <Input
              id="invite-email"
              type="email"
              required
              autoComplete="off"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          {error ? (
            <p role="alert" className="font-body-md text-destructive">
              {error}
            </p>
          ) : null}
          <Button type="submit" className="rounded-md" pending={pending}>
            Add
          </Button>
        </form>
      </SheetContent>
    </Sheet>
    </>
  );
}
