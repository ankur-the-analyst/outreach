"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";

export function LogoutButton() {
  const router = useRouter();
  return (
    <Button
      type="button"
      variant="ghost"
      className="w-full justify-start rounded-md text-muted-foreground"
      onClick={async () => {
        await fetch("/api/v1/auth/logout", { method: "POST", credentials: "include" });
        router.push("/login");
        router.refresh();
      }}
    >
      Sign out
    </Button>
  );
}
