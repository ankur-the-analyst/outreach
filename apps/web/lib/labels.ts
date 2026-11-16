import type { Draft, Mailbox } from "@/lib/api";

export const fieldSelect =
  "h-8 w-full rounded-md border border-input bg-card px-2.5 font-[family-name:var(--font-body)] text-sm text-foreground outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50";

export const fieldTextarea =
  "w-full rounded-md border border-input bg-card px-2.5 py-2 font-[family-name:var(--font-body)] text-sm text-foreground outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50";

export const intentLabel: Record<string, string> = {
  job: "Job search",
  referral: "Referral",
  informational: "Informational",
  cold: "Cold outreach",
  freelance: "Freelance",
};

export const campaignStatusLabel: Record<string, string> = {
  draft: "Draft",
  running: "Running",
  paused: "Paused",
  stopped: "Stopped",
  completed: "Done",
};

export const campaignPersonStatus: Record<string, string> = {
  pending: "Queued",
  waiting_review: "Drafted",
  active: "In sequence",
  replied: "Replied",
  stopped: "Stopped",
};

export const trackerStageLabel: Record<string, string> = {
  queued: "Queued",
  drafted: "Drafted",
  sent: "Sent",
  followed_up: "Followed up",
  replied: "Replied",
  interview: "Interview",
  offer: "Offer",
  rejected: "Rejected",
  closed: "Closed",
  stopped: "Stopped",
  bounced: "Bounced",
  unsubscribed: "Unsubscribed",
};

export function healthMetricLabel(metric: string): string {
  switch (metric) {
    case "unknown":
      return "Sends we're still checking";
    case "paused":
      return "Paused mailboxes";
    case "sync_lag_s":
      return "Oldest inbox sync (seconds)";
    default:
      return metric;
  }
}

export function replyModeLabel(mode: string): string {
  if (mode === "off") {
    return "Don't draft replies";
  }
  return "Draft replies for review";
}

export function mailboxUsageLabel(used: number, cap: number): string {
  return `${used} of ${cap} · last 24h`;
}

