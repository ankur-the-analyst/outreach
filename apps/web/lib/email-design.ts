export type BlockType =
  | "heading"
  | "text"
  | "link"
  | "button"
  | "list"
  | "quote"
  | "markdown"
  | "image"
  | "logo"
  | "video"
  | "social"
  | "divider"
  | "spacer"
  | "section"
  | "row"
  | "footer"
  | "codeBlock"
  | "codeInline"
  | "previewText";

export type EmailElement = {
  id: string;
  type: BlockType;
  content: string;
  props: Record<string, string>;
  children?: EmailElement[][];
};

export type EmailSettings = {
  backgroundColor: string;
  contentBackgroundColor: string;
  fontFamily: string;
  containerWidth: string;
  containerPadding: string;
  layout: string;
};

export type EmailDesign = {
  settings: EmailSettings;
  elements: EmailElement[];
};

export const defaultSettings: EmailSettings = {
  backgroundColor: "#f4f4f6",
  contentBackgroundColor: "#ffffff",
  fontFamily: "Arial, Helvetica, sans-serif",
  containerWidth: "600",
  containerPadding: "20",
  layout: "boxed",
};

const defaultProps: Record<BlockType, Record<string, string>> = {
  heading: { level: "1", align: "left", color: "#1e2024", fontSize: "28" },
  text: { align: "left", color: "#333333", fontSize: "16", lineHeight: "1.6" },
  link: { href: "https://", color: "#d9643a", fontSize: "16" },
  button: {
    href: "https://",
    backgroundColor: "#d9643a",
    textColor: "#ffffff",
    borderRadius: "6",
    padding: "12px 24px",
    align: "center",
    fontSize: "16",
  },
  list: { type: "unordered", color: "#333333", fontSize: "16" },
  quote: {
    borderColor: "#d9643a",
    backgroundColor: "#f8fafc",
    textColor: "#64748b",
    fontSize: "16",
  },
  markdown: { fontSize: "16", textColor: "#333333" },
  image: { src: "", alt: "", width: "100%", align: "center", borderRadius: "0" },
  logo: { src: "", alt: "Logo", width: "150", align: "center" },
  video: { src: "", thumbnail: "", width: "100%" },
  social: {
    align: "center",
    iconSize: "24",
    gap: "12",
    socialLinks: JSON.stringify([
      { platform: "linkedin", url: "" },
      { platform: "website", url: "" },
    ]),
  },
  divider: { color: "#e5e7eb", height: "1", margin: "20" },
  spacer: { height: "20" },
  section: { backgroundColor: "#f9fafb", padding: "20", borderRadius: "8" },
  row: { columns: "2", gap: "20" },
  footer: {
    backgroundColor: "#f9fafb",
    textColor: "#6b7280",
    fontSize: "12",
    padding: "20",
  },
  codeBlock: {
    backgroundColor: "#1f2937",
    textColor: "#e5e7eb",
    fontSize: "14",
  },
  codeInline: {
    backgroundColor: "#f3f4f6",
    textColor: "#e11d48",
    fontSize: "14",
  },
  previewText: {},
};

const defaultContent: Record<BlockType, string> = {
  heading: "Hi {{contact.first_name}}",
  text: "A short note about why I am writing.",
  link: "See more",
  button: "See more",
  list: "- First point\n- Second point",
  quote: "A line worth keeping.",
  markdown: "**Bold** and *italic* text",
  image: "",
  logo: "",
  video: "",
  social: "",
  divider: "",
  spacer: "",
  section: "A short section.",
  row: "",
  footer: "You can reply to this email to opt out.",
  codeBlock: "// note",
  codeInline: "code",
  previewText: "A short preview in the inbox list",
};

export const blockGroups: { id: string; label: string; types: BlockType[] }[] = [
  {
    id: "content",
    label: "Content",
    types: ["heading", "text", "link", "button", "list", "quote", "markdown"],
  },
  { id: "media", label: "Media", types: ["image", "logo", "video", "social"] },
  {
    id: "layout",
    label: "Layout",
    types: ["divider", "spacer", "section", "row", "footer"],
  },
  {
    id: "advanced",
    label: "Advanced",
    types: ["codeBlock", "codeInline", "previewText"],
  },
];

