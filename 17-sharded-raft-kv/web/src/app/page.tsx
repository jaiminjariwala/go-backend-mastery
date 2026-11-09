import { VisualConcepts } from "@/components/visual-concepts";
import { RaftConsole } from "@/components/raft-console";

// Local development and configured deployments connect to the Go backend.
// Standalone concept previews can still build without a running database.
export default function Page() {
  if (process.env.NODE_ENV === "development" || process.env.RAFT_API_ORIGIN)
    return <RaftConsole />;
  return (
    <main className="mx-auto max-w-270 px-5 pb-10 pt-9 font-sans sm:px-8 sm:pt-12">
      <header>
        <h1 className="text-ui-34 leading-tight font-semibold tracking-heading sm:text-ui-44">
          Sharded Raft KV
        </h1>
        <p className="mt-3 text-base leading-relaxed text-gray-500">
          A small database that keeps working when some servers fail.
        </p>
      </header>
      <VisualConcepts />
    </main>
  );
}
