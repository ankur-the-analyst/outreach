"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Pill } from "@/components/page-header";
import type { Contact, NeverEmail } from "@/lib/api";
import { contactName, fieldSelect, fieldTextarea } from "@/lib/labels";

function neverEmailError(message: string): string {
  if (message.includes("reason required")) {
    return "Say why";
  }
  if (message.includes("value required")) {
    return "Enter an address or domain";
  }
  if (message.includes("duplicate") || message.includes("unique")) {
    return "Already on the never-email list";
  }
  return message;
}

type PreviewRow = {
  email: string;
  first_name?: string;
  last_name?: string;
  title?: string;
  company?: string;
  role_class?: string;
  readiness: string;
  errors: string[];
  duplicate: boolean;
  generic_mailbox: boolean;
};

export function ContactBook({
  contacts,
  nextCursor,
  blocked = [],
}: {
  contacts: Contact[];
  nextCursor?: string;
  blocked?: NeverEmail[];
}) {
  const [rows, setRows] = useState(contacts);
  const [cursor, setCursor] = useState(nextCursor);
  const [morePending, setMorePending] = useState(false);

  useEffect(() => {
    setRows(contacts);
    setCursor(nextCursor);
  }, [contacts, nextCursor]);

  async function loadMore() {
    if (!cursor) {
      return;
    }
    setMorePending(true);
    const res = await fetch(
      `/api/v1/contacts?cursor=${encodeURIComponent(cursor)}`,
      { credentials: "include" },
    );
    setMorePending(false);
    if (!res.ok) {
      return;
    }
    const data = (await res.json()) as {
      contacts?: Contact[];
      next_cursor?: string;
    };
    setRows((cur) => [...cur, ...(data.contacts ?? [])]);
    setCursor(data.next_cursor);
  }

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
      <div className="lg:col-span-7">
        {rows.length === 0 ? (
          <p className="font-body-md text-muted-foreground">No contacts yet.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {rows.map((c) => (
              <li
                key={c.id}
                className="rounded-md border border-border bg-card p-4"
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <p className="font-label-md text-foreground">
                      {contactName(c)}
                    </p>
                    <p className="mt-0.5 font-body-sm text-muted-foreground">
                      {c.email}
                      {c.title ? ` · ${c.title}` : ""}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Pill
                      tone={
                        c.readiness === "blocked"
                          ? "danger"
                          : c.readiness === "risky"
                            ? "primary"
                            : "success"
                      }
                    >
                      {c.readiness}
                    </Pill>
                    <RemoveButton id={c.id} />
                  </div>
                </div>
                <p className="mt-2 font-body-sm text-muted-foreground">
                  {c.role_class ? `${c.role_class.replaceAll("_", " ")} · ` : ""}
                  {c.provenance.source}
                </p>
              </li>
            ))}
          </ul>
        )}
        {cursor ? (
          <Button
            type="button"
            variant="outline"
            className="mt-3 w-fit rounded-md"
            pending={morePending}
            onClick={loadMore}
          >
            Show more
          </Button>
        ) : null}
        <NeverEmailList blocked={blocked} />
      </div>
      <div className="flex flex-col gap-4 lg:col-span-5">
        <CreateForm />
        <NeverEmailForm />
        <CSVPreview />
      </div>
    </div>
  );
}

function RemoveButton({ id }: { id: string }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  async function onRemove() {
    setPending(true);
    const res = await fetch(`/api/v1/contacts/${id}/erase`, {
      method: "POST",
      credentials: "include",
    });
    setPending(false);
    if (!res.ok) {
      return;
    }
    router.refresh();
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="xs"
      className="rounded-md"
      pending={pending}
      onClick={onRemove}
    >
      Remove
    </Button>
  );
}

