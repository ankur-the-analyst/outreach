"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Plus, UserPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState, PageBody, PageHeader, Pill } from "@/components/page-header";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type {
  Campaign,
  CampaignPerson,
  Contact,
  Job,
  Mailbox,
  MailboxPool,
  Profile,
} from "@/lib/api";
import {
  campaignPersonStatus,
  campaignStatusLabel,
  contactName,
  fieldSelect,
  fieldTextarea,
  intentLabel,
} from "@/lib/labels";

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
    case "mailbox required":
      return "Pick a mailbox or a group";
    default:
      return message;
  }
}

function statusTone(
  status: string,
): "muted" | "primary" | "secondary" | "success" | "danger" {
  if (status === "running") {
    return "success";
  }
  if (status === "paused") {
    return "muted";
  }
  if (status === "stopped") {
    return "danger";
  }
  return "secondary";
}

export function CampaignBoard({
  campaigns,
  nextCursor,
  contacts,
  mailboxes,
  pools = [],
  jobs,
  profiles,
  peopleByCampaign,
}: {
  campaigns: Campaign[];
  nextCursor?: string;
  contacts: Contact[];
  mailboxes: Mailbox[];
  pools?: MailboxPool[];
  jobs: Job[];
  profiles: Profile[];
  peopleByCampaign: Record<string, CampaignPerson[]>;
}) {
  const [rows, setRows] = useState(campaigns);
  const [cursor, setCursor] = useState(nextCursor);
  const [people, setPeople] = useState(peopleByCampaign);
  const [morePending, setMorePending] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [addFor, setAddFor] = useState<string | null>(null);

  useEffect(() => {
    setRows(campaigns);
    setCursor(nextCursor);
    setPeople(peopleByCampaign);
  }, [campaigns, nextCursor, peopleByCampaign]);

  async function loadMore() {
    if (!cursor) {
      return;
    }
    setMorePending(true);
    const res = await fetch(
      `/api/v1/campaigns?cursor=${encodeURIComponent(cursor)}`,
      { credentials: "include" },
    );
    if (!res.ok) {
      setMorePending(false);
      return;
    }
    const data = (await res.json()) as {
      campaigns?: Campaign[];
      next_cursor?: string;
    };
    const extra = data.campaigns ?? [];
    const peopleEntries = await Promise.all(
      extra.map(async (c) => {
        const r = await fetch(`/api/v1/campaigns/${c.id}/enrolments`, {
          credentials: "include",
        });
        const body = (await r.json().catch(() => null)) as {
          enrolments?: CampaignPerson[];
        } | null;
        return [c.id, body?.enrolments ?? []] as const;
      }),
    );
    setRows((cur) => [...cur, ...extra]);
    setPeople((cur) => ({ ...cur, ...Object.fromEntries(peopleEntries) }));
    setCursor(data.next_cursor);
    setMorePending(false);
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader
        title="Campaigns"
        actions={
          <>
            <Button
              type="button"
              variant="outline"
              className="rounded-md"
              disabled={rows.length === 0 || contacts.length === 0}
              onClick={() => setAddFor(rows[0]?.id ?? null)}
            >
              <UserPlus className="size-4" strokeWidth={1.5} />
              Add person
            </Button>
            <Button
              type="button"
              className="rounded-md"
              disabled={mailboxes.length === 0}
              onClick={() => setCreateOpen(true)}
            >
              <Plus className="size-4" strokeWidth={1.5} />
              Create campaign
            </Button>
          </>
        }
      />
      <PageBody>
      <p className="mb-4 font-body-md text-muted-foreground">
        Cold and freelance campaigns need your name and postal address in{" "}
        <Link href="/settings" className="font-label-md text-secondary">
          Settings
        </Link>
        .
      </p>
      {mailboxes.length === 0 ? (
        <EmptyState
          title="Connect a mailbox first"
          description="Campaigns send from a mailbox. Add one, then come back here."
          action={
            <Link
              href="/mailboxes"
              className="inline-flex h-8 items-center rounded-md bg-primary px-2.5 font-label-md text-sm text-primary-foreground hover:bg-primary/80"
            >
              Mailboxes
            </Link>
          }
        />
      ) : (
      <div className="flex flex-col gap-3">
          {rows.length === 0 ? (
            <EmptyState
              title="No campaigns yet"
              description="A campaign is who you write and the sequence they go through."
              action={
                <Button
                  type="button"
                  className="rounded-md"
                  onClick={() => setCreateOpen(true)}
                >
                  Create campaign
                </Button>
              }
            />
          ) : (
            rows.map((c) => (
              <CampaignCard
                key={c.id}
                campaign={c}
                mailboxes={mailboxes}
                pools={pools}
                contacts={contacts}
                jobs={jobs}
                profiles={profiles}
                people={people[c.id] ?? []}
                onAdd={() => setAddFor(c.id)}
              />
            ))
          )}
          {cursor ? (
            <Button
              type="button"
              variant="outline"
              className="mt-1 w-fit rounded-md"
              pending={morePending}
              onClick={loadMore}
            >
              Show more
            </Button>
          ) : null}
      </div>
      )}
      </PageBody>
      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>New campaign</SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            <CreateCampaign
              mailboxes={mailboxes}
              pools={pools}
              profiles={profiles}
              onDone={() => setCreateOpen(false)}
            />
          </div>
        </SheetContent>
      </Sheet>
      <Dialog
        open={Boolean(addFor)}
        onOpenChange={(open) => {
          if (!open) {
            setAddFor(null);
          }
        }}
      >
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Add person to campaign</DialogTitle>
          </DialogHeader>
          {addFor ? (
            <AddPerson
              campaigns={rows}
              contacts={contacts}
              jobs={jobs}
              profiles={profiles}
              initialID={addFor}
              onDone={() => setAddFor(null)}
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function CampaignCard({
  campaign,
  mailboxes,
  pools = [],
  contacts,
  jobs,
  profiles,
  people,
  onAdd,
}: {
  campaign: Campaign;
  mailboxes: Mailbox[];
  pools?: MailboxPool[];
  contacts: Contact[];
  jobs: Job[];
  profiles: Profile[];
  people: CampaignPerson[];
  onAdd: () => void;
}) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);
  const [stopOpen, setStopOpen] = useState(false);
  const emails = campaign.steps.filter((s) => s.kind === "email").length;
  const box = mailboxes.find((m) => m.id === campaign.senders[0]?.mailbox_id);
  const group = pools.find((p) => p.id === campaign.pool_id);
  const profile = profiles.find((p) => p.id === campaign.profile_id);

  async function post(path: string) {
    setMsg("");
    setPending(true);
    const res = await fetch(path, { method: "POST", credentials: "include" });
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
    <article className="rounded-md border border-border bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border pb-4">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="font-headline-sm text-foreground">{campaign.name}</h2>
            <Pill tone="secondary">
              {intentLabel[campaign.intent] ?? campaign.intent}
            </Pill>
            <Pill tone={statusTone(campaign.status)}>
              {campaignStatusLabel[campaign.status] ?? campaign.status}
            </Pill>
          </div>
          <p className="mt-1 font-body-sm text-muted-foreground">
            {campaign.list_unsubscribe ? "Includes unsubscribe · " : ""}
            {people.length} {people.length === 1 ? "person" : "people"}
            {" · "}
            {emails} {emails === 1 ? "email" : "emails"}
            {group ? ` · ${group.name}` : box ? ` · ${box.email}` : ""}
            {profile ? ` · ${profile.name}` : ""}
          </p>
        </div>
      </div>
      <ol className="relative my-4 space-y-3 border-l border-border pl-5">
        {campaign.steps.map((step) => (
          <li key={step.position} className="relative">
            <span className="absolute top-1.5 -left-[1.45rem] size-2.5 rounded-full border-2 border-primary bg-card" />
            {step.kind === "wait" ? (
              <p className="font-body-sm text-muted-foreground">
                Wait {step.wait_days ?? 0} days
              </p>
            ) : (
              <div className="rounded-md border border-border bg-muted px-2.5 py-2">
                <p className="font-label-md text-foreground">
                  Email
                  {step.content === "ai" ? (
                    <span className="ml-2 rounded-sm bg-accent px-1.5 py-0.5 font-label-sm text-primary">
                      AI
                    </span>
                  ) : null}
                </p>
              </div>
            )}
          </li>
        ))}
      </ol>
      <div className="mb-4">
        {people.length === 0 ? (
          <p className="font-body-md text-muted-foreground">
            No one in this campaign yet.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {people.map((p) => {
              const contact = contacts.find((c) => c.id === p.contact_id);
              const job = jobs.find((j) => j.id === p.job_id);
              const profile = profiles.find((pr) => pr.id === p.profile_id);
              const who = contact ? contactName(contact) : p.contact_id;
              const bits = [
                job?.title || (job ? "Untitled job" : null),
                profile?.name,
              ].filter(Boolean);
              return (
                <li
                  key={p.id}
                  className="flex flex-wrap items-baseline justify-between gap-2 rounded-md border border-border bg-muted px-2.5 py-2"
                >
                  <div>
                    <p className="font-label-md text-foreground">{who}</p>
                    {bits.length > 0 ? (
                      <p className="font-body-sm text-muted-foreground">
                        {bits.join(" · ")}
                      </p>
                    ) : null}
                  </div>
                  <p className="font-body-sm text-muted-foreground">
                    {campaignPersonStatus[p.status] ?? p.status}
                  </p>
                </li>
              );
            })}
          </ul>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-2 border-t border-border pt-3">
        {campaign.status === "draft" || campaign.status === "paused" ? (
          <Button
            type="button"
            className="rounded-md"
            pending={pending}
            onClick={() => post(`/api/v1/campaigns/${campaign.id}/start`)}
          >
            {campaign.status === "paused" ? "Resume" : "Start"}
          </Button>
        ) : null}
        {campaign.status === "running" ? (
          <Button
            type="button"
            variant="outline"
            className="rounded-md"
            pending={pending}
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
            pending={pending}
            onClick={() => setStopOpen(true)}
          >
            Stop
          </Button>
        ) : null}
        <Button
          type="button"
          variant="outline"
          className="rounded-md"
          onClick={onAdd}
        >
          Add person
        </Button>
        {msg ? <p className="font-body-md text-muted-foreground">{msg}</p> : null}
      </div>
      <ConfirmDialog
        open={stopOpen}
        onOpenChange={setStopOpen}
        title="Stop this campaign?"
        description="No more drafts or sends for people in this campaign."
        confirmLabel="Stop campaign"
        confirmTone="danger"
        loading={pending}
        onConfirm={async () => {
          await post(`/api/v1/campaigns/${campaign.id}/stop`);
          setStopOpen(false);
        }}
      />
    </article>
  );
}

function CreateCampaign({
  mailboxes,
  pools = [],
  profiles,
  onDone,
}: {
  mailboxes: Mailbox[];
  pools?: MailboxPool[];
  profiles: Profile[];
  onDone: () => void;
}) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [intent, setIntent] = useState("job");
  const [mailboxID, setMailboxID] = useState(mailboxes[0]?.id ?? "");
  const [poolID, setPoolID] = useState("");
  const [profileID, setProfileID] = useState("");
  const [perDay, setPerDay] = useState("5");
  const [strategy, setStrategy] = useState("round_robin");
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
        mailbox_ids: poolID ? [] : mailboxID ? [mailboxID] : [],
        pool_id: poolID || undefined,
        new_contacts_per_day: Number(perDay),
        sender_strategy: strategy,
        profile_id: profileID || undefined,
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
    router.refresh();
    onDone();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="camp-name">Campaign name</Label>
        <Input
          id="camp-name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="camp-intent">Campaign type</Label>
          <select
            id="camp-intent"
            value={intent}
            onChange={(e) => setIntent(e.target.value)}
            className={fieldSelect}
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
            required={!poolID}
            disabled={Boolean(poolID)}
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
        {pools.length > 0 ? (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="camp-group">Mailbox group</Label>
            <select
              id="camp-group"
              value={poolID}
              onChange={(e) => setPoolID(e.target.value)}
              className={fieldSelect}
            >
              <option value="">One mailbox</option>
              {pools.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}
        {profiles.length > 0 ? (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="camp-profile">Profile</Label>
            <select
              id="camp-profile"
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
          <Label htmlFor="camp-per-day">New contacts per day</Label>
          <Input
            id="camp-per-day"
            type="number"
            min={1}
            required
            value={perDay}
            onChange={(e) => setPerDay(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="camp-strategy">Send strategy</Label>
          <select
            id="camp-strategy"
            value={strategy}
            onChange={(e) => setStrategy(e.target.value)}
            className={fieldSelect}
          >
            <option value="round_robin">Round robin</option>
            <option value="sequential">Sequential</option>
            <option value="capacity_first">Capacity first</option>
            <option value="weighted">Weighted</option>
            <option value="domain_spread">Domain spread</option>
            <option value="manual">Manual</option>
          </select>
        </div>
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onDone}>
          Cancel
        </Button>
        <Button type="submit" pending={pending}>
          Create
        </Button>
      </div>
    </form>
  );
}

function AddPerson({
  campaigns,
  contacts,
  jobs,
  profiles,
  initialID,
  onDone,
}: {
  campaigns: Campaign[];
  contacts: Contact[];
  jobs: Job[];
  profiles: Profile[];
  initialID: string;
  onDone: () => void;
}) {
  const router = useRouter();
  const [campaignID, setCampaignID] = useState(initialID);
  const [contactID, setContactID] = useState(contacts[0]?.id ?? "");
  const [jobID, setJobID] = useState("");
  const [profileID, setProfileID] = useState("");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch(`/api/v1/campaigns/${campaignID}/enrolments`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        contact_id: contactID,
        notes,
        job_id: jobID || undefined,
        profile_id: profileID || undefined,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(operatorError(data?.error?.message ?? "Could not add"));
      return;
    }
    router.refresh();
    onDone();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="add-campaign">Campaign</Label>
        <select
          id="add-campaign"
          value={campaignID}
          onChange={(e) => setCampaignID(e.target.value)}
          className={fieldSelect}
        >
          {campaigns.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="add-contact">Person</Label>
        <select
          id="add-contact"
          required
          value={contactID}
          onChange={(e) => setContactID(e.target.value)}
          className={fieldSelect}
        >
          {contacts.map((ct) => (
            <option key={ct.id} value={ct.id}>
              {contactName(ct)} · {ct.email}
            </option>
          ))}
        </select>
      </div>
      {jobs.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="add-job">Job</Label>
          <select
            id="add-job"
            value={jobID}
            onChange={(e) => setJobID(e.target.value)}
            className={fieldSelect}
          >
            <option value="">None</option>
            {jobs.map((j) => (
              <option key={j.id} value={j.id}>
                {j.title || "Untitled job"}
              </option>
            ))}
          </select>
        </div>
      ) : null}
      {profiles.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="add-profile">Profile</Label>
          <select
            id="add-profile"
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
        <Label htmlFor="add-notes">Why you’re writing</Label>
        <textarea
          id="add-notes"
          rows={3}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          className={fieldTextarea}
        />
        <p className="font-body-md text-muted-foreground">
          Needed unless you pick a job or the campaign already has instructions.
        </p>
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onDone}>
          Cancel
        </Button>
        <Button type="submit" pending={pending} disabled={!contactID}>
          Add to campaign
        </Button>
      </div>
    </form>
  );
}
