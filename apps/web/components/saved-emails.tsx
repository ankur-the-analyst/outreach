"use client";

import { useEffect, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState, PageBody, PageHeader } from "@/components/page-header";
import type { SavedEmail } from "@/lib/api";
import {
  type BlockType,
  type EmailDesign,
  type EmailElement,
  blockGroups,
  blockLabel,
  generateHTML,
  generateText,
  mergeTags,
  newBlock,
  parseDesign,
} from "@/lib/email-design";
import { fieldSelect, fieldTextarea } from "@/lib/labels";

export function SavedEmails({ emails }: { emails: SavedEmail[] }) {
  const [rows, setRows] = useState(emails);
  const [editing, setEditing] = useState<string | "new" | null>(null);

  useEffect(() => {
    setRows(emails);
  }, [emails]);

  const current = editing && editing !== "new" ? rows.find((e) => e.id === editing) : null;

  return (
    <>
      <PageHeader
        title="Email templates"
        actions={
          <Button type="button" onClick={() => setEditing("new")}>
            New template
          </Button>
        }
      />
      <PageBody>
        {editing ? (
          <TemplateBuilder
            key={editing}
            initial={current ?? undefined}
            onCancel={() => setEditing(null)}
            onSaved={(saved) => {
              setRows((cur) => {
                const i = cur.findIndex((e) => e.id === saved.id);
                if (i >= 0) {
                  const next = [...cur];
                  next[i] = saved;
                  return next;
                }
                return [saved, ...cur];
              });
              setEditing(saved.id);
            }}
          />
        ) : rows.length === 0 ? (
          <EmptyState
            title="No email templates yet"
            description="Build the email here, then pick it when you start a campaign."
            action={
              <Button type="button" onClick={() => setEditing("new")}>
                New template
              </Button>
            }
          />
        ) : (
          <ul className="flex flex-col gap-2">
            {rows.map((e) => (
              <li key={e.id}>
                <button
                  type="button"
                  className="flex w-full flex-col items-start gap-0.5 rounded-md border border-border bg-card px-4 py-3 text-left hover:bg-muted"
                  onClick={() => setEditing(e.id)}
                >
                  <span className="font-headline-sm text-foreground">{e.name}</span>
                  <span className="font-body-sm text-muted-foreground">{e.subject}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </PageBody>
    </>
  );
}

function TemplateBuilder({
  initial,
  onCancel,
  onSaved,
}: {
  initial?: SavedEmail;
  onCancel: () => void;
  onSaved: (e: SavedEmail) => void;
}) {
  const [name, setName] = useState(initial?.name ?? "");
  const [subject, setSubject] = useState(initial?.subject ?? "Hi {{contact.first_name}}");
  const [design, setDesign] = useState<EmailDesign>(() =>
    parseDesign(initial?.design_json, initial?.body_text),
  );
  const [selected, setSelected] = useState<string | null>(design.elements[0]?.id ?? null);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const [preview, setPreview] = useState(false);
  const [showCode, setShowCode] = useState(false);

  const selectedEl = findElement(design.elements, selected);

  function add(type: BlockType) {
    const el = newBlock(type);
    setDesign((d) => ({ ...d, elements: [...d.elements, el] }));
    setSelected(el.id);
  }

  function addToColumn(rowID: string, column: number, type: BlockType) {
    const el = newBlock(type);
    setDesign((d) => ({
      ...d,
      elements: mapElements(d.elements, (node) => {
        if (node.id !== rowID) {
          return node;
        }
        const children = [...(node.children ?? [])];
        while (children.length <= column) {
          children.push([]);
        }
        children[column] = [...children[column], el];
        return { ...node, children };
      }),
    }));
    setSelected(el.id);
  }

  function patch(id: string, next: Partial<EmailElement>) {
    setDesign((d) => ({
      ...d,
      elements: mapElements(d.elements, (node) =>
        node.id === id
          ? { ...node, ...next, props: { ...node.props, ...(next.props ?? {}) } }
          : node,
      ),
    }));
  }

  function setColumns(id: string, count: number) {
    const n = Math.min(4, Math.max(1, count));
    setDesign((d) => ({
      ...d,
      elements: mapElements(d.elements, (node) => {
        if (node.id !== id) {
          return node;
        }
        const children = [...(node.children ?? [])];
        while (children.length < n) {
          children.push([]);
        }
        return {
          ...node,
          props: { ...node.props, columns: String(n) },
          children: children.slice(0, n),
        };
      }),
    }));
  }

  function remove(id: string) {
    setDesign((d) => ({ ...d, elements: removeElement(d.elements, id) }));
    setSelected((cur) => (cur === id ? null : cur));
  }

  async function save() {
    setError("");
    setPending(true);
    const body_html = generateHTML(design);
    const body_text = generateText(design);
    const payload = {
      name,
      subject,
      body_text: body_text || ".",
      body_html,
      design_json: design,
    };
    const url = initial?.id ? `/api/v1/templates/${initial.id}` : "/api/v1/templates";
    const res = await fetch(url, {
      method: initial?.id ? "PUT" : "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as { error?: { message?: string } } | null;
      setError(data?.error?.message ?? "Could not save this template");
      return;
    }
    onSaved((await res.json()) as SavedEmail);
  }

  const html = generateHTML(design);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-48 flex-1 flex-col gap-1.5">
          <Label htmlFor="email-name">Name</Label>
          <Input id="email-name" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="flex min-w-48 flex-1 flex-col gap-1.5">
          <Label htmlFor="email-subject">Subject</Label>
          <Input id="email-subject" value={subject} onChange={(e) => setSubject(e.target.value)} />
        </div>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" onClick={() => setPreview(true)}>
            Preview
          </Button>
          <Button type="button" variant="outline" onClick={() => setShowCode((v) => !v)}>
            {showCode ? "Hide HTML" : "HTML"}
          </Button>
          <Button type="button" variant="outline" onClick={onCancel}>
            Close
          </Button>
          <Button type="button" pending={pending} onClick={() => void save()}>
            Save template
          </Button>
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="email-layout">Layout</Label>
        <select
          id="email-layout"
          className={`${fieldSelect} max-w-xs`}
          value={design.settings.layout}
          onChange={(e) =>
            setDesign((d) => ({
              ...d,
              settings: { ...d.settings, layout: e.target.value },
            }))
          }
        >
          <option value="boxed">Boxed</option>
          <option value="plain">Plain</option>
        </select>
      </div>
      <p className="font-body-sm text-muted-foreground">
        Add a block, then edit it on the right. Placeholders like {"{{contact.first_name}}"} are filled when it sends.
      </p>
      <div className="flex flex-wrap gap-1.5">
        {mergeTags.map((tag) => (
          <button
            key={tag}
            type="button"
            className="rounded-full border border-border bg-card px-2 py-0.5 font-label-sm text-muted-foreground hover:bg-muted"
            onClick={() => {
              if (!selectedEl) {
                return;
              }
              patch(selectedEl.id, { content: `${selectedEl.content} {{${tag}}}` });
            }}
          >
            {`{{${tag}}}`}
          </button>
        ))}
      </div>
      <div className="grid items-start gap-4 lg:grid-cols-[220px_minmax(0,1fr)_260px]">
        <aside className="flex flex-col gap-4 rounded-md border border-border bg-card p-3">
          {blockGroups.map((group) => (
            <div key={group.id} className="flex flex-col gap-1.5">
              <p className="font-label-sm text-muted-foreground">{group.label}</p>
              {group.types.map((type) => (
                <Button
                  key={type}
                  type="button"
                  variant="outline"
                  className="justify-start rounded-md"
                  onClick={() => add(type)}
                >
                  {blockLabel[type]}
                </Button>
              ))}
            </div>
          ))}
        </aside>
        <div className="min-h-80 rounded-md border border-border bg-muted p-4">
          {design.elements.length === 0 ? (
            <p className="font-body-md text-muted-foreground">Add a block to start the email.</p>
          ) : (
            <div
              className="mx-auto flex max-w-[600px] flex-col gap-2 bg-card p-4"
              style={{ backgroundColor: design.settings.contentBackgroundColor }}
            >
              {design.elements.map((el) => (
                <CanvasBlock
                  key={el.id}
                  element={el}
                  selected={selected}
                  onSelect={setSelected}
                  onAddToColumn={addToColumn}
                />
              ))}
            </div>
          )}
        </div>
        <aside className="rounded-md border border-border bg-card p-3">
          {selectedEl ? (
            <Properties
              element={selectedEl}
              onChange={(next) => patch(selectedEl.id, next)}
              onColumns={(n) => setColumns(selectedEl.id, n)}
              onRemove={() => remove(selectedEl.id)}
            />
          ) : (
            <p className="font-body-sm text-muted-foreground">Select a block to edit it.</p>
          )}
        </aside>
      </div>
      {error ? (
        <p className="font-body-md text-destructive" role="alert">
          {error}
        </p>
      ) : null}
      {showCode ? (
        <pre className="max-h-64 overflow-auto rounded-md border border-border bg-card p-3 font-body-sm text-foreground">
          {html || "Nothing to show yet."}
        </pre>
      ) : null}
      <Dialog open={preview} onOpenChange={setPreview}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{subject || "Preview"}</DialogTitle>
          </DialogHeader>
          <iframe
            title="Template preview"
            sandbox=""
            srcDoc={html}
            className="h-[480px] w-full rounded-md border border-border bg-card"
          />
        </DialogContent>
      </Dialog>
    </div>
  );
}

function CanvasBlock({
  element,
  selected,
  onSelect,
  onAddToColumn,
}: {
  element: EmailElement;
  selected: string | null;
  onSelect: (id: string) => void;
  onAddToColumn: (rowID: string, column: number, type: BlockType) => void;
}) {
  const active = element.id === selected;
  return (
    <div className="flex flex-col gap-2">
      <button
        type="button"
        onClick={() => onSelect(element.id)}
        className={`rounded-md border px-3 py-2 text-left ${
          active ? "border-primary bg-accent" : "border-border bg-card hover:bg-muted"
        }`}
      >
        <span className="font-label-sm text-muted-foreground">{blockLabel[element.type]}</span>
        <span className="mt-1 block whitespace-pre-wrap break-words font-body-sm text-foreground">
          {element.content || element.props.src || element.props.alt || "Empty"}
        </span>
      </button>
      {element.type === "row" ? (
        <div
          className="grid gap-2"
          style={{
            gridTemplateColumns: `repeat(${Math.min(4, Math.max(1, Number(element.props.columns) || 2))}, minmax(0, 1fr))`,
          }}
        >
          {(element.children ?? []).map((col, i) => (
            <div key={i} className="flex flex-col gap-2 rounded-md border border-dashed border-border p-2">
              {col.map((child) => (
                <CanvasBlock
                  key={child.id}
                  element={child}
                  selected={selected}
                  onSelect={onSelect}
                  onAddToColumn={onAddToColumn}
                />
              ))}
              <Button
                type="button"
                variant="outline"
                className="rounded-md"
                onClick={() => onAddToColumn(element.id, i, "text")}
              >
                Add text
              </Button>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function Properties({
  element,
  onChange,
  onColumns,
  onRemove,
}: {
  element: EmailElement;
  onChange: (next: Partial<EmailElement>) => void;
  onColumns: (n: number) => void;
  onRemove: () => void;
}) {
  const p = element.props;
  const prop = (key: string, value: string) => onChange({ props: { [key]: value } });
  return (
    <div className="flex flex-col gap-3">
      <p className="font-label-md text-foreground">{blockLabel[element.type]}</p>
      {element.type !== "divider" &&
      element.type !== "spacer" &&
      element.type !== "image" &&
      element.type !== "logo" &&
      element.type !== "video" &&
      element.type !== "social" ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="block-content">Content</Label>
          <textarea
            id="block-content"
            rows={4}
            className={fieldTextarea}
            value={element.content}
            onChange={(e) => onChange({ content: e.target.value })}
          />
        </div>
      ) : null}
      {element.type === "heading" ? (
        <Field label="Level">
          <select
            className={fieldSelect}
            value={p.level}
            onChange={(e) => prop("level", e.target.value)}
          >
            <option value="1">1</option>
            <option value="2">2</option>
            <option value="3">3</option>
            <option value="4">4</option>
          </select>
        </Field>
      ) : null}
      {"align" in p ? (
        <Field label="Align">
          <select
            className={fieldSelect}
            value={p.align}
            onChange={(e) => prop("align", e.target.value)}
          >
            <option value="left">Left</option>
            <option value="center">Center</option>
            <option value="right">Right</option>
          </select>
        </Field>
      ) : null}
      {"href" in p ? (
        <Field label="Link">
          <Input value={p.href} onChange={(e) => prop("href", e.target.value)} />
        </Field>
      ) : null}
      {"src" in p ? (
        <Field label="Image address">
          <Input value={p.src} onChange={(e) => prop("src", e.target.value)} />
        </Field>
      ) : null}
      {"thumbnail" in p ? (
        <Field label="Thumbnail address">
          <Input value={p.thumbnail} onChange={(e) => prop("thumbnail", e.target.value)} />
        </Field>
      ) : null}
      {"alt" in p ? (
        <Field label="Alt text">
          <Input value={p.alt} onChange={(e) => prop("alt", e.target.value)} />
        </Field>
      ) : null}
      {element.type === "list" ? (
        <Field label="List style">
          <select
            className={fieldSelect}
            value={p.type}
            onChange={(e) => prop("type", e.target.value)}
          >
            <option value="unordered">Bullets</option>
            <option value="ordered">Numbers</option>
          </select>
        </Field>
      ) : null}
      {element.type === "row" ? (
        <Field label="Columns">
          <Input
            type="number"
            min={1}
            max={4}
            value={p.columns}
            onChange={(e) => onColumns(Number(e.target.value))}
          />
        </Field>
      ) : null}
      {"color" in p ? (
        <Field label="Color">
          <Input value={p.color} onChange={(e) => prop("color", e.target.value)} />
        </Field>
      ) : null}
      {"backgroundColor" in p ? (
        <Field label="Background">
          <Input
            value={p.backgroundColor}
            onChange={(e) => prop("backgroundColor", e.target.value)}
          />
        </Field>
      ) : null}
      {"textColor" in p ? (
        <Field label="Text color">
          <Input value={p.textColor} onChange={(e) => prop("textColor", e.target.value)} />
        </Field>
      ) : null}
      <Button type="button" variant="outline" className="w-fit rounded-md" onClick={onRemove}>
        Remove block
      </Button>
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

function findElement(elements: EmailElement[], id: string | null): EmailElement | null {
  if (!id) {
    return null;
  }
  for (const el of elements) {
    if (el.id === id) {
      return el;
    }
    for (const col of el.children ?? []) {
      const found = findElement(col, id);
      if (found) {
        return found;
      }
    }
  }
  return null;
}

function mapElements(
  elements: EmailElement[],
  fn: (el: EmailElement) => EmailElement,
): EmailElement[] {
  return elements.map((el) => {
    const next = fn(el);
    if (!next.children) {
      return next;
    }
    return {
      ...next,
      children: next.children.map((col) => mapElements(col, fn)),
    };
  });
}

function removeElement(elements: EmailElement[], id: string): EmailElement[] {
  return elements
    .filter((el) => el.id !== id)
    .map((el) =>
      el.children
        ? { ...el, children: el.children.map((col) => removeElement(col, id)) }
        : el,
    );
}
