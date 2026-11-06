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
