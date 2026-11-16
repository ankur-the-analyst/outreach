import type { ReactNode } from "react";
import { Suspense } from "react";
import { connection } from "next/server";
import { redirect } from "next/navigation";
import { AppShell } from "@/components/app-shell";
import { PageLoading } from "@/components/page-header";
import { getMe, getWorkspace } from "@/lib/api";

export const instant = false;

async function SignedIn({ children }: { children: ReactNode }) {
  await connection();
  const me = await getMe();
  if (!me) {
    redirect("/login");
  }
  const workspace = await getWorkspace();

  return (
    <AppShell name={workspace?.name ?? "Workspace"}>{children}</AppShell>
  );
}

export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <Suspense fallback={<PageLoading />}>
      <SignedIn>{children}</SignedIn>
    </Suspense>
  );
}
