"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Profile } from "@/lib/api";
import { fieldTextarea } from "@/lib/labels";

export function ProfileBook({ profiles }: { profiles: Profile[] }) {
  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
      <div className="lg:col-span-7">
        {profiles.length === 0 ? (
          <p className="font-body-md text-muted-foreground">No profiles yet.</p>
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
      </div>
      <div className="lg:col-span-5">
        <CreateProfile />
      </div>
    </div>
  );
}

function CreateProfile() {
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
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex max-w-lg flex-col gap-4 rounded-md border border-border bg-card p-6"
    >
      <h2 className="font-headline-md text-foreground">Add a profile</h2>
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