export const blockLabel: Record<BlockType, string> = {
  heading: "Heading",
  text: "Text",
  link: "Link",
  button: "Button",
  list: "List",
  quote: "Quote",
  markdown: "Markdown",
  image: "Image",
  logo: "Logo",
  video: "Video",
  social: "Social",
  divider: "Divider",
  spacer: "Spacer",
  section: "Section",
  row: "Columns",
  footer: "Footer",
  codeBlock: "Code block",
  codeInline: "Inline code",
  previewText: "Preview text",
};

export const mergeTags = [
  "contact.first_name",
  "contact.last_name",
  "contact.email",
  "contact.title",
  "company.name",
  "job.title",
  "profile.name",
];

const types = new Set<string>(Object.keys(defaultProps));

export function newBlock(type: BlockType): EmailElement {
  const el: EmailElement = {
    id: crypto.randomUUID(),
    type,
    content: defaultContent[type],
    props: { ...defaultProps[type] },
  };
  if (type === "row") {
    el.children = [[], []];
  }
  return el;
}

export function parseDesign(raw: unknown, fallbackText = ""): EmailDesign {
  const empty: EmailDesign = { settings: { ...defaultSettings }, elements: [] };
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    if (fallbackText) {
      empty.elements = [{ ...newBlock("text"), content: fallbackText }];
    }
    return empty;
  }
  const o = raw as { settings?: Partial<EmailSettings>; elements?: unknown[] };
  const elements = Array.isArray(o.elements)
    ? o.elements.map(asElement).filter((e): e is EmailElement => e != null)
    : [];
  if (elements.length === 0 && fallbackText) {
    return {
      settings: { ...defaultSettings, ...o.settings },
      elements: [{ ...newBlock("text"), content: fallbackText }],
    };
  }
  return {
    settings: { ...defaultSettings, ...o.settings },
    elements,
  };
}

function asElement(v: unknown): EmailElement | null {
  if (!v || typeof v !== "object") {
    return null;
  }
  const e = v as EmailElement;
  if (typeof e.id !== "string" || typeof e.type !== "string" || !types.has(e.type)) {
    return null;
  }
  const type = e.type as BlockType;
  const children = Array.isArray(e.children)
    ? e.children.map((col) =>
        Array.isArray(col)
          ? col.map(asElement).filter((c): c is EmailElement => c != null)
          : [],
      )
    : undefined;
  return {
    id: e.id,
    type,
    content: typeof e.content === "string" ? e.content : "",
    props: { ...defaultProps[type], ...(e.props ?? {}) },
    children,
  };
}

