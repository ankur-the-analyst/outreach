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
      return undoLabel(draft.release_at);
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

export function undoLabel(releaseAt?: string): string {
  if (!releaseAt) {
    return "You can still undo";
  }
  const ms = new Date(releaseAt).getTime() - Date.now();
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
