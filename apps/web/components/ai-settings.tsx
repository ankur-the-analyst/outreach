"use client";

import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Panel } from "@/components/page-header";
import type { AIProvider, ModelSettings } from "@/lib/api";
import { fieldSelect } from "@/lib/labels";

function pick(settings?: ModelSettings | null): AIProvider | undefined {
  const list = settings?.providers ?? [];
  return (
    list.find((p) => p.kind === settings?.kind) ??
    list[0]
  );
}

function withCurrent(list: string[] | undefined, current: string): string[] {
  const out = [...(list ?? [])];
  if (current && !out.includes(current)) {
    out.unshift(current);
  }
  return out;
}

export function AISettingsForm({
  settings,
}: {
  settings?: ModelSettings | null;
}) {
  const providers = settings?.providers ?? [];
  const initial = pick(settings);
  const [kind, setKind] = useState(settings?.kind ?? initial?.kind ?? "openai");
  const [apiKey, setAPIKey] = useState("");
  const [baseURL, setBaseURL] = useState(settings?.base_url ?? "");
  const [composeModel, setComposeModel] = useState(
    settings?.compose_model || initial?.compose_models?.[0] || "",
  );
  const [embedModel, setEmbedModel] = useState(
    settings?.embed_model || initial?.embed_models?.[0] || "",
  );
  const [msg, setMsg] = useState("");
  const [pending, setPending] = useState(false);

  const spec = useMemo(
    () => providers.find((p) => p.kind === kind) ?? initial,
    [providers, kind, initial],
  );
  const composeOptions = withCurrent(spec?.compose_models, composeModel);
  const embedOptions = withCurrent(spec?.embed_models, embedModel);

  function onKind(next: string) {
    setKind(next);
    const p = providers.find((x) => x.kind === next);
    if (!p?.custom_models) {
      setComposeModel(p?.compose_models?.[0] ?? "");
      setEmbedModel(p?.embed_models?.[0] ?? "");
    }
  }

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
          onChange={(e) => onKind(e.target.value)}
          className={fieldSelect}
        >
          {providers.map((p) => (
            <option key={p.kind} value={p.kind}>
              {p.label}
            </option>
          ))}
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
      {spec?.needs_base_url ? (
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
        {spec?.custom_models ? (
          <Input
            id="ai-compose"
            required
            value={composeModel}
            onChange={(e) => setComposeModel(e.target.value)}
          />
        ) : (
          <select
            id="ai-compose"
            required
            value={composeModel}
            onChange={(e) => setComposeModel(e.target.value)}
            className={fieldSelect}
          >
            {composeOptions.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        )}
      </div>
      {spec?.embed ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="ai-embed">Embed model</Label>
          {spec.custom_models ? (
            <Input
              id="ai-embed"
              required
              value={embedModel}
              onChange={(e) => setEmbedModel(e.target.value)}
            />
          ) : (
            <select
              id="ai-embed"
              required
              value={embedModel}
              onChange={(e) => setEmbedModel(e.target.value)}
              className={fieldSelect}
            >
              {embedOptions.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          )}
        </div>
      ) : (
        <p className="font-body-sm text-muted-foreground">
          This provider cannot embed. Pick OpenAI or Google, or an
          OpenAI-compatible endpoint.
        </p>
      )}
      {msg ? (
        <p role="status" className="font-body-md text-muted-foreground">
          {msg}
        </p>
      ) : null}
      <Button
        type="submit"
        className="rounded-md"
        pending={pending}
        disabled={!spec?.embed}
      >
        Save
      </Button>
    </form>
    </Panel>
  );
}