export function generateHTML(design: EmailDesign): string {
  const s = { ...defaultSettings, ...design.settings };
  const elements = design.elements ?? [];
  if (elements.length === 0) {
    return "";
  }
  const width = Number.parseInt(s.containerWidth, 10) || 600;
  const pad = Number.parseInt(s.containerPadding, 10) || 20;
  const preview = elements
    .filter((el) => el.type === "previewText")
    .map((el) => previewHTML(el.content))
    .join("\n");
  const body = elements
    .filter((el) => el.type !== "previewText")
    .map((el) => wrapElement(el, s.layout === "plain"))
    .join("\n");
  if (s.layout === "plain") {
    return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; font-family: ${esc(s.fontFamily)};">
${preview}
${body}
</body>
</html>`;
  }
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; background-color: ${esc(s.backgroundColor)}; font-family: ${esc(s.fontFamily)};">
${preview}
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color: ${esc(s.backgroundColor)};">
    <tr>
      <td align="center" style="padding: 20px 10px;">
        <table role="presentation" width="${width}" cellpadding="0" cellspacing="0" style="background-color: ${esc(s.contentBackgroundColor)}; max-width: ${width}px; width: 100%;">
          <tr>
            <td style="padding: ${pad}px;">
${body}
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`;
}

function wrapElement(el: EmailElement, plain: boolean): string {
  const inner = elementHTML(el);
  const top = Number.parseInt(el.props.paddingTop || "0", 10) || 0;
  const bottom = Number.parseInt(el.props.paddingBottom || "0", 10) || 0;
  if (plain) {
    const spacing =
      top || bottom ? `margin: ${top}px 0 ${bottom}px 0;` : "margin: 0 0 12px 0;";
    return `<div style="${spacing}">${inner}</div>`;
  }
  const pad =
    top || bottom ? `padding: ${top}px 0 ${bottom}px 0;` : "padding: 0 0 12px 0;";
  return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="${pad}">${inner}</td></tr></table>`;
}

export function generateText(design: EmailDesign): string {
  return flatten(design.elements ?? [])
    .map((el) => {
      if (el.type === "image" || el.type === "logo") {
        return el.props.alt || el.props.src || "";
      }
      if (el.type === "button" || el.type === "link") {
        return `${el.content} ${el.props.href || ""}`.trim();
      }
      if (el.type === "video") {
        return el.props.src || "";
      }
      return el.content || "";
    })
    .map((s) => s.trim())
    .filter(Boolean)
    .join("\n\n");
}

function flatten(elements: EmailElement[]): EmailElement[] {
  const out: EmailElement[] = [];
  for (const el of elements) {
    out.push(el);
    for (const col of el.children ?? []) {
      out.push(...flatten(col));
    }
  }
  return out;
}

function elementHTML(el: EmailElement): string {
  const p = el.props || {};
  switch (el.type) {
    case "heading":
      return headingHTML(el.content, p);
    case "text":
      return textHTML(el.content, p);
    case "link":
      return `<a href="${esc(p.href || "#")}" style="color: ${esc(p.color || "#d9643a")}; font-size: ${num(p.fontSize, 16)}px; text-decoration: underline;">${inline(el.content || "Click here")}</a>`;
    case "button":
      return buttonHTML(el.content, p);
    case "list":
      return listHTML(el.content, p);
    case "quote":
      return quoteHTML(el.content, p);
    case "image":
      return imageHTML(p, false);
    case "logo":
      return imageHTML(p, true);
    case "video":
      return videoHTML(p);
    case "social":
      return socialHTML(p);
    case "divider":
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="padding: ${num(p.margin, 20)}px 0;"><hr style="border: none; border-top: ${num(p.height, 1)}px solid ${esc(p.color || "#e5e7eb")}; margin: 0;" /></td></tr></table>`;
    case "spacer":
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="height: ${num(p.height, 20)}px; line-height: ${num(p.height, 20)}px; font-size: 0;">&nbsp;</td></tr></table>`;
    case "section":
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="background-color: ${esc(p.backgroundColor || "#f9fafb")}; padding: ${num(p.padding, 20)}px; border-radius: ${num(p.borderRadius, 8)}px;"><div style="margin: 0; text-align: center; color: #666666;">${richText(el.content || "")}</div></td></tr></table>`;
    case "row":
      return rowHTML(el);
    case "codeBlock":
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="background-color: ${esc(p.backgroundColor || "#1f2937")}; padding: 16px; border-radius: 8px;"><pre style="margin: 0; color: ${esc(p.textColor || "#e5e7eb")}; font-size: ${num(p.fontSize, 14)}px; font-family: monospace; white-space: pre-wrap;">${esc(el.content || "")}</pre></td></tr></table>`;
    case "codeInline":
      return `<code style="background-color: ${esc(p.backgroundColor || "#f3f4f6")}; color: ${esc(p.textColor || "#e11d48")}; font-size: ${num(p.fontSize, 14)}px; padding: 2px 6px; border-radius: 4px; font-family: monospace;">${esc(el.content || "code")}</code>`;
    case "previewText":
      return previewHTML(el.content);
    case "footer":
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="background-color: ${esc(p.backgroundColor || "#f9fafb")}; padding: ${num(p.padding, 20)}px; text-align: center;"><div style="margin: 0; color: ${esc(p.textColor || "#6b7280")}; font-size: ${num(p.fontSize, 12)}px;">${richText(el.content || "")}</div></td></tr></table>`;
    case "markdown":
      return markdownHTML(el.content, p);
    default:
      return "";
  }
}

