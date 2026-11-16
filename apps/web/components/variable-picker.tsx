"use client";

import { useEffect, useState } from "react";
import { Braces } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export type TemplateVar = {
  group: string;
  label: string;
  value: string;
};

type FieldDef = {
  key: string;
  label: string;
  builtin?: boolean;
};

const personFallback: TemplateVar[] = [
  { group: "Person", label: "First name", value: "{{contact.first_name}}" },
  { group: "Person", label: "Last name", value: "{{contact.last_name}}" },
  { group: "Person", label: "Email", value: "{{contact.email}}" },
  { group: "Person", label: "Title", value: "{{contact.title}}" },
];

const youFallback: TemplateVar[] = [
  { group: "You", label: "Name", value: "{{user.name}}" },
  { group: "You", label: "Email", value: "{{user.email}}" },
];

const sendVars: TemplateVar[] = [
  { group: "This send", label: "Company", value: "{{company.name}}" },
  { group: "This send", label: "Job title", value: "{{job.title}}" },
  { group: "This send", label: "Profile", value: "{{profile.name}}" },
];

export function useTemplateVars(): TemplateVar[] {
  const [vars, setVars] = useState<TemplateVar[]>(sendVars);

  useEffect(() => {
    void (async () => {
      const [contact, member] = await Promise.all([
        loadFields("contact"),
        loadFields("member"),
      ]);
      const person =
        contact.length > 0
          ? contact.map((f) => fieldVar("Person", "contact", f))
          : personFallback;
      const you =
        member.length > 0
          ? member
              .filter((f) => !(f.builtin && f.key === "role"))
              .map((f) => fieldVar("You", "user", f))
          : youFallback;
      setVars([...person, ...you, ...sendVars]);
    })();
  }, []);

  return vars;
}

function fieldVar(group: string, prefix: string, field: FieldDef): TemplateVar {
  return {
    group,
    label: field.label,
    value: `{{${prefix}.${field.key}}}`,
  };
}

async function loadFields(object: string): Promise<FieldDef[]> {
  const res = await fetch(`/api/v1/fields?object=${object}&catalog=1`, {
    credentials: "include",
  });
  if (!res.ok) {
    return [];
  }
  const data = (await res.json()) as { fields?: FieldDef[] };
  return data.fields ?? [];
}

export function VariablePicker({
  vars,
  onInsert,
}: {
  vars: TemplateVar[];
  onInsert: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const q = search.trim().toLowerCase();
  const shown = vars.filter(
    (v) =>
      !q ||
      v.label.toLowerCase().includes(q) ||
      v.value.toLowerCase().includes(q),
  );
  const groups = [...new Set(shown.map((v) => v.group))];

  return (
    <div className="relative">
      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-expanded={open}
        onMouseDown={(e) => e.preventDefault()}
        onClick={() => setOpen((v) => !v)}
      >
        <Braces />
        Insert field
      </Button>
      {open ? (
        <div className="absolute right-0 z-30 mt-1 flex max-h-72 w-72 max-w-[85vw] flex-col overflow-hidden rounded-md border border-border bg-card shadow-sm">
          <div className="border-b border-border p-2">
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search fields"
              aria-label="Search fields"
            />
          </div>
          <div className="overflow-y-auto p-1">
            {groups.length === 0 ? (
              <p className="px-2 py-3 font-body-sm text-muted-foreground">No matching fields.</p>
            ) : (
              groups.map((group) => (
                <div key={group} className="mb-1">
                  <p className="px-2 py-1 font-label-sm text-muted-foreground">{group}</p>
                  {shown
                    .filter((v) => v.group === group)
                    .map((v) => (
                      <button
                        key={v.value}
                        type="button"
                        className="flex w-full flex-col items-start rounded-md px-2 py-1.5 text-left hover:bg-muted"
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => {
                          onInsert(v.value);
                          setOpen(false);
                          setSearch("");
                        }}
                      >
                        <span className="font-label-md text-foreground">{v.label}</span>
                        <span className="font-body-sm text-muted-foreground">{v.value}</span>
                      </button>
                    ))}
                </div>
              ))
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}
