import { VisualConcepts } from "@/components/visual-concepts";

// The published page contains concept illustrations only. The live database
// console remains local development work until its interface is ready.
export default function Page() {
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