function CreateForm() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [company, setCompany] = useState("");
  const [source, setSource] = useState("typed");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/contacts", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email,
        first_name: firstName,
        company,
        provenance: { source },
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not save contact");
      return;
    }
    setEmail("");
    setFirstName("");
    setCompany("");
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">Add a contact</h2>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="contact-email">Email</Label>
        <Input
          id="contact-email"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="contact-first">First name</Label>
        <Input
          id="contact-first"
          value={firstName}
          onChange={(e) => setFirstName(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="contact-company">Company</Label>
        <Input
          id="contact-company"
          value={company}
          onChange={(e) => setCompany(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="contact-source">How you found them</Label>
        <Input
          id="contact-source"
          required
          value={source}
          onChange={(e) => setSource(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Save
      </Button>
    </form>
  );
}

function NeverEmailList({ blocked }: { blocked: NeverEmail[] }) {
  return (
    <div className="mt-8">
      <h2 className="mb-3 font-headline-md text-foreground">Never email</h2>
      {blocked.length === 0 ? (
        <p className="font-body-md text-muted-foreground">
          No blocked addresses yet.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {blocked.map((row) => (
            <li
              key={`${row.kind}:${row.value ?? ""}:${row.created_at ?? row.reason}`}
              className="rounded-md border border-border bg-card p-4"
            >
              <p className="font-label-md text-foreground">
                {row.value || "Removed contact"}
              </p>
              <p className="mt-1 font-body-sm text-muted-foreground">
                {row.kind === "domain" ? "Whole domain" : "This address"}
                {row.reason ? ` · ${row.reason}` : ""}
              </p>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function NeverEmailForm() {
  const router = useRouter();
  const [kind, setKind] = useState("email");
  const [value, setValue] = useState("");
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setDone("");
    setPending(true);
    const res = await fetch("/api/v1/suppression", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ kind, value, reason }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(neverEmailError(data?.error?.message ?? "Could not save"));
      return;
    }
    const data = (await res.json()) as { value?: string };
    setDone(`Won't send to ${data.value || value}`);
    setValue("");
    setReason("");
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">Never email</h2>
      <p className="font-body-md text-muted-foreground">
        Blocks new sends to this address or whole domain.
      </p>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="never-kind">Block</Label>
        <select
          id="never-kind"
          value={kind}
          onChange={(e) => setKind(e.target.value)}
          className={fieldSelect}
        >
          <option value="email">This address</option>
          <option value="domain">This domain</option>
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="never-value">
          {kind === "domain" ? "Domain" : "Email"}
        </Label>
        <Input
          id="never-value"
          required
          type={kind === "domain" ? "text" : "email"}
          value={value}
          onChange={(e) => setValue(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="never-reason">Why</Label>
        <Input
          id="never-reason"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      {done ? <p className="font-body-md text-muted-foreground">{done}</p> : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Block sends
      </Button>
    </form>
  );
}

function CSVPreview() {
  const router = useRouter();
  const [csv, setCsv] = useState("email,first_name,company\n");
  const [source, setSource] = useState("csv");
  const [rows, setRows] = useState<PreviewRow[]>([]);
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  const [pending, setPending] = useState(false);

  async function onPreview(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setDone("");
    setPending(true);
    const res = await fetch("/api/v1/contacts/preview", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ csv }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Preview failed");
      return;
    }
    const data = (await res.json()) as { rows: PreviewRow[] };
    setRows(data.rows ?? []);
  }

  async function onImport() {
    setError("");
    setDone("");
    setPending(true);
    const res = await fetch("/api/v1/contacts/import", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        csv,
        provenance: { source },
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Import failed");
      return;
    }
    const data = (await res.json()) as { created?: number; skipped?: number };
    setRows([]);
    setDone(`Added ${data.created ?? 0}, skipped ${data.skipped ?? 0}`);
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onPreview}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">CSV preview</h2>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="contact-csv">CSV</Label>
        <textarea
          id="contact-csv"
          required
          rows={5}
          value={csv}
          onChange={(e) => setCsv(e.target.value)}
          className={fieldTextarea}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="csv-source">How you found them</Label>
        <Input
          id="csv-source"
          required
          value={source}
          onChange={(e) => setSource(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      {done ? <p className="font-body-md text-muted-foreground">{done}</p> : null}
      <div className="flex flex-wrap gap-3">
        <Button type="submit" className="rounded-md" pending={pending}>
          Preview
        </Button>
        {rows.length > 0 ? (
          <Button
            type="button"
            variant="outline"
            className="rounded-md"
            pending={pending}
            onClick={onImport}
          >
            Import valid rows
          </Button>
        ) : null}
      </div>
      {rows.length > 0 ? (
        <ul className="flex flex-col gap-2">
          {rows.map((row, i) => (
            <li key={`${row.email}-${i}`} className="font-body-md text-muted-foreground">
              {row.email || "(missing email)"} · {row.readiness}
              {row.generic_mailbox ? " · generic mailbox" : ""}
              {row.errors.length > 0 ? ` · ${row.errors.join(", ")}` : ""}
            </li>
          ))}
        </ul>
      ) : null}
    </form>
  );
}
