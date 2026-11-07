"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";

export function LogoutButton() {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className="size-8 shrink-0 text-muted-foreground"
      title="Sign out"
      pending={pending}
      onClick={async () => {
        setPending(true);
        await fetch("/api/v1/auth/logout", {
          method: "POST",
          credentials: "include",
        });
        router.push("/login");
        router.refresh();
      }}
    >
      <LogOut className="size-[18px]" strokeWidth={1.5} />
      <span className="sr-only">Sign out</span>
    </Button>
  );
}
