import type { Draft } from "@/lib/api";

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

export function replyModeLabel(mode: string): string {
  if (mode === "off") {
    return "Don't draft replies";
  }
  return "Draft replies for review";
}

export function mailboxUsageLabel(used: number, cap: number): string {
  return `${used} of ${cap} · last 24h`;
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
      return "OAuth";
    default:
      return kind;
  }
}

export type ReviewFilter = "pending" | "undo" | "checking" | "sent";

export function reviewBucket(draft: Draft): ReviewFilter | "other" {
  if (draft.status === "pending") {
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

export function sendStatus(draft: Draft): string {
  if (draft.status === "pending") {
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
