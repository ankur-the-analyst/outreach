"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Campaign, Contact, Mailbox } from "@/lib/api";

const intentLabel: Record<string, string> = {
  job: "Job search",
  referral: "Referral",
  informational: "Informational",
  cold: "Cold outreach",
  freelance: "Freelance",
};

const statusLabel: Record<string, string> = {
  draft: "Draft",
  running: "Running",
  paused: "Paused",
  stopped: "Stopped",
  completed: "Done",
};

function operatorError(message: string): string {
  switch (message) {
    case "unknown intent pack":
    case "unknown campaign type":
      return "Unknown campaign type";
    case "enrolment context required":
    case "add a note, a job, or campaign instructions":
      return "Add a note, a job, or campaign instructions";
    case "already enrolled":
    case "already in this campaign":
      return "Already in this campaign";
    case "sender identity required":
      return "Add your name and postal address first";
    default:
      return message;
  }
}

export function CampaignBoard({
  campaigns,
  contacts,
  mailboxes,
}: {
  campaigns: Campaign[];
  contacts: Contact[];
  mailboxes: Mailbox[];
}) {
  return (
    <div className="flex flex-col gap-8">
      {campaigns.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No campaigns yet.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {campaigns.map((c) => (
            <CampaignRow key={c.id} campaign={c} contacts={contacts} />
          ))}
        </ul>
      )}
      <CreateCampaign mailboxes={mailboxes} />
      <SenderIdentityForm />
    </div>
  );
}

function CampaignRow({
  campaign,
  contacts,
}: {
  campaign: Campaign;
  contacts: Contact[];
}) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);
  const [contactID, setContactID] = useState(contacts[0]?.id ?? "");
  const [notes, setNotes] = useState("");

  async function post(path: string, body?: unknown) {
    setMsg("");
    setPending(true);
    const res = await fetch(path, {
      method: "POST",
      credentials: "include",
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(operatorError(data?.error?.message ?? "Request failed"));
      return;
    }
    router.refresh();
  }

  return (
    <li className="rounded-md border border-border bg-card p-4">
      <p className="font-label-md text-foreground">{campaign.name}</p>
      <p className="mt-1 font-body-md text-muted-foreground">
        {intentLabel[campaign.intent] ?? campaign.intent} ·{" "}
        {statusLabel[campaign.status] ?? campaign.status}
        {campaign.list_unsubscribe ? " · includes unsubscribe" : ""}
        {` · ${campaign.steps.filter((s) => s.kind === "email").length} emails`}
      </p>
      <div className="mt-4 flex flex-wrap items-end gap-3">
        {campaign.status === "draft" || campaign.status === "paused" ? (
          <Button
            type="button"
            className="rounded-md"
            disabled={pending}
            onClick={() => post(`/api/v1/campaigns/${campaign.id}/start`)}
          >
            Start
          </Button>
        ) : null}
        {campaign.status === "running" ? (
          <Button
            type="button"
            variant="outline"
            className="rounded-md"
            disabled={pending}
            onClick={() => post(`/api/v1/campaigns/${campaign.id}/pause`)}
          >
            Pause
          </Button>
        ) : null}
        {campaign.status === "running" || campaign.status === "paused" ? (
          <Button
            type="button"
            variant="outline"
            className="rounded-md"
            disabled={pending}
            onClick={() => post(`/api/v1/campaigns/${campaign.id}/stop`)}
          >
            Stop
          </Button>
        ) : null}
        {contacts.length > 0 ? (
          <>
            <select
              value={contactID}
              onChange={(e) => setContactID(e.target.value)}
              className="h-8 rounded-md border border-input bg-card px-2.5 font-[family-name:var(--font-body)] text-sm"
            >
              {contacts.map((ct) => (
                <option key={ct.id} value={ct.id}>
                  {ct.email}
                </option>
              ))}
            </select>
            <Input
              placeholder="Why you're writing"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
            <Button
              type="button"
              variant="outline"
              className="rounded-md"
              disabled={pending || !contactID}
              onClick={() =>
                post(`/api/v1/campaigns/${campaign.id}/enrolments`, {
                  contact_id: contactID,
                  notes,
                })
              }
            >
              Add to campaign
            </Button>
          </>
        ) : null}
        {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      </div>
    </li>
  );
}

function CreateCampaign({ mailboxes }: { mailboxes: Mailbox[] }) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [intent, setIntent] = useState("job");
  const [mailboxID, setMailboxID] = useState(mailboxes[0]?.id ?? "");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/campaigns", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name,
        intent,
        mailbox_ids: mailboxID ? [mailboxID] : [],
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(operatorError(data?.error?.message ?? "Could not create campaign"));
      return;
    }
    setName("");
    router.refresh();
  }

  if (mailboxes.length === 0) {
    return (
      <p className="font-body-md text-muted-foreground">
        Connect a mailbox before creating a campaign.
      </p>
    );
  }

  return (
    <form
      onSubmit={onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">New campaign</h2>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="camp-name">Name</Label>
        <Input
          id="camp-name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="camp-intent">Campaign type</Label>
        <select
          id="camp-intent"
          value={intent}
          onChange={(e) => setIntent(e.target.value)}
          className="h-8 w-full rounded-md border border-input bg-card px-2.5 font-[family-name:var(--font-body)] text-sm"
        >
          <option value="job">Job search</option>
          <option value="referral">Referral</option>
          <option value="informational">Informational</option>
          <option value="cold">Cold outreach</option>
          <option value="freelance">Freelance</option>
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="camp-mailbox">Mailbox</Label>
        <select
          id="camp-mailbox"
          required
          value={mailboxID}
          onChange={(e) => setMailboxID(e.target.value)}
          className="h-8 w-full rounded-md border border-input bg-card px-2.5 font-[family-name:var(--font-body)] text-sm"
        >
          {mailboxes.map((mb) => (
            <option key={mb.id} value={mb.id}>
              {mb.email}
            </option>
          ))}
        </select>
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" disabled={pending}>
        {pending ? "Saving…" : "Create"}
      </Button>
    </form>
  );
}

function SenderIdentityForm() {
  const [legalName, setLegalName] = useState("");
  const [reply, setReply] = useState("");
  const [postal, setPostal] = useState("");
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
      onSubmit={onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">Sender identity</h2>
      <p className="font-body-md text-muted-foreground">
        Needed before you start cold or freelance campaigns (unsubscribe
        footer).
      </p>
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
      <Button type="submit" className="rounded-md" disabled={pending}>
        {pending ? "Saving…" : "Save identity"}
      </Button>
    </form>
  );
}
