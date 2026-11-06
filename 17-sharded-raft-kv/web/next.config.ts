import type { NextConfig } from "next";

// Concepts run on their own, without forwarding requests to the live Go lab.
const config: NextConfig = {
  agentRules: false,
  devIndicators: false,
  output: "standalone",
};

export default config;
