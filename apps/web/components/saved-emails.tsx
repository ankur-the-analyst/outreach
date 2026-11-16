"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { Bold, Highlighter, Italic, Link2, List, Strikethrough, Underline } from "lucide-react";
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
import {
  type TemplateVar,
  useTemplateVars,
  VariablePicker,
} from "@/components/variable-picker";
import type { SavedEmail } from "@/lib/api";
import {
  type BlockType,
  type EmailDesign,
  type EmailElement,
  applyKind,
  blockGroups,
  blockLabel,
  emailFonts,
  type EditHistory,
  type FormatKind,
  formatKindFromKey,
  generateHTML,
  generateText,
  historyPush,
  historyRedo,
  historyStart,
  historyUndo,
  newBlock,
  parseDesign,
  renderBlock,
  wrapLink,
} from "@/lib/email-design";
import { fieldSelect, fieldTextarea } from "@/lib/labels";

export function SavedEmails({
  emails,
  template,
}: {
  emails: SavedEmail[];
  template: string | null;
}) {
  const router = useRouter();
  const editing = template;
  const [rows, setRows] = useState(emails);
  const [toast, setToast] = useState<{ tone: "ok" | "bad"; text: string } | null>(null);

  useEffect(() => {
    setRows(emails);
  }, [emails]);

  useEffect(() => {
    if (!toast) {
      return;
    }
    const timer = window.setTimeout(() => setToast(null), 3200);
    return () => window.clearTimeout(timer);
  }, [toast]);

  useEffect(() => {
    if (editing && editing !== "new" && !rows.some((row) => row.id === editing)) {
      router.replace("/emails");
    }
  }, [editing, rows, router]);

  const current = editing && editing !== "new" ? rows.find((e) => e.id === editing) : null;

  function openTemplate(id: "new" | string) {
    router.push(`/emails?template=${encodeURIComponent(id)}`);
  }

  return (
    <>
      <PageHeader
        title="Email templates"
        actions={
          <Button type="button" onClick={() => openTemplate("new")}>
            New template
          </Button>
        }
      />
      <PageBody>
        {editing === "new" || current ? (
          <TemplateBuilder
            key={editing}
            initial={current ?? undefined}
            onCancel={() => router.push("/emails")}
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
              setToast({ tone: "ok", text: "Template saved" });
              router.replace(`/emails?template=${encodeURIComponent(saved.id)}`);
            }}
            onFailed={(text) => setToast({ tone: "bad", text })}
          />
        ) : rows.length === 0 ? (
          <EmptyState
            title="No email templates yet"
            description="Build the email here, then pick it when you start a campaign."
            action={
              <Button type="button" onClick={() => openTemplate("new")}>
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
                  onClick={() => openTemplate(e.id)}
                >
                  <span className="font-headline-sm text-foreground">{e.name}</span>
                  <span className="font-body-sm text-muted-foreground">{e.subject}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </PageBody>
      {toast ? (
        <p
          role={toast.tone === "bad" ? "alert" : "status"}
          className={`fixed bottom-4 left-1/2 z-50 max-w-sm -translate-x-1/2 rounded-md border bg-card px-3 py-2 font-body-sm shadow-sm ${
            toast.tone === "bad" ? "border-destructive text-destructive" : "border-border text-foreground"
          }`}
        >
          {toast.text}
        </p>
      ) : null}
    </>
  );
}

function TemplateBuilder({
  initial,
  onCancel,
  onSaved,
  onFailed,
}: {
  initial?: SavedEmail;
  onCancel: () => void;
  onSaved: (e: SavedEmail) => void;
  onFailed: (text: string) => void;
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
  const subjectRef = useRef<HTMLInputElement>(null);
  const vars = useTemplateVars();

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
      const text = data?.error?.message ?? "Could not save this template";
      setError(text);
      onFailed(text);
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
          <div className="flex items-center justify-between gap-2">
            <Label htmlFor="email-subject">Subject</Label>
            <VariablePicker
              vars={vars}
              onInsert={(token) =>
                setSubject((cur) => insertToken(cur, subjectRef.current, token))
              }
            />
          </div>
          <Input
            id="email-subject"
            ref={subjectRef}
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
          />
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
      <div className="flex flex-wrap gap-3">
        <div className="flex min-w-40 flex-col gap-1.5">
          <Label htmlFor="email-layout">Layout</Label>
          <select
            id="email-layout"
            className={fieldSelect}
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
        <div className="flex min-w-40 flex-col gap-1.5">
          <Label htmlFor="email-font">Font</Label>
          <select
            id="email-font"
            className={fieldSelect}
            value={design.settings.fontFamily}
            onChange={(e) =>
              setDesign((d) => ({
                ...d,
                settings: { ...d.settings, fontFamily: e.target.value },
              }))
            }
          >
            {emailFonts.map((font) => (
              <option key={font.value} value={font.value}>
                {font.label}
              </option>
            ))}
          </select>
        </div>
      </div>
      <p className="font-body-sm text-muted-foreground">
        Add a block, then edit it on the right. Insert a field where the cursor is. It is filled when the email sends.
      </p>
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
              style={{
                backgroundColor: design.settings.contentBackgroundColor,
                fontFamily: design.settings.fontFamily,
              }}
            >
              {design.elements.map((el) => (
                <CanvasBlock
                  key={el.id}
                  element={el}
                  fontFamily={design.settings.fontFamily}
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
              vars={vars}
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
  fontFamily,
  selected,
  onSelect,
  onAddToColumn,
}: {
  element: EmailElement;
  fontFamily: string;
  selected: string | null;
  onSelect: (id: string) => void;
  onAddToColumn: (rowID: string, column: number, type: BlockType) => void;
}) {
  const active = element.id === selected;
  return (
    <div className="flex flex-col gap-2">
      <div
        role="button"
        tabIndex={0}
        onClick={() => onSelect(element.id)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onSelect(element.id);
          }
        }}
        className={`rounded-md border px-3 py-2 text-left ${
          active ? "border-primary bg-accent" : "border-border bg-card hover:bg-muted"
        }`}
      >
        <span className="font-label-sm text-muted-foreground">{blockLabel[element.type]}</span>
        <div
          className="pointer-events-none mt-1 break-words text-sm text-foreground [&_ul]:list-disc [&_ol]:list-decimal"
          style={{ fontFamily }}
          dangerouslySetInnerHTML={{ __html: canvasPreview(element) }}
        />
      </div>
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
                  fontFamily={fontFamily}
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
  vars,
  onChange,
  onColumns,
  onRemove,
}: {
  element: EmailElement;
  vars: TemplateVar[];
  onChange: (next: Partial<EmailElement>) => void;
  onColumns: (n: number) => void;
  onRemove: () => void;
}) {
  const p = element.props;
  const contentRef = useRef<HTMLTextAreaElement>(null);
  const history = useRef<EditHistory & { id: string }>({
    ...historyStart(element.content),
    id: element.id,
  });
  const [linkOpen, setLinkOpen] = useState(false);
  const [linkUrl, setLinkUrl] = useState("https://");
  const prop = (key: string, value: string) => onChange({ props: { [key]: value } });
  const hasContent = contentBlocks.has(element.type);
  const formatted = formattedBlocks.has(element.type);

  if (history.current.id !== element.id) {
    history.current = { ...historyStart(element.content), id: element.id };
  }

  function setContent(content: string) {
    history.current = { ...historyPush(history.current, content), id: element.id };
    onChange({ content });
    return content;
  }

  function stepHistory(direction: "undo" | "redo") {
    const next = direction === "undo" ? historyUndo(history.current) : historyRedo(history.current);
    history.current = { ...next, id: element.id };
    onChange({ content: next.stack[next.index] });
    const pos = next.stack[next.index].length;
    setTimeout(() => {
      contentRef.current?.focus();
      contentRef.current?.setSelectionRange(pos, pos);
    }, 0);
  }

  function applyLink() {
    const el = contentRef.current;
    const start = el?.selectionStart ?? element.content.length;
    const end = el?.selectionEnd ?? element.content.length;
    const next = wrapLink(element.content, start, end, linkUrl);
    setContent(next.value);
    setLinkOpen(false);
    setTimeout(() => {
      el?.focus();
      el?.setSelectionRange(next.end, next.end);
    }, 0);
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="font-label-md text-foreground">{blockLabel[element.type]}</p>
      {hasContent ? (
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center justify-between gap-2">
            <Label htmlFor="block-content">Content</Label>
            <VariablePicker
              vars={vars}
              onInsert={(token) =>
                setContent(insertToken(element.content, contentRef.current, token))
              }
            />
          </div>
          {formatted ? (
            <FormatBar
              bullets={bulletBlocks.has(element.type)}
              onFormat={(kind) =>
                runFormat(kind, element.content, contentRef.current, setContent)
              }
              onLink={() => setLinkOpen((open) => !open)}
            />
          ) : null}
          {formatted && linkOpen ? (
            <form
              className="flex flex-wrap items-center gap-1.5"
              onSubmit={(e) => {
                e.preventDefault();
                applyLink();
              }}
            >
              <Input
                value={linkUrl}
                onChange={(e) => setLinkUrl(e.target.value)}
                placeholder="https://"
                aria-label="Link address"
              />
              <Button type="submit" size="sm">
                Add link
              </Button>
            </form>
          ) : null}
          <textarea
            id="block-content"
            ref={contentRef}
            rows={element.type === "text" ? 8 : 4}
            className={fieldTextarea}
            value={element.content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={
              formatted
                ? (e) => {
                    if (!(e.metaKey || e.ctrlKey) || e.altKey) {
                      return;
                    }
                    const key = e.key.toLowerCase();
                    if (key === "z") {
                      e.preventDefault();
                      stepHistory(e.shiftKey ? "redo" : "undo");
                      return;
                    }
                    if (key === "y") {
                      e.preventDefault();
                      stepHistory("redo");
                      return;
                    }
                    if (key === "k" && !e.shiftKey) {
                      e.preventDefault();
                      setLinkOpen(true);
                      return;
                    }
                    const kind = formatKindFromKey(
                      e.key,
                      e.shiftKey,
                      bulletBlocks.has(element.type),
                    );
                    if (!kind) {
                      return;
                    }
                    e.preventDefault();
                    runFormat(kind, element.content, contentRef.current, setContent);
                  }
                : undefined
            }
          />
          {bulletBlocks.has(element.type) ? (
            <p className="font-body-sm text-muted-foreground">
              Start a line with - for a bullet, or 1. for a number. That stays in this block.
            </p>
          ) : null}
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

const contentBlocks = new Set<BlockType>([
  "heading",
  "text",
  "link",
  "button",
  "list",
  "quote",
  "markdown",
  "section",
  "row",
  "footer",
  "codeBlock",
  "codeInline",
  "previewText",
]);

const formattedBlocks = new Set<BlockType>([
  "heading",
  "text",
  "link",
  "button",
  "list",
  "quote",
  "markdown",
  "section",
  "footer",
]);

const bulletBlocks = new Set<BlockType>([
  "text",
  "quote",
  "markdown",
  "section",
  "footer",
  "list",
]);

function runFormat(
  kind: FormatKind,
  value: string,
  el: HTMLTextAreaElement | null,
  onChange: (value: string) => void,
) {
  const start = el?.selectionStart ?? value.length;
  const end = el?.selectionEnd ?? value.length;
  const next = applyKind(kind, value, start, end);
  onChange(next.value);
  const from = kind === "bullets" ? next.end : next.start;
  const to = next.end;
  setTimeout(() => {
    el?.focus();
    el?.setSelectionRange(from, to);
  }, 0);
}

function FormatBar({
  bullets,
  onFormat,
  onLink,
}: {
  bullets: boolean;
  onFormat: (kind: FormatKind) => void;
  onLink: () => void;
}) {
  return (
    <div className="flex flex-wrap gap-0.5 rounded-md border border-border bg-muted p-0.5">
      <Button type="button" variant="ghost" size="icon-xs" title="Bold (⌘B)" onMouseDown={(e) => e.preventDefault()} onClick={() => onFormat("bold")}>
        <Bold />
      </Button>
      <Button type="button" variant="ghost" size="icon-xs" title="Italic (⌘I)" onMouseDown={(e) => e.preventDefault()} onClick={() => onFormat("italic")}>
        <Italic />
      </Button>
      <Button type="button" variant="ghost" size="icon-xs" title="Underline (⌘U)" onMouseDown={(e) => e.preventDefault()} onClick={() => onFormat("underline")}>
        <Underline />
      </Button>
      <Button type="button" variant="ghost" size="icon-xs" title="Highlight (⌘⇧H)" onMouseDown={(e) => e.preventDefault()} onClick={() => onFormat("highlight")}>
        <Highlighter />
      </Button>
      <Button type="button" variant="ghost" size="icon-xs" title="Strikethrough (⌘⇧X)" onMouseDown={(e) => e.preventDefault()} onClick={() => onFormat("strike")}>
        <Strikethrough />
      </Button>
      <Button type="button" variant="ghost" size="icon-xs" title="Link (⌘K)" onMouseDown={(e) => e.preventDefault()} onClick={onLink}>
        <Link2 />
      </Button>
      {bullets ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          title="Bullets (⌘⇧8)"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => onFormat("bullets")}
        >
          <List />
        </Button>
      ) : null}
    </div>
  );
}

function insertToken(
  value: string,
  field: {
    selectionStart: number | null;
    selectionEnd: number | null;
    focus: () => void;
    setSelectionRange: (start: number, end: number) => void;
  } | null,
  token: string,
): string {
  const start = field?.selectionStart ?? value.length;
  const end = field?.selectionEnd ?? value.length;
  const next = value.slice(0, start) + token + value.slice(end);
  const pos = start + token.length;
  setTimeout(() => {
    field?.focus();
    field?.setSelectionRange(pos, pos);
  }, 0);
  return next;
}

function canvasPreview(element: EmailElement): string {
  if (
    !element.content &&
    !element.props.src &&
    !element.props.alt &&
    element.type !== "divider" &&
    element.type !== "spacer"
  ) {
    return "Empty";
  }
  const html = renderBlock(element);
  if (!html || html.startsWith("<!--")) {
    return "Empty";
  }
  return html;
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
