"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

function oauthMessage(code: string): string {
  switch (code) {
    case "google_denied":
      return "Google sign-in was cancelled";
    case "google_not_configured":
      return "Google sign-in is not configured on the server";
    case "google_state":
    case "google_code":
    case "google_token":
      return "Could not finish Google sign-in. Try again.";
    default:
      return "Could not connect Google mailbox";
  }
}

export function ConnectMailboxControl() {
  const [open, setOpen] = useState(false);
  const [notice, setNotice] = useState("");
  useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    if (q.get("connected") === "1") {
      setNotice("Mailbox connected");
    }
    const err = q.get("error");
    if (err) {
      setNotice(oauthMessage(err));
    }
  }, []);
  return (
    <div className="flex flex-wrap items-center gap-2">
      {notice ? (
        <p className="font-body-sm text-muted-foreground">{notice}</p>
      ) : null}
      <Button
        type="button"
        className="rounded-md"
        onClick={() => {
          window.location.href = "/api/v1/mailboxes/oauth/google/start";
        }}
      >
        Connect Gmail
      </Button>
      <Button
        type="button"
        variant="outline"
        className="rounded-md"
        onClick={() => setOpen(true)}
      >
        Connect SMTP
      </Button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>Connect mailbox</SheetTitle>
          </SheetHeader>
          <div className="px-4 pb-4">
            <ConnectMailboxForm onDone={() => setOpen(false)} />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

export function ConnectMailboxForm({ onDone }: { onDone?: () => void }) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [timezone, setTimezone] = useState("UTC");
  const [smtpHost, setSmtpHost] = useState("127.0.0.1");
  const [smtpPort, setSmtpPort] = useState("1025");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPending(true);
    const res = await fetch("/api/v1/mailboxes", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email,
        display_name: displayName,
        auth_kind: "smtp_imap",
        timezone,
        smtp_host: smtpHost,
        smtp_port: Number(smtpPort),
        smtp_tls: "none",
      }),
    });
    setPending(false);
    if (!res.ok) {
      const data = (await res.json().catch(() => null)) as
        | { error?: { message?: string } }
        | null;
      setError(data?.error?.message ?? "Could not connect mailbox");
      return;
    }
    setEmail("");
    setDisplayName("");
    onDone?.();
    router.refresh();
  }

  return (
    <form
      onSubmit={pending ? (e) => e.preventDefault() : onSubmit}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="mailbox-email">Email</Label>
        <Input
          id="mailbox-email"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="mailbox-name">Display name</Label>
        <Input
          id="mailbox-name"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="mailbox-tz">Timezone</Label>
        <Input
          id="mailbox-tz"
          required
          value={timezone}
          onChange={(e) => setTimezone(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="mailbox-smtp-host">SMTP host</Label>
        <Input
          id="mailbox-smtp-host"
          required
          value={smtpHost}
          onChange={(e) => setSmtpHost(e.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="mailbox-smtp-port">SMTP port</Label>
        <Input
          id="mailbox-smtp-port"
          type="number"
          required
          min={1}
          value={smtpPort}
          onChange={(e) => setSmtpPort(e.target.value)}
        />
      </div>
      {error ? <p className="font-body-md text-destructive">{error}</p> : null}
      <Button type="submit" pending={pending} className="rounded-md">
        Connect
      </Button>
    </form>
  );
}
