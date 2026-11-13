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

export type AIProvider = {
  kind: string;
  label: string;
  needs_base_url: boolean;
  embed: boolean;
  custom_models?: boolean;
  compose_models: string[];
  embed_models: string[];
};

export type ModelSettings = {
  kind?: string;
  has_key: boolean;
  hint?: string;
  base_url?: string;
  compose_model?: string;
  embed_model?: string;
  providers?: AIProvider[];
};

export function getModelSettings() {
  return apiGet<ModelSettings>("/api/v1/settings/ai");
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
  reply_mode?: string;
  warmup_kind?: string;
  domain?: string;
  domain_daily_limit?: number;
  open_today?: boolean;
  sending_now?: boolean;
  next_open?: string;
};

export function getMailboxes() {
  return apiGet<{ mailboxes: Mailbox[] }>("/api/v1/mailboxes");
}

export type MailboxPool = {
  id: string;
  name: string;
  mailboxes: { id: string; email: string }[];
};

export function getMailboxPools() {
  return apiGet<{ pools: MailboxPool[] }>("/api/v1/mailbox-pools");
}

export type WarmupProfile = {
  id: string;
  name: string;
  kind: string;
  first_day?: number;
};

export function getWarmupProfiles() {
  return apiGet<{ profiles: WarmupProfile[] }>("/api/v1/warmup-profiles");
}

export type HealthSample = {
  id: number;
  metric: string;
  value: number;
  window: string;
  created_at: string;
};

export function getHealthSamples() {
  return apiGet<{ samples: HealthSample[] }>("/api/v1/health-samples");
}

export type SendingDomain = {
  id: string;
  domain: string;
  daily_limit: number;
};

export function getSendingDomains() {
  return apiGet<{ domains: SendingDomain[] }>("/api/v1/sending-domains");
}

export type WorkspaceKey = {
  id: string;
  domain: string;
  client_email: string;
  client_id: string;
};

export function getWorkspaceKeys() {
  return apiGet<{ service_accounts: WorkspaceKey[] }>(
    "/api/v1/google-service-accounts",
  );
}

export async function getMailboxUsage(mailboxID: string) {
  const data = await apiGet<{ used: number }>(
    `/api/v1/mailboxes/${mailboxID}/usage`,
  );
  return data?.used ?? 0;
}

export async function getMailboxUsageByID(mailboxes: Mailbox[]) {
  const entries = await Promise.all(
    mailboxes.map(async (mb) => [mb.id, await getMailboxUsage(mb.id)] as const),
  );
  return Object.fromEntries(entries) as Record<string, number>;
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
  facts_used?: { claim: string; evidence_id: string }[];
  attachment_name?: string;
  attachment_from?: string;
  opened_at?: string;
  clicked_at?: string;
};

export function getDrafts() {
  return apiGet<{ drafts: Draft[]; next_cursor?: string }>("/api/v1/drafts");
}

export type EvidenceItem = {
  id: string;
  profile_id?: string;
  text: string;
  source_ref?: string;
  tags: string[];
  do_not_use: boolean;
  confirmed: boolean;
  embedding_model?: string;
  created_at: string;
};

export function getEvidence() {
  return apiGet<{ evidence: EvidenceItem[]; next_cursor?: string }>(
    "/api/v1/evidence",
  );
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

export function getContacts(cursor?: string) {
  const q = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
  return apiGet<{ contacts: Contact[]; next_cursor?: string }>(
    `/api/v1/contacts${q}`,
  );
}

export type NeverEmail = {
  kind: string;
  value?: string;
  reason: string;
  created_at?: string;
};

export function getNeverEmail() {
  return apiGet<{ entries: NeverEmail[] }>("/api/v1/suppression");
}

export type Campaign = {
  id: string;
  name: string;
  profile_id?: string;
  intent: string;
  status: string;
  commercial: boolean;
  list_unsubscribe: boolean;
  new_contacts_per_day?: number;
  sender_strategy: string;
  steps: {
    position: number;
    kind: string;
    wait_days?: number;
    content?: string;
    template_id?: string;
  }[];
  senders: { mailbox_id: string }[];
  pool_id?: string;
  attach?: string;
  track_opens?: boolean;
  track_clicks?: boolean;
  created_at: string;
};

export type SavedEmail = {
  id: string;
  name: string;
  subject: string;
  body_text: string;
  body_html?: string;
  design_json?: unknown;
};

export function getSavedEmails() {
  return apiGet<{ templates: SavedEmail[] }>("/api/v1/templates");
}

export function getCampaigns() {
  return apiGet<{ campaigns: Campaign[]; next_cursor?: string }>(
    "/api/v1/campaigns",
  );
}

export type CampaignPerson = {
  id: string;
  campaign_id: string;
  contact_id: string;
  job_id?: string;
  profile_id?: string;
  status: string;
  created_at: string;
};

export async function getCampaignPeople(campaignID: string) {
  const data = await apiGet<{ enrolments: CampaignPerson[] }>(
    `/api/v1/campaigns/${campaignID}/enrolments`,
  );
  return data?.enrolments ?? [];
}

export type Profile = {
  id: string;
  name: string;
  about?: string;
  created_at: string;
};

export function getProfiles() {
  return apiGet<{ profiles: Profile[] }>("/api/v1/profiles");
}

export type Job = {
  id: string;
  title?: string;
  source_url?: string;
  raw_text: string;
  analysis?: { summary?: string; title?: string } | null;
  analysis_status: string;
  created_at: string;
};

export function getJobs() {
  return apiGet<{ jobs: Job[]; next_cursor?: string }>("/api/v1/jobs");
}

export type Fit = {
  id: string;
  job_id: string;
  profile_id: string;
  score?: number;
  result?: { score?: number; summary?: string } | string;
};

export function getFits() {
  return apiGet<{ fits: Fit[] }>("/api/v1/fits");
}

export type TrackerRow = {
  id: string;
  application_id?: string;
  contact: string;
  email: string;
  company?: string;
  campaign: string;
  campaign_id: string;
  stage: string;
  opened_at?: string;
  clicked_at?: string;
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
