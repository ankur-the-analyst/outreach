"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Panel } from "@/components/page-header";
import { fieldSelect } from "@/lib/labels";

type Def = {
  id?: string;
  key: string;
  label: string;
  kind?: string;
  builtin?: boolean;
  options?: string[];
};

const kinds = [
  ["text", "Text"],
  ["number", "Number"],
  ["url", "Link"],
  ["date", "Date"],
  ["boolean", "Yes / no"],
  ["select", "Dropdown"],
] as const;

export function FieldCatalog({
  object,
  title,
  description,
  values,
}: {
  object: "contact" | "member";
  title: string;
  description: string;
  values?: boolean;
}) {
  const [defs, setDefs] = useState<Def[]>([]);
  const [label, setLabel] = useState("");
  const [kind, setKind] = useState<string>("text");
  const [choices, setChoices] = useState("");
  const [mine, setMine] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    void (async () => {
      const res = await fetch(`/api/v1/fields?object=${object}&catalog=1`, {
        credentials: "include",
      });
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as { fields?: Def[] };
      setDefs(data.fields ?? []);
      if (!values) {
        return;
      }
      const mineRes = await fetch("/api/v1/fields/mine", {
        credentials: "include",
      });
      if (!mineRes.ok) {
        return;
      }
      const mineData = (await mineRes.json()) as {
        values?: Record<string, string>;
      };
      setMine(mineData.values ?? {});
    })();
  }, [object, values]);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    const key = label
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, "_")
      .replace(/^_|_$/g, "")
      .slice(0, 40);
    if (!key) {
      setError("Use a name with letters or numbers.");
      return;
    }
    const options =
      kind === "select"
        ? choices
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean)
        : [];
    if (kind === "select" && options.length === 0) {
      setError("Add at least one choice, separated by commas.");
      return;
    }
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/fields", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ object, key, label: label.trim(), kind, options }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not add field");
      return;
    }
    const created = (await res.json()) as Def;
    setDefs((cur) => [...cur, created]);
    setLabel("");
    setChoices("");
  }

  async function saveValues(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/fields/mine", {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ values: mine }),
    });
    setPending(false);
    if (!res.ok) {
      setError("Could not save");
    }
  }

  return (
    <Panel title={title} description={description}>
      {defs.length > 0 ? (
        <ul className="mb-3 flex flex-col gap-1">
          {defs.map((d) => (
            <li key={d.key} className="font-body-md text-foreground">
              {d.label}
              <span className="ml-2 font-body-sm text-muted-foreground">
                {object === "member" ? `{{user.${d.key}}}` : `{{contact.${d.key}}}`}
                {d.builtin ? " · Built in" : ""}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mb-3 font-body-sm text-muted-foreground">None yet.</p>
      )}
      <form onSubmit={add} className="flex flex-wrap items-end gap-2">
        <div className="flex min-w-40 flex-1 flex-col gap-1.5">
          <Label htmlFor={`${object}-field`}>New field</Label>
          <Input
            id={`${object}-field`}
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder={object === "member" ? "Signature" : "Why this person"}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={`${object}-kind`}>Type</Label>
          <select
            id={`${object}-kind`}
            value={kind}
            onChange={(e) => setKind(e.target.value)}
            className={fieldSelect}
          >
            {kinds.map(([value, name]) => (
              <option key={value} value={value}>
                {name}
              </option>
            ))}
          </select>
        </div>
        {kind === "select" ? (
          <div className="flex min-w-40 flex-1 flex-col gap-1.5">
            <Label htmlFor={`${object}-choices`}>Choices</Label>
            <Input
              id={`${object}-choices`}
              value={choices}
              onChange={(e) => setChoices(e.target.value)}
              placeholder="Low, Medium, High"
            />
          </div>
        ) : null}
        <Button type="submit" variant="outline" className="rounded-md" pending={pending}>
          Add field
        </Button>
      </form>
      {values && defs.some((d) => !d.builtin) ? (
        <form onSubmit={saveValues} className="mt-4 flex flex-col gap-3">
          {defs
            .filter((d) => !d.builtin)
            .map((d) => (
            <div key={d.key} className="flex flex-col gap-1.5">
              <Label htmlFor={`mine-${d.key}`}>{d.label}</Label>
              {d.kind === "select" && d.options && d.options.length > 0 ? (
                <select
                  id={`mine-${d.key}`}
                  value={mine[d.key] ?? ""}
                  onChange={(e) => setMine((cur) => ({ ...cur, [d.key]: e.target.value }))}
                  className={fieldSelect}
                >
                  <option value="">Choose</option>
                  {d.options.map((opt) => (
                    <option key={opt} value={opt}>
                      {opt}
                    </option>
                  ))}
                </select>
              ) : d.kind === "boolean" ? (
                <select
                  id={`mine-${d.key}`}
                  value={mine[d.key] ?? ""}
                  onChange={(e) => setMine((cur) => ({ ...cur, [d.key]: e.target.value }))}
                  className={fieldSelect}
                >
                  <option value="">Choose</option>
                  <option value="yes">Yes</option>
                  <option value="no">No</option>
                </select>
              ) : (
                <Input
                  id={`mine-${d.key}`}
                  type={d.kind === "number" ? "number" : d.kind === "date" ? "date" : "text"}
                  value={mine[d.key] ?? ""}
                  onChange={(e) => setMine((cur) => ({ ...cur, [d.key]: e.target.value }))}
                />
              )}
            </div>
          ))}
          <Button type="submit" className="w-fit rounded-md" pending={pending}>
            Save
          </Button>
        </form>
      ) : null}
      {error ? (
        <p role="alert" className="mt-2 font-body-sm text-destructive">
          {error}
        </p>
      ) : null}
    </Panel>
  );
}
