"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState, PageBody, PageHeader } from "@/components/page-header";
import type { SavedEmail } from "@/lib/api";
import {
  type BlockType,
  type EmailDesign,
  type EmailElement,
  blockLabels,
  generateHTML,
  generateText,
  mergeTags,
  newBlock,
  parseDesign,
} from "@/lib/email-design";
import { fieldTextarea } from "@/lib/labels";

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
        title="Saved emails"
        actions={
          <Button type="button" onClick={() => setEditing("new")}>
            New email
          </Button>
        }
      />
      <PageBody>
        {editing ? (
          <CanvasEditor
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
            title="No saved emails yet"
            description="Build a heading and a note, then reuse it in a campaign."
            action={
              <Button type="button" onClick={() => setEditing("new")}>
                New email
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

function CanvasEditor({
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
  const [dragType, setDragType] = useState<BlockType | null>(null);
  const [dragId, setDragId] = useState<string | null>(null);

  const selectedEl = design.elements.find((e) => e.id === selected);

  function add(type: BlockType, at?: number) {
    const el = newBlock(type);
    setDesign((d) => {
      const elements = [...d.elements];
      const i = at == null ? elements.length : at;
      elements.splice(i, 0, el);
      return { ...d, elements };
    });
    setSelected(el.id);
  }

  function patch(id: string, next: Partial<EmailElement>) {
    setDesign((d) => ({
      ...d,
      elements: d.elements.map((e) => (e.id === id ? { ...e, ...next, props: { ...e.props, ...next.props } } : e)),
    }));
  }

  function remove(id: string) {
    setDesign((d) => ({ ...d, elements: d.elements.filter((e) => e.id !== id) }));
    setSelected((cur) => (cur === id ? null : cur));
  }

  function move(from: string, before: string | null) {
    setDesign((d) => {
      const elements = [...d.elements];
      const i = elements.findIndex((e) => e.id === from);
      if (i < 0) {
        return d;
      }
      const [item] = elements.splice(i, 1);
      const j = before ? elements.findIndex((e) => e.id === before) : elements.length;
      elements.splice(j < 0 ? elements.length : j, 0, item);
      return { ...d, elements };
    });
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
      setError(data?.error?.message ?? "Could not save this email");
      return;
    }
    onSaved((await res.json()) as SavedEmail);
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 md:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="email-name">Name</Label>
          <Input id="email-name" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="email-subject">Subject</Label>
          <Input id="email-subject" value={subject} onChange={(e) => setSubject(e.target.value)} />
        </div>
      </div>
      <p className="font-body-sm text-muted-foreground">
        Drag a block onto the canvas. Merge tags like {"{{contact.first_name}}"} are filled when it sends.
      </p>
      <div className="flex flex-wrap gap-1.5">
        {mergeTags.map((tag) => (
          <button
            key={tag}
            type="button"
            className="rounded-full border border-border bg-card px-2 py-0.5 font-label-sm text-muted-foreground hover:bg-muted"
            onClick={() => {
              const token = `{{${tag}}}`;
              if (selectedEl) {
                patch(selectedEl.id, { content: `${selectedEl.content} ${token}`.trim() });
              } else {
                setSubject((s) => `${s} ${token}`.trim());
              }
            }}
          >
            {`{{${tag}}}`}
          </button>
        ))}
      </div>
      <div className="grid min-h-[28rem] gap-3 lg:grid-cols-[11rem_minmax(0,1fr)_16rem]">
        <div className="flex flex-col gap-1.5 rounded-md border border-border bg-card p-2">
          <p className="px-1 font-label-sm text-muted-foreground uppercase tracking-[0.08em]">Blocks</p>
          {blockLabels.map((b) => (
            <button
              key={b.type}
              type="button"
              draggable
              onDragStart={() => setDragType(b.type)}
              onDragEnd={() => setDragType(null)}
              onClick={() => add(b.type)}
              className="rounded-md border border-border px-2 py-2 text-left font-label-md text-foreground hover:bg-muted"
            >
              {b.label}
            </button>
          ))}
        </div>
        <div
          className="flex flex-col gap-2 rounded-md border border-dashed border-border bg-muted/40 p-3"
          onDragOver={(e) => e.preventDefault()}
          onDrop={() => {
            if (dragType) {
              add(dragType);
              setDragType(null);
            }
          }}
        >
          {design.elements.length === 0 ? (
            <p className="m-auto font-body-md text-muted-foreground">Drop a heading here</p>
          ) : (
            design.elements.map((el) => (
              <div
                key={el.id}
                draggable
                onDragStart={() => setDragId(el.id)}
                onDragEnd={() => setDragId(null)}
                onDragOver={(e) => e.preventDefault()}
                onDrop={(e) => {
                  e.stopPropagation();
                  if (dragType) {
                    const at = design.elements.findIndex((x) => x.id === el.id);
                    add(dragType, at);
                    setDragType(null);
                    return;
                  }
                  if (dragId && dragId !== el.id) {
                    move(dragId, el.id);
                    setDragId(null);
                  }
                }}
                onClick={() => setSelected(el.id)}
                className={
                  selected === el.id
                    ? "cursor-grab rounded-md border border-primary bg-card p-3"
                    : "cursor-grab rounded-md border border-border bg-card p-3"
                }
              >
                <CanvasPreview el={el} />
              </div>
            ))
          )}
        </div>
        <div className="flex flex-col gap-3 rounded-md border border-border bg-card p-3">
          <p className="font-label-sm text-muted-foreground uppercase tracking-[0.08em]">Block</p>
          {selectedEl ? (
            <BlockFields
              el={selectedEl}
              onChange={(next) => patch(selectedEl.id, next)}
              onRemove={() => remove(selectedEl.id)}
            />
          ) : (
            <p className="font-body-sm text-muted-foreground">Select a block to edit it.</p>
          )}
        </div>
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Back
        </Button>
        <Button type="button" pending={pending} onClick={() => void save()}>
          Save
        </Button>
      </div>
    </div>
  );
}

function CanvasPreview({ el }: { el: EmailElement }) {
  if (el.type === "heading") {
    return <p className="font-headline-sm text-foreground">{el.content || "Heading"}</p>;
  }
  if (el.type === "button") {
    return (
      <span className="inline-flex rounded-md bg-primary px-3 py-1.5 font-label-md text-primary-foreground">
        {el.content || "Button"}
      </span>
    );
  }
  if (el.type === "image") {
    return (
      <p className="font-body-sm text-muted-foreground">
        {el.props.src ? el.props.alt || el.props.src : "Image URL"}
      </p>
    );
  }
  return <p className="whitespace-pre-wrap font-body-md text-foreground">{el.content || el.type}</p>;
}

function BlockFields({
  el,
  onChange,
  onRemove,
}: {
  el: EmailElement;
  onChange: (next: Partial<EmailElement>) => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      {el.type !== "image" ? (
        <div className="flex flex-col gap-1.5">
          <Label>Text</Label>
          <textarea
            className={fieldTextarea}
            rows={4}
            value={el.content}
            onChange={(e) => onChange({ content: e.target.value })}
          />
        </div>
      ) : null}
      {el.type === "button" || el.type === "image" ? (
        <div className="flex flex-col gap-1.5">
          <Label>{el.type === "image" ? "Image URL" : "Link"}</Label>
          <Input
            value={el.type === "image" ? el.props.src : el.props.href}
            onChange={(e) =>
              onChange({
                props: el.type === "image" ? { src: e.target.value } : { href: e.target.value },
              })
            }
          />
        </div>
      ) : null}
      <Button type="button" variant="ghost" className="w-fit" onClick={onRemove}>
        Remove block
      </Button>
    </div>
  );
}
