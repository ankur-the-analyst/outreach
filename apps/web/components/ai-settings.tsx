"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Panel } from "@/components/page-header";
import type { ModelSettings } from "@/lib/api";
import { fieldSelect } from "@/lib/labels";

export function AISettingsForm({
  settings,
}: {
  settings?: ModelSettings | null;
}) {
  const [kind, setKind] = useState(settings?.kind ?? "openai");
  const [apiKey, setAPIKey] = useState("");
  const [baseURL, setBaseURL] = useState(settings?.base_url ?? "");
  const [composeModel, setComposeModel] = useState(
    settings?.compose_model ?? "",
  );
  const [embedModel, setEmbedModel] = useState(settings?.embed_model ?? "");
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    setPending(true);
    const res = await fetch("/api/v1/settings/ai", {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        kind,
        api_key: apiKey || undefined,
        base_url: baseURL,
        compose_model: composeModel,
        embed_model: embedModel,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setMsg(data?.error?.message ?? "Could not save models");
      return;
    }
    setAPIKey("");
    setMsg("Saved");
  }

  return (
    <Panel
      title="Models"
      description="Keys stay in this workspace. The model cannot send email."
    >
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ai-kind">Provider</Label>
        <select
          id="ai-kind"
          value={kind}
          onChange={(e) => setKind(e.target.value)}
          className={fieldSelect}
        >
          <option value="openai">OpenAI</option>
          <option value="google">Google</option>
          <option value="anthropic">Anthropic</option>
          <option value="openai_compat">OpenAI-compatible</option>
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ai-key">API key</Label>
        <Input
          id="ai-key"
          type="password"
          autoComplete="off"
          value={apiKey}
          onChange={(e) => setAPIKey(e.target.value)}
          required={!settings?.has_key}
          placeholder={
            settings?.has_key
              ? `Saved ${settings.hint ?? ""}`.trim()
              : undefined
          }
        />
      </div>
      {kind === "openai_compat" ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="ai-base">Base URL</Label>
          <Input
            id="ai-base"
            value={baseURL}
            onChange={(e) => setBaseURL(e.target.value)}
            required
          />
        </div>
      ) : null}
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ai-compose">Compose model</Label>
        <Input
          id="ai-compose"
          required
          value={composeModel}
          onChange={(e) => setComposeModel(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="ai-embed">Embed model</Label>
        <Input
          id="ai-embed"
          required
          value={embedModel}
          onChange={(e) => setEmbedModel(e.target.value)}
        />
      </div>
      {msg ? (
        <p role="status" className="font-body-md text-muted-foreground">
          {msg}
        </p>
      ) : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Save
      </Button>
    </form>
    </Panel>
  );
}
