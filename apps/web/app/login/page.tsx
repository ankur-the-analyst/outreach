"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { LoginForm } from "@/components/login-form";

export default function LoginPage() {
  const router = useRouter();
  useEffect(() => {
    fetch("/api/v1/me", { credentials: "include" }).then((res) => {
      if (res.ok) {
        router.replace("/");
      }
    });
  }, [router]);

  return (
    <main className="flex min-h-full flex-col items-center justify-center p-[var(--margin-desktop)]">
      <div className="w-full max-w-sm rounded-md border border-border bg-card p-6">
        <h1 className="font-headline-xl text-foreground">Outreach</h1>
        <p className="mt-1 mb-6 font-body-md text-muted-foreground">Sign in to your workspace.</p>
        <LoginForm />
      </div>
    </main>
  );
}
