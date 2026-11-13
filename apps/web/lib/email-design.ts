export type BlockType = "heading" | "text" | "button" | "image" | "footer";

export type EmailElement = {
  id: string;
  type: BlockType;
  content: string;
  props: Record<string, string>;
};

export type EmailSettings = {
  backgroundColor: string;
  contentBackgroundColor: string;
  fontFamily: string;
  containerWidth: string;
  containerPadding: string;
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
};

const defaultProps: Record<BlockType, Record<string, string>> = {
  heading: { level: "1", align: "left", color: "#1e2024", fontSize: "28" },
  text: { align: "left", color: "#333333", fontSize: "16", lineHeight: "1.6" },
  button: {
    href: "https://",
    backgroundColor: "#d9643a",
    textColor: "#ffffff",
    borderRadius: "6",
    padding: "12px 24px",
    align: "center",
    fontSize: "16",
  },
  image: { src: "", alt: "", width: "100%", align: "center", borderRadius: "0" },
  footer: {
    backgroundColor: "#f9fafb",
    textColor: "#6b7280",
    fontSize: "12",
    padding: "20",
  },
};

const defaultContent: Record<BlockType, string> = {
  heading: "Hi {{contact.first_name}}",
  text: "A short note about why I am writing.",
  button: "See more",
  image: "",
  footer: "You can reply to this email to opt out.",
};

export const blockLabels: { type: BlockType; label: string }[] = [
  { type: "heading", label: "Heading" },
  { type: "text", label: "Text" },
  { type: "button", label: "Button" },
  { type: "image", label: "Image" },
  { type: "footer", label: "Footer" },
];

export const mergeTags = [
  "contact.first_name",
  "contact.last_name",
  "contact.email",
  "contact.title",
  "company.name",
  "job.title",
  "profile.name",
];

export function newBlock(type: BlockType): EmailElement {
  return {
    id: crypto.randomUUID(),
    type,
    content: defaultContent[type],
    props: { ...defaultProps[type] },
  };
}

export function parseDesign(raw: unknown, fallbackText = ""): EmailDesign {
  const empty: EmailDesign = { settings: { ...defaultSettings }, elements: [] };
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    if (fallbackText) {
      empty.elements = [{ ...newBlock("text"), content: fallbackText }];
    }
    return empty;
  }
  const o = raw as { settings?: Partial<EmailSettings>; elements?: EmailElement[] };
  const elements = Array.isArray(o.elements) ? o.elements.filter(isElement) : [];
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

function isElement(v: unknown): v is EmailElement {
  if (!v || typeof v !== "object") {
    return false;
  }
  const e = v as EmailElement;
  return typeof e.id === "string" && typeof e.type === "string";
}

export function generateHTML(design: EmailDesign): string {
  const s = { ...defaultSettings, ...design.settings };
  const width = s.containerWidth || "600";
  const pad = s.containerPadding || "20";
  const body = (design.elements ?? [])
    .map((el) => `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td>${elementHTML(el)}</td></tr></table>`)
    .join("\n");
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; background-color: ${esc(s.backgroundColor)}; font-family: ${esc(s.fontFamily)};">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color: ${esc(s.backgroundColor)};">
    <tr>
      <td align="center" style="padding: 20px 10px;">
        <table role="presentation" width="${esc(width)}" cellpadding="0" cellspacing="0" style="background-color: ${esc(s.contentBackgroundColor)}; max-width: ${esc(width)}px; width: 100%;">
          <tr>
            <td style="padding: ${esc(pad)}px;">
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

export function generateText(design: EmailDesign): string {
  return (design.elements ?? [])
    .map((el) => {
      if (el.type === "image") {
        return el.props.alt || el.props.src || "";
      }
      if (el.type === "button") {
        return `${el.content} ${el.props.href || ""}`.trim();
      }
      return el.content || "";
    })
    .map((s) => s.trim())
    .filter(Boolean)
    .join("\n\n");
}

function elementHTML(el: EmailElement): string {
  const p = el.props || {};
  switch (el.type) {
    case "heading": {
      const level = ["1", "2", "3"].includes(p.level) ? p.level : "1";
      const size = { "1": "28px", "2": "24px", "3": "20px" }[level] ?? "28px";
      return `<h${level} style="margin: 0; font-size: ${size}; color: ${esc(p.color || "#1e2024")}; text-align: ${esc(p.align || "left")}; font-weight: bold;">${esc(el.content || "Heading")}</h${level}>`;
    }
    case "text":
      return `<p style="margin: 0; font-size: ${esc(p.fontSize || "16")}px; color: ${esc(p.color || "#333333")}; text-align: ${esc(p.align || "left")}; line-height: ${esc(p.lineHeight || "1.6")};">${esc(el.content || "").replace(/\n/g, "<br>")}</p>`;
    case "button":
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="${esc(p.align || "center")}"><table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="background-color: ${esc(p.backgroundColor || "#d9643a")}; border-radius: ${esc(p.borderRadius || "6")}px; padding: ${esc(p.padding || "12px 24px")};"><a href="${esc(p.href || "#")}" style="color: ${esc(p.textColor || "#ffffff")}; font-size: ${esc(p.fontSize || "16")}px; font-weight: 500; text-decoration: none; display: inline-block;">${esc(el.content || "Click")}</a></td></tr></table></td></tr></table>`;
    case "image":
      if (!p.src) {
        return "<!-- no image -->";
      }
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="${esc(p.align || "center")}"><img src="${esc(p.src)}" alt="${esc(p.alt || "")}" style="width: ${esc(p.width || "100%")}; max-width: 100%; border-radius: ${esc(p.borderRadius || "0")}px; display: block;" /></td></tr></table>`;
    case "footer":
      return `<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td style="background-color: ${esc(p.backgroundColor || "#f9fafb")}; padding: ${esc(p.padding || "20")}px; text-align: center;"><p style="margin: 0; color: ${esc(p.textColor || "#6b7280")}; font-size: ${esc(p.fontSize || "12")}px;">${esc(el.content || "").replace(/\n/g, "<br>")}</p></td></tr></table>`;
    default:
      return "";
  }
}

function esc(text: string): string {
  return (text || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
