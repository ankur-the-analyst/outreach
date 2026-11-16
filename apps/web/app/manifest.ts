import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Outreach",
    short_name: "Outreach",
    description: "One-to-one outreach",
    start_url: "/",
    display: "standalone",
    background_color: "#f4f4f6",
    theme_color: "#ffffff",
    icons: [
      {
        src: "/favicon.ico",
        sizes: "any",
        type: "image/x-icon",
      },
    ],
  };
}
