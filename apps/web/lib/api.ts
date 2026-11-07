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
