"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Pill } from "@/components/page-header";
import type { Fit, Job, Profile } from "@/lib/api";
import { fieldSelect, fieldTextarea, jobAnalysisLabel } from "@/lib/labels";

export function JobBoard({
  jobs,
  nextCursor,
  profiles,
  fits = [],
}: {
  jobs: Job[];
  nextCursor?: string;
  profiles: Profile[];
  fits?: Fit[];
}) {
  const [rows, setRows] = useState(jobs);
  const [cursor, setCursor] = useState(nextCursor);
  const [morePending, setMorePending] = useState(false);

  useEffect(() => {
    setRows(jobs);
    setCursor(nextCursor);
  }, [jobs, nextCursor]);

  async function loadMore() {
    if (!cursor) {
      return;
    }
    setMorePending(true);
    const res = await fetch(
      `/api/v1/jobs?cursor=${encodeURIComponent(cursor)}`,
      { credentials: "include" },
    );
    setMorePending(false);
    if (!res.ok) {
      return;
    }
    const data = (await res.json()) as {
      jobs?: Job[];
      next_cursor?: string;
    };
    setRows((cur) => [...cur, ...(data.jobs ?? [])]);
    setCursor(data.next_cursor);
  }

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
      <div className="lg:col-span-7">
        {rows.length === 0 ? (
          <p className="font-body-md text-muted-foreground">No jobs yet.</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {rows.map((job) => (
              <li
                key={job.id}
                className="rounded-md border border-border bg-card p-4"
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <p className="font-label-md text-foreground">
                    {job.title || "Untitled job"}
                  </p>
                  <Pill
                    tone={
                      job.analysis_status === "ready"
                        ? "success"
                        : job.analysis_status === "failed"
                          ? "danger"
                          : "muted"
                    }
                  >
                    {jobAnalysisLabel(job.analysis_status)}
                  </Pill>
                </div>
                {job.source_url ? (
                  <p className="mt-1 break-all font-body-md text-muted-foreground">
                    {job.source_url}
                  </p>
                ) : null}
                {job.analysis?.summary ? (
                  <p className="mt-2 font-body-md text-foreground">
                    {job.analysis.summary}
                  </p>
                ) : null}
                <p className="mt-2 line-clamp-4 whitespace-pre-wrap font-body-md text-muted-foreground">
                  {job.raw_text}
                </p>
                <CheckFit
                  jobID={job.id}
                  profiles={profiles}
                  fits={fits.filter((f) => f.job_id === job.id)}
                />
              </li>
            ))}
          </ul>
        )}
        {cursor ? (
          <Button
            type="button"
            variant="outline"
            className="mt-3 w-fit rounded-md"
            pending={morePending}
            onClick={loadMore}
          >
            Show more
          </Button>
        ) : null}
      </div>
      <div className="lg:col-span-5">
        <CreateJob />
      </div>
    </div>
  );
}

function CreateJob() {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [sourceURL, setSourceURL] = useState("");
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/jobs", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        title,
        source_url: sourceURL,
        text,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not save job");
      return;
    }
    setTitle("");
    setSourceURL("");
    setText("");
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">Add a job</h2>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="job-title">Title</Label>
        <Input
          id="job-title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="job-url">Link</Label>
        <Input
          id="job-url"
          type="url"
          value={sourceURL}
          onChange={(e) => setSourceURL(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="job-text">Job description</Label>
        <textarea
          id="job-text"
          required
          rows={8}
          value={text}
          onChange={(e) => setText(e.target.value)}
          className={fieldTextarea}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" className="rounded-md" pending={pending}>
        Save
      </Button>
    </form>
  );
}

type FitView = {
  score?: number;
  summary?: string;
};

function viewFit(f: Fit | null): FitView | null {
  if (!f) {
    return null;
  }
  let result: { score?: number; summary?: string } | undefined;
  if (typeof f.result === "string") {
    try {
      result = JSON.parse(f.result) as { score?: number; summary?: string };
    } catch {
      result = undefined;
    }
  } else {
    result = f.result;
  }
  return { score: f.score ?? result?.score, summary: result?.summary };
}

function CheckFit({
  jobID,
  profiles,
  fits,
}: {
  jobID: string;
  profiles: Profile[];
  fits: Fit[];
}) {
  const [profileID, setProfileID] = useState(profiles[0]?.id ?? "");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const [fit, setFit] = useState<FitView | null>(
    viewFit(fits.find((f) => f.profile_id === (profiles[0]?.id ?? "")) ?? null),
  );

  function pickProfile(id: string) {
    setProfileID(id);
    setFit(viewFit(fits.find((f) => f.profile_id === id) ?? null));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setFit(null);
    setPending(true);
    const res = await fetch(`/api/v1/jobs/${jobID}/fit`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ profile_id: profileID }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not check this person");
      return;
    }
    const data = (await res.json()) as Fit;
    setFit(viewFit(data));
  }

  if (profiles.length === 0) {
    return (
      <p className="mt-3 font-body-md text-muted-foreground">
        Add a profile first to check a match.
      </p>
    );
  }

  const score = fit?.score;
  const summary = fit?.summary;

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="mt-4 flex flex-col gap-2"
    >
      <Label htmlFor={`fit-${jobID}`}>Check against</Label>
      <select
        id={`fit-${jobID}`}
        required
        value={profileID}
        onChange={(e) => pickProfile(e.target.value)}
        className={fieldSelect}
      >
        {profiles.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name}
          </option>
        ))}
      </select>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      {fit && !summary && (score == null || score === 0) ? (
        <p className="font-body-md text-muted-foreground">
          Couldn&apos;t score this with the current model.
        </p>
      ) : null}
      {score != null && (summary || score !== 0) ? (
        <p className="font-body-md text-foreground">Score {score}</p>
      ) : null}
      {summary ? (
        <p className="whitespace-pre-wrap font-body-md text-muted-foreground">
          {summary}
        </p>
      ) : null}
      <Button
        type="submit"
        variant="outline"
        className="w-fit rounded-md"
        pending={pending}
      >
        Check match
      </Button>
    </form>
  );
}