function headingHTML(content: string, p: Record<string, string>): string {
  const level = ["1", "2", "3", "4"].includes(p.level) ? p.level : "1";
  const size = { "1": "28px", "2": "24px", "3": "20px", "4": "18px" }[level] ?? "28px";
  return `<h${level} style="margin: 0; font-size: ${size}; color: ${esc(p.color || "#1e2024")}; text-align: ${esc(p.align || "left")}; font-weight: bold;">${inline(content || "Heading")}</h${level}>`;
}

function textHTML(content: string, p: Record<string, string>): string {
  return `<div style="margin: 0; font-size: ${num(p.fontSize, 16)}px; color: ${esc(p.color || "#333333")}; text-align: ${esc(p.align || "left")}; line-height: ${esc(p.lineHeight || "1.6")};">${richText(content)}</div>`;
}

function buttonHTML(content: string, p: Record<string, string>): string {
  const align = p.align || "center";
  return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="${esc(align)}"><table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="background-color: ${esc(p.backgroundColor || "#d9643a")}; border-radius: ${num(p.borderRadius, 6)}px; padding: ${esc(p.padding || "12px 24px")};"><a href="${esc(p.href || "#")}" style="color: ${esc(p.textColor || "#ffffff")}; font-size: ${num(p.fontSize, 16)}px; font-weight: 500; text-decoration: none; display: inline-block;">${inline(content || "Click")}</a></td></tr></table></td></tr></table>`;
}

function listHTML(content: string, p: Record<string, string>): string {
  const items = (content || "- Item")
    .split("\n")
    .map((item) => item.replace(/^[-*]\s*/, "").trim())
    .filter(Boolean);
  const tag = p.type === "ordered" ? "ol" : "ul";
  const lis = items
    .map((item) => `<li style="margin-bottom: 4px;">${inline(item)}</li>`)
    .join("");
  return `<${tag} style="margin: 0; padding-left: 20px; color: ${esc(p.color || "#333333")}; font-size: ${num(p.fontSize, 16)}px;">${lis}</${tag}>`;
}

function quoteHTML(content: string, p: Record<string, string>): string {
  return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="border-left: 4px solid ${esc(p.borderColor || "#d9643a")}; background-color: ${esc(p.backgroundColor || "#f8fafc")}; padding: 16px 20px;"><div style="margin: 0; color: ${esc(p.textColor || "#64748b")}; font-size: ${num(p.fontSize, 16)}px; font-style: italic;">${richText(content)}</div></td></tr></table>`;
}

