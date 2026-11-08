"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Job, Profile } from "@/lib/api";
import { fieldSelect, fieldTextarea } from "@/lib/labels";

export function JobBoard({
  jobs,
  profiles,
}: {
  jobs: Job[];
  profiles: Profile[];
}) {
  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
      <div className="lg:col-span-7">
        {jobs.length === 0 ? (
          <p className="font-body-md text-muted-foreground">No jobs yet.</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {jobs.map((job) => (
              <li
                key={job.id}
                className="rounded-md border border-border bg-card p-4"
              >
                <p className="font-label-md text-foreground">
                  {job.title || "Untitled job"}
                </p>
                {job.source_url ? (
                  <p className="mt-1 break-all font-body-md text-muted-foreground">
                    {job.source_url}
                  </p>
                ) : null}
                <p className="mt-2 line-clamp-4 whitespace-pre-wrap font-body-md text-muted-foreground">
                  {job.raw_text}
                </p>
                <CheckFit jobID={job.id} profiles={profiles} />
              </li>
            ))}
          </ul>
        )}
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

type FitResult = {
  score?: number;
  result?: { score?: number; summary?: string };
};

function CheckFit({
  jobID,
  profiles,
}: {
  jobID: string;
  profiles: Profile[];
}) {
  const [profileID, setProfileID] = useState(profiles[0]?.id ?? "");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const [fit, setFit] = useState<FitResult | null>(null);

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
    const data = (await res.json()) as {
      score?: number;
      result?: { score?: number; summary?: string } | string;
    };
    let result: { score?: number; summary?: string } | undefined;
    if (typeof data.result === "string") {
      try {
        result = JSON.parse(data.result) as {
          score?: number;
          summary?: string;
        };
      } catch {
        result = undefined;
      }
    } else {
      result = data.result;
    }
    setFit({ score: data.score, result });
  }

  if (profiles.length === 0) {
    return (
      <p className="mt-3 font-body-md text-muted-foreground">
        Add a profile first to check a match.
      </p>
    );
  }

  const score = fit?.score ?? fit?.result?.score;
  const summary = fit?.result?.summary;

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
        onChange={(e) => setProfileID(e.target.value)}
        className={fieldSelect}
      >
        {profiles.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name}
          </option>
        ))}
      </select>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      {score != null ? (
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
