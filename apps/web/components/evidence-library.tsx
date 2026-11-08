"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Pill } from "@/components/page-header";
import type { EvidenceItem, Profile } from "@/lib/api";
import { fieldSelect, fieldTextarea } from "@/lib/labels";

export function EvidenceLibrary({
  items,
  profiles = [],
}: {
  items: EvidenceItem[];
  profiles?: Profile[];
}) {
  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
      <div className="lg:col-span-7">
        {items.length === 0 ? (
          <p className="font-body-md text-muted-foreground">No evidence yet.</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {items.map((item) => (
              <EvidenceRow key={item.id} item={item} profiles={profiles} />
            ))}
          </ul>
        )}
      </div>
      <div className="lg:col-span-5">
        <ProposeForm profiles={profiles} />
      </div>
    </div>
  );
}

function EvidenceRow({
  item,
  profiles,
}: {
  item: EvidenceItem;
  profiles: Profile[];
}) {
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
    ? "Do not use"
    : item.confirmed
      ? "Confirmed"
      : "Unconfirmed";
  const profile = profiles.find((p) => p.id === item.profile_id);

  return (
    <li className="rounded-md border border-border bg-card p-4">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <Pill
          tone={
            item.do_not_use ? "danger" : item.confirmed ? "success" : "muted"
          }
        >
          {status}
        </Pill>
        {profile?.name || item.source_ref ? (
          <span className="font-body-sm text-muted-foreground">
            {[profile?.name, item.source_ref].filter(Boolean).join(" · ")}
          </span>
        ) : null}
      </div>
      <p className="whitespace-pre-wrap font-body-md text-foreground">
        {item.text}
      </p>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        {!item.confirmed && !item.do_not_use ? (
          <Button
            type="button"
            className="rounded-md"
            pending={pending}
            onClick={onConfirm}
          >
            Confirm
          </Button>
        ) : null}
        {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      </div>
    </li>
  );
}

function ProposeForm({ profiles }: { profiles: Profile[] }) {
  const router = useRouter();
  const [text, setText] = useState("");
  const [sourceRef, setSourceRef] = useState("");
  const [profileID, setProfileID] = useState("");
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
        profile_id: profileID || undefined,
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
    setProfileID("");
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
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
          className={fieldTextarea}
        />
      </div>
      {profiles.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="evidence-profile">Profile</Label>
          <select
            id="evidence-profile"
            value={profileID}
            onChange={(e) => setProfileID(e.target.value)}
            className={fieldSelect}
          >
            <option value="">None</option>
            {profiles.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>
      ) : null}
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="evidence-source">Source (optional)</Label>
        <Input
          id="evidence-source"
          value={sourceRef}
          onChange={(e) => setSourceRef(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Propose
      </Button>
    </form>
  );
}