function imageHTML(p: Record<string, string>, logo: boolean): string {
  if (!p.src) {
    return logo ? "<!-- no logo -->" : "<!-- no image -->";
  }
  const align = p.align || "center";
  const margin =
    align === "left"
      ? "margin-left: 0; margin-right: auto;"
      : align === "right"
        ? "margin-left: auto; margin-right: 0;"
        : "margin-left: auto; margin-right: auto;";
  if (logo) {
    const width = num(p.width, 150);
    return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="${esc(align)}"><img src="${esc(p.src)}" alt="${esc(p.alt || "Logo")}" width="${width}" style="width: ${width}px; max-width: 100%; height: auto; display: block; ${margin}" /></td></tr></table>`;
  }
  return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="${esc(align)}"><img src="${esc(p.src)}" alt="${esc(p.alt || "")}" style="width: ${esc(p.width || "100%")}; max-width: 100%; border-radius: ${num(p.borderRadius, 0)}px; display: block; ${margin}" /></td></tr></table>`;
}

function videoHTML(p: Record<string, string>): string {
  if (!p.thumbnail) {
    return "<!-- no video thumbnail -->";
  }
  return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="center"><a href="${esc(p.src || "#")}"><img src="${esc(p.thumbnail)}" alt="Video" style="width: ${esc(p.width || "100%")}; max-width: 100%; display: block;" /></a></td></tr></table>`;
}

function socialHTML(p: Record<string, string>): string {
  let links: { platform?: string; url?: string }[] = [];
  try {
    const parsed = JSON.parse(p.socialLinks || "[]") as { platform?: string; url?: string }[];
    if (Array.isArray(parsed)) {
      links = parsed.filter((l) => l.platform);
    }
  } catch {
    links = [];
  }
  if (links.length === 0) {
    links = [{ platform: "website", url: "" }];
  }
  const gap = Math.floor(num(p.gap, 12) / 2);
  const html = links
    .map((link) => {
      const name = link.platform || "website";
      return `<a href="${esc(link.url || "#")}" style="display: inline-block; margin: 0 ${gap}px; color: #1e2024; font-size: 14px; text-decoration: underline;">${esc(name)}</a>`;
    })
    .join("");
  return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="${esc(p.align || "center")}" style="padding: 10px 0;">${html}</td></tr></table>`;
}

function rowHTML(el: EmailElement): string {
  const columns = Math.min(4, Math.max(1, num(el.props.columns, 2)));
  const gap = num(el.props.gap, 20);
  const width = Math.floor(100 / columns);
  const cells = Array.from({ length: columns }, (_, i) => {
    const kids = el.children?.[i] ?? [];
    const inner =
      kids.length > 0
        ? kids.map((child) => elementHTML(child)).join("\n")
        : `<p style="margin: 0; color: #9ca3af; text-align: center; font-size: 14px;">Column ${i + 1}</p>`;
    return `<td style="width: ${width}%; padding: 0 ${gap / 2}px; vertical-align: top;">${inner}</td>`;
  }).join("");
  return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr>${cells}</tr></table>`;
}

function markdownHTML(content: string, p: Record<string, string>): string {
  return `<div style="margin: 0; font-size: ${num(p.fontSize, 16)}px; color: ${esc(p.textColor || "#333333")}; line-height: 1.6;">${richText(content, true)}</div>`;
}

function previewHTML(content: string): string {
  return `<div style="display: none; max-height: 0; overflow: hidden; mso-hide: all;">${esc(content || "")}</div>`;
}

function inline(text: string): string {
  let out = esc(text);
  out = out.replace(/\*\*([^*\n]+?)\*\*/g, "<strong>$1</strong>");
  out = out.replace(/(^|[\s(])\*([^*\n]+?)\*(?=[\s.,!?):]|$)/g, "$1<em>$2</em>");
  out = out.replace(/__([^_\n]+?)__/g, "<u>$1</u>");
  out = out.replace(/==([^=\n]+?)==/g, '<mark style="background-color: #fdeee9; color: inherit;">$1</mark>');
  out = out.replace(/~~([^~\n]+?)~~/g, "<s>$1</s>");
  return out;
}

export function richText(content: string, links = false): string {
  const lines = (content || "").split("\n");
  const parts: string[] = [];
  let i = 0;
  while (i < lines.length) {
    const kind = lineKind(lines[i]);
    if (kind === "ul" || kind === "ol") {
      const items: string[] = [];
      while (i < lines.length && lineKind(lines[i]) === kind) {
        items.push(stripMarker(lines[i]));
        i++;
      }
      const lis = items
        .map((item) => `<li style="margin: 0 0 4px;">${formatInline(item, links)}</li>`)
        .join("");
      parts.push(`<${kind} style="margin: 8px 0; padding-left: 20px;">${lis}</${kind}>`);
      continue;
    }
    const para: string[] = [];
    while (i < lines.length && lineKind(lines[i]) === "text") {
      para.push(lines[i]);
      i++;
    }
    parts.push(para.map((line) => (line === "" ? "" : formatInline(line, links))).join("<br>"));
  }
  return parts.filter((part) => part !== "").join("<br>");
}

function formatInline(text: string, links: boolean): string {
  let out = inline(text);
  if (links) {
    out = out.replace(
      /\[(.+?)\]\((.+?)\)/g,
      `<a href="$2" style="color: #d9643a;">$1</a>`,
    );
  }
  return out;
}

function lineKind(line: string): "ul" | "ol" | "text" {
  if (/^\s*(?:[-*]|•)\s+\S/.test(line) || /^\s*(?:[-*]|•)\s*$/.test(line)) {
    return "ul";
  }
  if (/^\s*\d+[.)]\s+\S/.test(line)) {
    return "ol";
  }
  return "text";
}

