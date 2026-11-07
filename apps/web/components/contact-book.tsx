"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Contact } from "@/lib/api";

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

export function ContactBook({ contacts }: { contacts: Contact[] }) {
  return (
    <div className="flex flex-col gap-8">
      {contacts.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No contacts yet.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {contacts.map((c) => (
            <li
              key={c.id}
              className="rounded-md border border-border bg-card p-4"
            >
              <p className="font-label-md text-foreground">{c.email}</p>
              <p className="mt-1 font-body-md text-muted-foreground">
                {[c.first_name, c.last_name].filter(Boolean).join(" ") || "—"}
                {c.title ? ` · ${c.title}` : ""}
                {` · ${c.readiness}`}
                {c.role_class ? ` · ${c.role_class}` : ""}
                {` · ${c.provenance.source}`}
              </p>
            </li>
          ))}
        </ul>
      )}
      <CreateForm />
      <CSVPreview />
    </div>
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
      onSubmit={onSubmit}
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
        <Label htmlFor="contact-source">Provenance source</Label>
        <Input
          id="contact-source"
          required
          value={source}
          onChange={(e) => setSource(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" disabled={pending}>
        {pending ? "Saving…" : "Save"}
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
  const [pending, setPending] = useState(false);

  async function onPreview(e: React.FormEvent) {
    e.preventDefault();
    setError("");
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
    setPending(true);
    for (const row of rows) {
      if (row.errors.length > 0 || row.duplicate) {
        continue;
      }
      const res = await fetch("/api/v1/contacts", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: row.email,
          first_name: row.first_name,
          last_name: row.last_name,
          title: row.title,
          company: row.company,
          provenance: { source },
        }),
      });
      if (!res.ok && res.status !== 409) {
        const data = (await res.json().catch(() => null)) as
          | { error?: { message?: string } }
          | null;
        setPending(false);
        setError(data?.error?.message ?? "Import failed");
        return;
      }
    }
    setPending(false);
    setRows([]);
    router.refresh();
  }

  return (
    <form
      onSubmit={onPreview}
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
          className="w-full rounded-md border border-input bg-card px-2.5 py-2 font-[family-name:var(--font-body)] text-sm text-foreground outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="csv-source">Provenance source</Label>
        <Input
          id="csv-source"
          required
          value={source}
          onChange={(e) => setSource(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <div className="flex flex-wrap gap-3">
        <Button type="submit" className="rounded-md" disabled={pending}>
          {pending ? "…" : "Preview"}
        </Button>
        {rows.length > 0 ? (
          <Button
            type="button"
            variant="outline"
            className="rounded-md"
            disabled={pending}
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
