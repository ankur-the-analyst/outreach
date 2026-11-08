"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { WorkspaceKey } from "@/lib/api";
import { fieldTextarea } from "@/lib/labels";

function workspaceKeyError(message: string): string {
  switch (message) {
    case "cannot attach a service account to gmail.com":
      return "Use a Workspace domain, not personal Gmail";
    case "service account already exists for domain":
      return "This domain already has a key";
    case "mailbox: invalid service account json":
      return "Paste a valid Google JSON key";
    case "mailbox: domain required":
      return "Enter a domain";
    default:
      return message;
  }
}

export function WorkspaceKeys({ keys }: { keys: WorkspaceKey[] }) {
  return (
    <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-12">
      <div className="lg:col-span-7">
        <h2 className="mb-3 font-headline-md text-foreground">
          Google Workspace keys
        </h2>
        {keys.length === 0 ? (
          <p className="font-body-md text-muted-foreground">
            No Workspace keys yet.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {keys.map((k) => (
              <li
                key={k.id}
                className="rounded-md border border-border bg-card p-4"
              >
                <p className="font-label-md text-foreground">{k.domain}</p>
                <p className="mt-1 break-all font-body-sm text-muted-foreground">
                  {k.client_email}
                </p>
              </li>
            ))}
          </ul>
        )}
      </div>
      <div className="lg:col-span-5">
        <AddWorkspaceKey />
      </div>
    </div>
  );
}

function AddWorkspaceKey() {
  const router = useRouter();
  const [domain, setDomain] = useState("");
  const [jsonKey, setJsonKey] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    let parsed: unknown;
    try {
      parsed = JSON.parse(jsonKey);
    } catch {
      setError("Paste a valid Google JSON key");
      return;
    }
    setPending(true);
    const res = await fetch("/api/v1/google-service-accounts", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ domain, json_key: parsed }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(workspaceKeyError(data?.error?.message ?? "Could not save key"));
      return;
    }
    setDomain("");
    setJsonKey("");
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-5"
    >
      <h2 className="font-headline-md text-foreground">Add a Workspace key</h2>
      <p className="font-body-md text-muted-foreground">
        One JSON key per sending domain. Not for @gmail.com.
      </p>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ws-domain">Domain</Label>
        <Input
          id="ws-domain"
          required
          placeholder="acme.com"
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ws-json">JSON key</Label>
        <textarea
          id="ws-json"
          required
          rows={6}
          value={jsonKey}
          onChange={(e) => setJsonKey(e.target.value)}
          className={fieldTextarea}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Save key
      </Button>
    </form>
  );
}
