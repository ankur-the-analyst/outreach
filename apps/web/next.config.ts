import type { NextConfig } from "next";

const apiURL = process.env.API_URL ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  /* config options here */
  cacheComponents: true,
  partialPrefetching: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiURL}/api/:path*` }];
  },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
