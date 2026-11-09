"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState, PageBody, PageHeader, Pill } from "@/components/page-header";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { Contact, NeverEmail } from "@/lib/api";
import { contactName, fieldSelect } from "@/lib/labels";

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
  const [sheet, setSheet] = useState<"add" | "never" | "csv" | null>(null);

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
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader
        title="Contacts"
        actions={
          <>
            <Button
              type="button"
              variant="outline"
              className="rounded-md"
              onClick={() => setSheet("never")}
            >
              Never email
            </Button>
            <Button
              type="button"
              variant="outline"
              className="rounded-md"
              onClick={() => setSheet("csv")}
            >
              Import CSV
            </Button>
            <Button
              type="button"
              className="rounded-md"
              onClick={() => setSheet("add")}
            >
              Add contact
            </Button>
          </>
        }
      />
      <PageBody>
        {rows.length === 0 ? (
          <EmptyState
            title="No contacts yet"
            description="Add one person, or import a CSV with email, first_name, and company."
            action={
              <>
                <Button
                  type="button"
                  className="rounded-md"
                  onClick={() => setSheet("add")}
                >
                  Add contact
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  className="rounded-md"
                  onClick={() => setSheet("csv")}
                >
                  Import CSV
                </Button>
              </>
            }
          />
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
      </PageBody>
      <Sheet
        open={sheet !== null}
        onOpenChange={(open) => {
          if (!open) {
            setSheet(null);
          }
        }}
      >
        <SheetContent>
          <SheetHeader>
            <SheetTitle>
              {sheet === "never"
                ? "Never email"
                : sheet === "csv"
                  ? "Import CSV"
                  : "Add a contact"}
            </SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            {sheet === "add" ? <CreateForm /> : null}
            {sheet === "never" ? <NeverEmailForm /> : null}
            {sheet === "csv" ? <CSVPreview /> : null}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function RemoveButton({ id }: { id: string }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [open, setOpen] = useState(false);

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
    setOpen(false);
    router.refresh();
  }

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="xs"
        className="rounded-md"
        pending={pending}
        onClick={() => setOpen(true)}
      >
        Remove
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title="Remove this person?"
        description="Deletes their messages and drafts. They stay on the never-email list."
        confirmLabel="Remove"
        confirmTone="danger"
        loading={pending}
        onConfirm={onRemove}
      />
    </>
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
      className="flex flex-col gap-4"
    >
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
      className="flex flex-col gap-4"
    >
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

function isCSVFile(file: File): boolean {
  const name = file.name.toLowerCase();
  return name.endsWith(".csv") || file.type === "text/csv" || file.type === "text/plain";
}

function CSVPreview() {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [csv, setCsv] = useState("");
  const [fileName, setFileName] = useState("");
  const [source, setSource] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const [rows, setRows] = useState<PreviewRow[]>([]);
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  const [pending, setPending] = useState(false);

  async function takeFile(file: File | undefined) {
    if (!file) {
      return;
    }
    if (!isCSVFile(file)) {
      setError("Choose a .csv file");
      return;
    }
    const text = await file.text();
    setCsv(text);
    setFileName(file.name);
    setRows([]);
    setError("");
    setDone("");
  }

  async function onPreview(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setDone("");
    if (!csv.trim()) {
      setError("Choose a CSV file first");
      return;
    }
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
    setCsv("");
    setFileName("");
    if (inputRef.current) {
      inputRef.current.value = "";
    }
    setDone(`Added ${data.created ?? 0}, skipped ${data.skipped ?? 0}`);
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onPreview}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="contact-csv">CSV file</Label>
        <input
          ref={inputRef}
          id="contact-csv"
          type="file"
          accept=".csv,text/csv"
          className="sr-only"
          onChange={(e) => void takeFile(e.target.files?.[0])}
        />
        <div
          onDragOver={(e) => {
            e.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragOver(false);
            void takeFile(e.dataTransfer.files[0]);
          }}
          className={`flex flex-col gap-3 rounded-md border border-dashed p-4 ${
            dragOver ? "border-primary bg-accent" : "border-border bg-muted"
          }`}
        >
          <p className="font-body-md text-muted-foreground">
            {fileName
              ? fileName
              : "Drop a CSV here, or choose a file. Needs email, first_name, company columns."}
          </p>
          <Button
            type="button"
            variant="outline"
            className="w-fit rounded-md"
            onClick={() => inputRef.current?.click()}
          >
            Choose file
          </Button>
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="csv-source">How you found them</Label>
        <Input
          id="csv-source"
          required
          placeholder="Apollo list, conference, referral…"
          value={source}
          onChange={(e) => setSource(e.target.value)}
        />
      </div>
      {error ? (
        <p role="alert" className="font-body-md text-destructive">
          {error}
        </p>
      ) : null}
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