function stripMarker(line: string): string {
  return line
    .replace(/^\s*(?:(?:[-*]|•)|\d+[.)])\s+/, "")
    .replace(/^\s*(?:[-*]|•)\s*$/, "");
}

export function applyFormat(
  content: string,
  start: number,
  end: number,
  left: string,
  right = left,
  placeholder = "text",
): { value: string; start: number; end: number } {
  const current = content || "";
  const from = clamp(Math.min(start, end), 0, current.length);
  const to = clamp(Math.max(start, end), 0, current.length);
  const selected = current.slice(from, to);
  if (selected.startsWith(left) && selected.endsWith(right) && selected.length >= left.length + right.length) {
    const inner = selected.slice(left.length, selected.length - right.length);
    return { value: current.slice(0, from) + inner + current.slice(to), start: from, end: from + inner.length };
  }
  const before = current.slice(Math.max(0, from - left.length), from);
  const after = current.slice(to, to + right.length);
  if (before === left && after === right) {
    const nextFrom = from - left.length;
    return {
      value: current.slice(0, nextFrom) + selected + current.slice(to + right.length),
      start: nextFrom,
      end: nextFrom + selected.length,
    };
  }
  const inner = selected || placeholder;
  const value = current.slice(0, from) + left + inner + right + current.slice(to);
  return { value, start: from + left.length, end: from + left.length + inner.length };
}

export function toggleBullets(
  content: string,
  start: number,
  end: number,
): { value: string; start: number; end: number } {
  const current = content || "";
  if (current === "") {
    return { value: "- ", start: 2, end: 2 };
  }
  const from = clamp(Math.min(start, end), 0, current.length);
  const to = clamp(Math.max(start, end), 0, current.length);
  const lineStart = current.lastIndexOf("\n", from - 1) + 1;
  let lineEnd = current.indexOf("\n", to);
  if (lineEnd < 0) {
    lineEnd = current.length;
  }
  if (to > from && current[to - 1] === "\n") {
    lineEnd = to - 1;
  }
  const lines = current.slice(lineStart, lineEnd).split("\n");
  const filled = lines.filter((line) => line.trim() !== "");
  const allMarked = filled.length > 0 && filled.every((line) => lineKind(line) !== "text");
  const nextLines = lines.map((line) => {
    if (line.trim() === "") {
      return line;
    }
    if (allMarked) {
      return stripMarker(line);
    }
    if (lineKind(line) !== "text") {
      return line;
    }
    const indent = line.match(/^\s*/)?.[0] ?? "";
    return `${indent}- ${line.slice(indent.length)}`;
  });
  const block = nextLines.join("\n");
  return {
    value: current.slice(0, lineStart) + block + current.slice(lineEnd),
    start: lineStart,
    end: lineStart + block.length,
  };
}

function clamp(n: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, n));
}

export function renderBlock(el: EmailElement): string {
  return elementHTML(el);
}

function num(value: string | undefined, fallback: number): number {
  const n = Number.parseInt(value || "", 10);
  return Number.isFinite(n) ? n : fallback;
}

function esc(text: string): string {
  const variables: string[] = [];
  const held = (text || "").replace(/\{\{[^}]+\}\}/g, (match) => {
    variables.push(match);
    return `__VAR_${variables.length - 1}__`;
  });
  let out = held
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
  variables.forEach((variable, i) => {
    out = out.replace(`__VAR_${i}__`, variable);
  });
  return out;
}
