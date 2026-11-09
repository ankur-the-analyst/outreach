"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState, PageBody, PageHeader } from "@/components/page-header";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { Profile } from "@/lib/api";
import { fieldTextarea } from "@/lib/labels";

export function ProfileBook({ profiles }: { profiles: Profile[] }) {
  const [createOpen, setCreateOpen] = useState(false);
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader
        title="Profiles"
        actions={
          <Button
            type="button"
            className="rounded-md"
            onClick={() => setCreateOpen(true)}
          >
            Add a profile
          </Button>
        }
      />
      <PageBody>
        {profiles.length === 0 ? (
          <EmptyState
            title="No profiles yet"
            description="A profile is who you are writing as, plus resume text for job fit."
            action={
              <Button
                type="button"
                className="rounded-md"
                onClick={() => setCreateOpen(true)}
              >
                Add a profile
              </Button>
            }
          />
        ) : (
          <ul className="flex flex-col gap-3">
            {profiles.map((p) => (
              <li
                key={p.id}
                className="rounded-md border border-border bg-card p-4"
              >
                <p className="font-label-md text-foreground">{p.name}</p>
                {p.about ? (
                  <p className="mt-1 whitespace-pre-wrap font-body-md text-muted-foreground">
                    {p.about}
                  </p>
                ) : null}
                <ResumePaste profileID={p.id} />
              </li>
            ))}
          </ul>
        )}
      </PageBody>
      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>Add a profile</SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            <CreateProfile onDone={() => setCreateOpen(false)} />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function CreateProfile({ onDone }: { onDone?: () => void }) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [about, setAbout] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/profiles", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, about }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not save profile");
      return;
    }
    setName("");
    setAbout("");
    onDone?.();
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="profile-name">Name</Label>
        <Input
          id="profile-name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="profile-about">About</Label>
        <textarea
          id="profile-about"
          rows={4}
          value={about}
          onChange={(e) => setAbout(e.target.value)}
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

function ResumePaste({ profileID }: { profileID: string }) {
  const router = useRouter();
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch(`/api/v1/profiles/${profileID}/resumes`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        file_name: "resume.txt",
        mime: "text/plain",
        text,
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not save resume");
      return;
    }
    setText("");
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="mt-4 flex flex-col gap-2"
    >
      <Label htmlFor={`resume-${profileID}`}>Paste resume text</Label>
      <textarea
        id={`resume-${profileID}`}
        required
        rows={4}
        value={text}
        onChange={(e) => setText(e.target.value)}
        className={fieldTextarea}
      />
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" variant="outline" className="w-fit rounded-md" pending={pending}>
        Save resume
      </Button>
    </form>
  );
}