export function formatVisitTime(iso?: string): string {
  if (!iso) {
    return "";
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(d);
}

export function formatNextSend(iso?: string): string {
  if (!iso) {
    return "later";
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "later";
  }
  return new Intl.DateTimeFormat(undefined, {
    weekday: "short",
    hour: "numeric",
    minute: "2-digit",
  }).format(d);
}

export function mailboxIdle(mailbox?: Pick<
  Mailbox,
  "sending_now" | "open_today"
>): boolean {
  if (!mailbox) {
    return false;
  }
  if (mailbox.sending_now === false) {
    return true;
  }
  if (mailbox.sending_now === undefined && mailbox.open_today === false) {
    return true;
  }
  return false;
}

export function mailboxSendLabel(
  mailbox?: Pick<Mailbox, "sending_now" | "open_today" | "next_open">,
): string {
  if (!mailboxIdle(mailbox)) {
    return "";
  }
  if (mailbox?.next_open) {
    return `Not sending until ${formatNextSend(mailbox.next_open)}`;
  }
  return "Not sending today";
}

export function mailboxSendsAtLabel(
  mailbox?: Pick<Mailbox, "sending_now" | "open_today" | "next_open">,
): string {
  if (!mailboxIdle(mailbox)) {
    return "";
  }
  if (mailbox?.next_open) {
    return `Sends ${formatNextSend(mailbox.next_open)}`;
  }
  return "Sends later";
}

export function mailboxStatus(status: string): string {
  switch (status) {
    case "warming":
      return "Warming up";
    case "healthy":
      return "Ready";
    case "paused":
      return "Paused";
    case "disconnected":
      return "Disconnected";
    default:
      return status;
  }
}

export function jobAnalysisLabel(status: string): string {
  switch (status) {
    case "pending":
      return "Not summarized yet";
    case "ready":
      return "Summarized";
    case "failed":
      return "Couldn't summarize";
    default:
      return status;
  }
}

export function warmupKindHint(kind: string): string {
  switch (kind) {
    case "standard":
      return "Starts at 2 a day, then 3, then 5. After that, at most 20% more on days you actually send.";
    case "conservative":
      return "Starts at 2 a day and stays there an extra day, then 3, then 4. After that, at most 10% more on days you send.";
    case "fast":
      return "Starts at 2 a day, then 4, 6, and 9. After that, at most 30% more on days you send.";
    case "off":
      return "No climb. The mailbox’s own daily cap still applies.";
    case "custom":
      return "Stays at the number you set. It does not climb on its own.";
    default:
      return "";
  }
}

export function warmupKindLabel(kind: string): string {
  switch (kind) {
    case "standard":
      return "Standard ramp";
    case "conservative":
      return "Conservative ramp";
    case "fast":
      return "Faster ramp";
    case "off":
      return "No ramp";
    case "custom":
      return "Custom ramp";
    default:
      return kind;
  }
}

export function readableMail(raw?: string): string {
  if (!raw) {
    return "";
  }
  let s = raw
    .replace(/&nbsp;/gi, " ")
    .replace(/&#39;|&apos;/gi, "'")
    .replace(/&quot;/gi, '"')
    .replace(/&#(\d+);/g, (_, n) => String.fromCharCode(Number(n)))
    .replace(/&#x([0-9a-f]+);/gi, (_, n) =>
      String.fromCharCode(parseInt(n, 16)),
    )
    .replace(/&lt;/gi, "<")
    .replace(/&gt;/gi, ">")
    .replace(/&amp;/gi, "&");
  s = s
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/p>/gi, "\n")
    .replace(/<[^>]+>/g, " ");
  return s.replace(/[ \t]+\n/g, "\n").replace(/\n{3,}/g, "\n\n").trim();
}

export function authKindLabel(kind: string): string {
  switch (kind) {
    case "app_password":
      return "Gmail app password";
    case "smtp_imap":
      return "SMTP";
    case "service_account":
      return "Google Workspace";
    case "oauth":
      return "Google";
    default:
      return kind;
  }
}

export type ReviewFilter = "pending" | "tomorrow" | "undo" | "checking" | "sent";

export function reviewBucket(
  draft: Draft,
  mailbox?: Pick<Mailbox, "open_today" | "sending_now">,
): ReviewFilter | "other" {
  if (draft.status === "pending") {
    if (mailboxIdle(mailbox)) {
      return "tomorrow";
    }
    return "pending";
  }
  switch (draft.ledger_state) {
    case "held":
      return "undo";
    case "unknown":
    case "reserved":
    case "sending":
      return "checking";
    case "sent":
      return "sent";
    default:
      return "other";
  }
}

export function sendStatus(
  draft: Draft,
  mailbox?: Pick<Mailbox, "open_today" | "sending_now" | "next_open">,
  nowMs?: number,
): string {
  if (draft.status === "pending") {
    const when = mailboxSendsAtLabel(mailbox);
    if (when) {
      return when;
    }
    return "Needs approval";
  }
  switch (draft.ledger_state) {
    case "held":
      return undoLabel(draft.release_at, nowMs);
    case "reserved":
    case "sending":
      return "Sending";
    case "sent":
      return "Sent";
    case "unknown":
      return "Checking if it sent";
    case "cancelled":
      return "Cancelled";
    case "released":
      return "Not sent";
    case "failed":
      return "Could not send";
    default:
      return draft.status === "approved" ? "Approved" : draft.status;
  }
}

export function undoLabel(releaseAt?: string, nowMs?: number): string {
  if (!releaseAt || nowMs == null) {
    return "You can still undo";
  }
  const ms = new Date(releaseAt).getTime() - nowMs;
  if (ms <= 0) {
    return "You can still undo";
  }
  const mins = Math.max(1, Math.ceil(ms / 60_000));
  return `You can still undo · ${mins} min left`;
}

export function contactName(c: {
  first_name?: string;
  last_name?: string;
  email: string;
}): string {
  const n = [c.first_name, c.last_name].filter(Boolean).join(" ");
  return n || c.email;
}
