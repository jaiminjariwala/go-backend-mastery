import type { Metadata } from "next";
import "./globals.css";
export const metadata: Metadata = {
  title: "Sharded Raft KV",
  description:
    "Visual explanations of replication, agreement, recovery, and sharding.",
};
export default function Layout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
