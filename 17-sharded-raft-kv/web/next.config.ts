import type { NextConfig } from "next";

// Concepts run on their own, without forwarding requests to the live Go lab.
const config: NextConfig = {
  agentRules: false,
  devIndicators: false,
  output: "standalone",
  async rewrites() {
    // Proxy browser API calls to the configured Go controller.
    return process.env.NODE_ENV === "development" || process.env.RAFT_API_ORIGIN
      ? [{ source: "/api/:path*", destination: `${process.env.RAFT_API_ORIGIN || "http://127.0.0.1:8084"}/api/:path*` }]
      : [];
  },
};

export default config;
