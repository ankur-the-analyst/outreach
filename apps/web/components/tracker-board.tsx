"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Input } from "@/components/ui/input";
import { Pill } from "@/components/page-header";
import type { TrackerRow } from "@/lib/api";
import { fieldSelect, trackerStageLabel } from "@/lib/labels";

const stages = ["", "drafted", "sent", "replied", "interview", "offer", "closed"] as const;
const settableStages = [
  "followed_up",
  "interview",
  "offer",
  "rejected",
  "closed",
] as const;

function stageTone(
  stage: string,
): "muted" | "primary" | "secondary" | "success" | "danger" {
  if (stage === "replied") {
    return "success";
  }
  if (stage === "sent") {
    return "secondary";
  }
  if (stage === "drafted" || stage === "interview") {
    return "primary";
  }
  if (stage === "offer") {
    return "success";
  }
  if (stage === "bounced" || stage === "unsubscribed" || stage === "rejected") {
    return "danger";
  }
  return "muted";
}

export function TrackerBoard({ rows }: { rows: TrackerRow[] }) {
  const [q, setQ] = useState("");
  const [stage, setStage] = useState("");
  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase();
    return rows.filter((row) => {
      if (stage && row.stage !== stage) {
        return false;
      }
      if (!needle) {
        return true;
      }
      return [row.contact, row.email, row.company, row.campaign]
        .filter(Boolean)
        .join(" ")
        .toLowerCase()
        .includes(needle);
    });
  }, [rows, q, stage]);

  if (rows.length === 0) {
    return (
      <p className="font-body-md text-muted-foreground">
        No one in a campaign yet.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search"
          className="max-w-xs"
          aria-label="Search tracker"
        />
        {stages.map((s) => (
          <button
            key={s || "all"}
            type="button"
            onClick={() => setStage(s)}
            className={`rounded-full px-2.5 py-0.5 font-label-sm ${
              stage === s
                ? "bg-accent text-primary"
                : "bg-muted text-muted-foreground"
            }`}
          >
            {s ? (trackerStageLabel[s] ?? s) : "All"}
          </button>
        ))}
      </div>
      {filtered.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No matching rows.</p>
      ) : (
        <div className="overflow-x-auto rounded-md border border-border bg-card">
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-border">
                <th className="px-4 py-3 font-label-sm text-muted-foreground">
                  Contact
                </th>
                <th className="px-4 py-3 font-label-sm text-muted-foreground">
                  Company
                </th>
                <th className="px-4 py-3 font-label-sm text-muted-foreground">
                  Campaign
                </th>
                <th className="px-4 py-3 font-label-sm text-muted-foreground">
                  Stage
                </th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((row) => (
                <tr key={row.id} className="border-b border-border last:border-0">
                  <td className="px-4 py-3">
                    <p className="font-label-md text-foreground">{row.contact}</p>
                    <p className="font-body-sm text-muted-foreground">
                      {row.email}
                    </p>
                  </td>
                  <td className="px-4 py-3 font-body-md text-foreground">
                    {row.company || "—"}
                  </td>
                  <td className="px-4 py-3 font-body-md text-foreground">
                    {row.campaign}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <Pill tone={stageTone(row.stage)}>
                        {trackerStageLabel[row.stage] ?? row.stage}
                      </Pill>
                      {row.application_id ? (
                        <StageSelect
                          applicationID={row.application_id}
                          stage={row.stage}
                        />
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function StageSelect({
  applicationID,
  stage,
}: {
  applicationID: string;
  stage: string;
}) {
  const router = useRouter();
  const [value, setValue] = useState(
    (settableStages as readonly string[]).includes(stage) ? stage : "",
  );
  const [pending, setPending] = useState(false);

  async function onChange(next: string) {
    setValue(next);
    if (!next) {
      return;
    }
    setPending(true);
    await fetch(`/api/v1/applications/${applicationID}`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ stage: next }),
    });
    setPending(false);
    router.refresh();
  }

  return (
    <select
      aria-label="Set stage"
      value={value}
      disabled={pending}
      onChange={(e) => onChange(e.target.value)}
      className={`${fieldSelect} w-auto min-w-[8rem]`}
    >
      <option value="">Set stage</option>
      {settableStages.map((s) => (
        <option key={s} value={s}>
          {trackerStageLabel[s] ?? s}
        </option>
      ))}
    </select>
  );
}
