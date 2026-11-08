import { cookies } from "next/headers";

const apiURL = process.env.API_URL ?? "http://127.0.0.1:8080";

export type Me = {
  user_id: string;
  workspace_id: string;
};

export type Workspace = {
  id: string;
  name: string;
  sends_paused: boolean;
  role: string;
};

async function apiGet<T>(path: string): Promise<T | null> {
  const jar = await cookies();
  const sid = jar.get("session")?.value;
  if (!sid) {
    return null;
  }
  const res = await fetch(`${apiURL}${path}`, {
    headers: { cookie: `session=${sid}` },
    cache: "no-store",
  });
  if (!res.ok) {
    return null;
  }
  return (await res.json()) as T;
}

export function getMe() {
  return apiGet<Me>("/api/v1/me");
}

export function getWorkspace() {
  return apiGet<Workspace>("/api/v1/workspace");
}

export type SenderIdentity = {
  legal_name: string;
  reply_address: string;
  postal_address: string;
};

export function getSenderIdentity() {
  return apiGet<SenderIdentity>("/api/v1/sender-identity");
}

export type Member = {
  user_id: string;
  email: string;
  name: string;
  role: string;
};

export function getMembers() {
  return apiGet<{ members: Member[] }>("/api/v1/workspace/members");
}

export type Mailbox = {
  id: string;
  email: string;
  display_name: string;
  auth_kind: string;
  provider: string;
  status: string;
  timezone: string;
  smtp_host: string;
  smtp_port: number;
  effective_daily_limit: number;
};

export function getMailboxes() {
  return apiGet<{ mailboxes: Mailbox[] }>("/api/v1/mailboxes");
}

export type Draft = {
  id: string;
  mailbox_id: string;
  to: string;
  rfc_message_id: string;
  kind: string;
  subject: string;
  body_html: string;
  body_text: string;
  status: string;
  ledger_id?: string;
  ledger_state?: string;
  release_at?: string;
};

export function getDrafts() {
  return apiGet<{ drafts: Draft[] }>("/api/v1/drafts");
}

export type EvidenceItem = {
  id: string;
  text: string;
  source_ref?: string;
  tags: string[];
  do_not_use: boolean;
  confirmed: boolean;
  embedding_model?: string;
  created_at: string;
};

export function getEvidence() {
  return apiGet<{ evidence: EvidenceItem[] }>("/api/v1/evidence");
}

export type Contact = {
  id: string;
  email: string;
  first_name?: string;
  last_name?: string;
  title?: string;
  company_id?: string;
  role_class?: string;
  provenance: { source: string; added_by: string; at: string };
  readiness: string;
  created_at: string;
};

export function getContacts() {
  return apiGet<{ contacts: Contact[] }>("/api/v1/contacts");
}

export type Campaign = {
  id: string;
  name: string;
  intent: string;
  status: string;
  commercial: boolean;
  list_unsubscribe: boolean;
  new_contacts_per_day?: number;
  sender_strategy: string;
  steps: { position: number; kind: string; wait_days?: number; content?: string }[];
  senders: { mailbox_id: string }[];
  created_at: string;
};

export function getCampaigns() {
  return apiGet<{ campaigns: Campaign[] }>("/api/v1/campaigns");
}

export type TrackerRow = {
  id: string;
  contact: string;
  email: string;
  company?: string;
  campaign: string;
  campaign_id: string;
  stage: string;
  updated_at: string;
};

export function getTracker() {
  return apiGet<{ rows: TrackerRow[] }>("/api/v1/tracker");
}

export type ThreadMessage = {
  id: string;
  direction: string;
  rfc_message_id: string;
  from: string;
  to: string[];
  subject: string;
  body_text: string;
  classification?: {
    class?: string;
    reply_draft?: boolean;
    suppress?: boolean;
  };
};

export type Thread = {
  id: string;
  mailbox_id: string;
  subject: string;
  unread: boolean;
  last_message_at?: string;
  messages?: ThreadMessage[];
};

export function getThreads(mailboxID: string) {
  return apiGet<{ threads: Thread[] }>(`/api/v1/mailboxes/${mailboxID}/threads`);
}
