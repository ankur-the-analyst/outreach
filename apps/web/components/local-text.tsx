"use client";

import { useEffect, useState } from "react";

export function LocalText({
  text,
  className,
  as: Tag = "span",
}: {
  text: string;
  className?: string;
  as?: "span" | "p";
}) {
  const [shown, setShown] = useState("");
  useEffect(() => {
    setShown(text);
  }, [text]);
  if (!shown) {
    return null;
  }
  return <Tag className={className}>{shown}</Tag>;
}
