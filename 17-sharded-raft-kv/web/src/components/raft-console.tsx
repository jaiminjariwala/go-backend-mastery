"use client";

import { useEffect, useState } from "react";
import { KubernetesRecovery } from "./kubernetes-recovery";
import { LiveShardMap } from "./live-shard-map";
import { VisualConcepts } from "./visual-concepts";
import { NetworkCanvas } from "./network-canvas";
import { useOperationPlayback } from "@/lib/use-operation-playback";
import { useCluster } from "@/lib/use-cluster";

const secondary =
  "rounded-lg border border-gray-200 bg-white px-4 py-2.5 text-sm text-gray-700 transition-colors hover:bg-gray-50 disabled:opacity-40";
const primary =
  "rounded-lg border border-brand bg-brand px-4 py-2.5 text-sm text-white transition-colors hover:bg-brand-hover disabled:opacity-40";
function explain(text: string) {
  return text
    .replaceAll("Node ", "Server ")
    .replace(
      "Three nodes started. Waiting for an election timeout.",
      "No leader yet. Run the experiment: when a server’s waiting timer expires, it asks the others for votes.",
    )
    .replace(
      /starts election for term (\d+)\./,
      "starts election round $1 and asks for votes.",
    )
    .replace(
      /commits through entry (\d+) with a majority\./,
      "commits through entry $1 after enough replication replies arrive.",
    )
    .replace(
      "In-memory log retained for this simulated restart.",
      "Its simulated memory is retained. Messages sent to it will be dropped.",
    );
}

export function RaftConsole() {
  const [key, setKey] = useState("101"),
    [value, setValue] = useState("Maya");
  const {
    cluster,
    error,
    busy: databaseBusy,
    running,
    speed,
    setSpeed,
    setRunning,
    request,
    operate,
    receipt,
    pending,
    clientBusy,
    exchange,
    selectedShard,
    selectShard,
    recover,
  } = useCluster(key);
  const { playback, replay, skip } = useOperationPlayback(cluster, exchange);
  const [view, setView] = useState<"live" | "concepts">("live");
  const [practiceBusy, setPracticeBusy] = useState(false);
  const busy = databaseBusy || practiceBusy;
  const addPracticeRows = async () => {
    setPracticeBusy(true);
    try {
      for (const [index, name] of [
        "Maya",
        "Noah",
        "Ava",
        "Leo",
        "Ivy",
      ].entries()) {
        const result = await operate(
          "write",
          String(101 + index),
          name,
        );
        if (!result) break; // Preserve an uncertain operation for an explicit retry.
      }
      await request();
    } finally {
      setPracticeBusy(false);
    }
  };
  const [selected, setSelected] = useState<number | null>(null);
  const [showHeartbeats, setShowHeartbeats] = useState(false);
  const [target, setTarget] = useState("auto");
  useEffect(() => {
    setSelected(null);
    setTarget("auto");
  }, [selectedShard]);
  const sharded = (cluster?.ShardCount || 1) > 1;
  const route = cluster?.Route?.Key === key ? cluster.Route : null;
  const targetID = target === "auto" ? undefined : Number(target);
  const leaders = (cluster?.Nodes || [])
    .filter((n) => n.Up && n.Role === "leader")
    .sort((a, b) => b.Term - a.Term);
  const leader = leaders[0];
  const partitioned =
    (cluster?.PartitionTarget !== null &&
      cluster?.PartitionTarget !== undefined) ||
    cluster?.Isolated?.some(Boolean);
  const isolatedID = cluster?.Isolated?.findIndex(Boolean) ?? -1;
  const onlineCount = (cluster?.Nodes || []).filter((n) => n.Up).length;
  const connectedCount = (cluster?.Nodes || []).filter(
    (n) => n.Up && !cluster?.Isolated?.[n.ID],
  ).length;
  const progress = cluster?.Progress;
  const drop = (cluster?.Deliveries || [])
    .filter((d) => d.Status === "dropped" && cluster!.Tick - d.At <= 3)
    .at(-1);
  const node = selected !== null ? cluster?.Nodes[selected] : null;
  const latest = cluster?.Events.at(-1);

  return (
    <main className="mx-auto max-w-270 px-5 pb-10 pt-6 font-sans sm:px-8 sm:pt-8">
      <header>
        <h1 className="text-3xl leading-tight font-semibold tracking-heading sm:text-ui-34">
          Sharded Raft KV
        </h1>
        <p className="mt-2 text-sm leading-relaxed text-gray-500">
          A small database that keeps working when some servers fail.
        </p>
      </header>
      <nav className="mt-4 flex gap-2" aria-label="Project views">
        <button
          className={view === "live" ? primary : secondary}
          onClick={() => setView("live")}
          aria-pressed={view === "live"}
        >
          Live cluster
        </button>
        <button
          className={view === "concepts" ? primary : secondary}
          onClick={() => setView("concepts")}
          aria-pressed={view === "concepts"}
        >
          Concepts
        </button>
      </nav>
      {view === "concepts" && (
        <VisualConcepts
          onOpenLab={() => setView("live")}
        />
      )}
      {view === "live" && (
        <>
          {sharded && (
            <LiveShardMap
              shards={cluster?.Shards || []}
              selected={selectedShard}
              onSelect={selectShard}
            />
          )}

          <h2 className="mt-5 text-base font-semibold">Client request</h2>
          <p className="mt-1 text-xs text-gray-500">
            Save changes one key. Tables show all rows already stored in the selected shard.
          </p>
          <div className="sticky top-0 z-30 mt-2 flex flex-wrap items-end justify-between gap-x-6 gap-y-2 border-b border-gray-100 bg-white/95 py-3 backdrop-blur-sm">
            <form
              id="row-editor"
              className="flex max-w-full flex-wrap items-end gap-3"
              onSubmit={(e) => {
                e.preventDefault();
                if (cluster?.Live)
                  void operate("write", key, value, false, targetID);
                else void request({ Action: "write", Key: key, Value: value });
              }}
            >
              <label className="text-ui-13 text-gray-500">
                Key
                <input
                  className="mt-1.5 block w-25 rounded-lg border border-gray-200 px-3 py-2.5 text-ui-15 text-gray-900 sm:w-30"
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  maxLength={64}
                  required
                  autoComplete="off"
                />
              </label>
              <label className="text-ui-13 text-gray-500">
                Value
                <input
                  className="mt-1.5 block w-31.25 rounded-lg border border-gray-200 px-3 py-2.5 text-ui-15 text-gray-900 sm:w-42.5"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  maxLength={256}
                  autoComplete="off"
                />
              </label>
              <button className={primary} disabled={busy || !cluster}>
                Save row
              </button>
              {cluster?.Live && (
                <button
                  type="button"
                  className={secondary}
                  disabled={busy || !key}
                  onClick={() => void operate("read", key, "", false, targetID)}
                >
                  Read row
                </button>
              )}
              {cluster?.Live && (
                <button
                  type="button"
                  className={secondary}
                  disabled={busy || !key}
                  onClick={() =>
                    void operate("delete", key, "", false, targetID)
                  }
                >
                  Delete key
                </button>
              )}
            </form>
            <div className="flex flex-wrap items-center gap-2">
              <button
                className={primary}
                disabled={!cluster}
                onClick={() =>
                  cluster?.Live ? void request() : setRunning(!running)
                }
              >
                {cluster?.Live
                  ? "Refresh state"
                  : running
                    ? "Pause"
                    : "Run experiment ↗"}
              </button>
              {!cluster?.Live && (
                <>
                  <button
                    className={secondary}
                    disabled={busy || !cluster || cluster.Live}
                    title={
                      cluster?.Live
                        ? "Live servers run on their own clocks."
                        : undefined
                    }
                    onClick={() => {
                      setRunning(false);
                      void request({ Action: "step" });
                    }}
                  >
                    Step
                  </button>
                  <button
                    className={secondary}
                    disabled={busy || !cluster || cluster.Durable}
                    title={
                      cluster?.Durable
                        ? "Use a new RAFT_DATA_DIR for a fresh experiment; saved data is protected."
                        : undefined
                    }
                    onClick={() => {
                      setRunning(false);
                      setSelected(null);
                      void request({ Action: "reset" });
                    }}
                  >
                    Reset
                  </button>
                </>
              )}
            </div>
          </div>
          {sharded && (
            <div className="mt-3 flex flex-wrap items-center gap-3 text-ui-13 text-gray-500">
              <span>
                {route ? (
                  <>
                    <code className="whitespace-nowrap">
                      {key || "(empty key)"}
                    </code>{" "}
                    → Shard {route.ShardID + 1} · stable key hash modulo 2
                  </>
                ) : (
                  "Finding the key’s shard…"
                )}
              </span>
              {route && route.ShardID !== selectedShard && (
                <button
                  type="button"
                  className="text-brand underline underline-offset-4"
                  onClick={() => selectShard(route.ShardID)}
                >
                  View owning shard
                </button>
              )}
            </div>
          )}
          {error && (
            <div
              role="alert"
              className="mt-4 flex flex-wrap items-center gap-3 text-sm text-red-700"
            >
              <span>{error}</span>
              <button
                className="underline underline-offset-4"
                onClick={() => void request()}
              >
                Refresh state
              </button>
            </div>
          )}
          {clientBusy && (
            <p role="status" className="mt-4 text-ui-13 text-gray-500">
              Waiting for a committed response. Server state continues to
              refresh below.
            </p>
          )}
          {cluster?.Live && pending && (
            <div className="mt-4 flex flex-wrap items-center gap-3 text-ui-13 text-gray-500">
              <span>
                Earlier {pending.Operation === "delete" ? "removal" : "save"} is
                still unconfirmed:{" "}
                <span className="font-mono whitespace-nowrap">
                  {pending.Key} = {pending.Value}
                </span>
              </span>
              <button
                className="text-brand underline underline-offset-4 disabled:opacity-40"
                disabled={busy}
                onClick={() =>
                  void operate(
                    pending.Operation || "write",
                    pending.Key,
                    pending.Value,
                    true,
                    targetID,
                  )
                }
              >
                Retry the earlier request
              </button>
              <span className="w-full overflow-x-auto font-mono text-ui-11 text-gray-400">
                Request ID {pending.RequestID}. Keep this ID when retrying.
                Restore a majority first; then retry before saving different
                data.
              </span>
            </div>
          )}
          {cluster?.Live && receipt && (
            <div
              role="status"
              className="mt-3 rounded-lg border border-gray-200 px-4 py-2 text-sm"
            >
              <p className="font-medium text-brand">
                {receipt.Operation === "delete"
                  ? "Row removed · client received a committed response"
                  : receipt.Operation === "write"
                    ? "Saved · client received a committed response"
                    : receipt.Found
                      ? "Read completed after a committed barrier"
                      : "Read completed · key not found"}
              </p>
              <p className="mt-1 overflow-x-auto whitespace-nowrap font-mono text-gray-700">
                {receipt.Key}{" "}
                {receipt.Found ? `= ${receipt.Value}` : "is absent"}
              </p>
              <details className="mt-1 text-xs text-gray-500">
                <summary className="cursor-pointer">Request details</summary>
              <p className="mt-1 text-xs text-gray-500">
                {sharded && receipt.ShardID !== undefined
                  ? `Shard ${receipt.ShardID + 1} · `
                  : ""}
                Entry {receipt.Index} · term {receipt.Term}
                {receipt.Replayed
                  ? " · same request ID, original result returned"
                  : ""}
              </p>
              {receipt.RequestID && (
                <p className="mt-1 overflow-x-auto whitespace-nowrap font-mono text-ui-11 text-gray-400">
                  Request ID {receipt.RequestID}
                </p>
              )}
              </details>
            </div>
          )}

          <section className="mt-4" aria-label="Server cluster">
            {cluster?.Live && (!leader || connectedCount < 2) && (
              <div
                className="mb-5 rounded-xl border border-amber-200 bg-amber-50/30 p-4 text-sm"
                role="status"
              >
                <p className="font-medium">
                  {connectedCount < 2
                    ? "This group cannot form a majority."
                    : "An election is in progress."}
                </p>
                <p className="mt-2 leading-relaxed text-gray-600">
                  {onlineCount}/3 replicas are running and {connectedCount}/3
                  are not isolated. Two communicating replicas are needed.
                  Repeated election attempts increase the term; they do not mean
                  a write succeeded. Heal network does not start stopped
                  processes.
                </p>
                <button
                  className={`${secondary} mt-3`}
                  disabled={busy || cluster?.Deployment === "kubernetes"}
                  onClick={() => void recover()}
                >
                  Restore this group’s servers and links
                </button>
                <p className="mt-2 text-xs text-gray-500">
                  Recovery keeps rows, logs, snapshots, and retry IDs.
                </p>
              </div>
            )}

            {cluster?.Live && (
              <details className="mb-3 rounded-xl border border-gray-200 px-4 py-3 text-ui-13">
                <summary className="cursor-pointer font-medium text-gray-700">
                  Network experiments{partitioned ? " · isolation active" : ""}
                  {target !== "auto" ? ` · requests target Server ${Number(target) + 1}` : ""}
                </summary>
                <div className="space-y-3 pt-3">
                  <p className="text-xs text-gray-500">Controls affect Shard {selectedShard + 1}. Isolation blocks connections; it does not stop the process.</p>
                  <div className="flex flex-wrap gap-2">
                    <button className={secondary} disabled={busy || !leader || !!partitioned}
                      onClick={() => leader && void request({ Action: "partition", ID: leader.ID })}>Isolate leader</button>
                    <button className={secondary} disabled={!partitioned}
                      onClick={() => void request({ Action: "heal" })}>Heal network</button>
                  </div>
                  {partitioned && <p className="text-xs text-warning">
                    {isolatedID >= 0 ? `Server ${isolatedID + 1} is isolated. The other two can elect a leader. The isolated server cannot commit alone.` : "Isolation is configured for a stopped or restarting server."}
                  </p>}
                  <label className="flex flex-wrap items-center gap-2 text-gray-500">
                    Target server within the key’s shard
                    <select aria-label="Client request destination" className="rounded-md border border-gray-200 bg-white px-2 py-1.5 text-gray-700"
                      value={target} onChange={(e) => setTarget(e.target.value)}>
                      <option value="auto">Automatic (find the leader)</option>
                      {cluster.Nodes.map((n) => <option key={n.ID} value={n.ID}>Server {n.ID + 1}{cluster.Isolated?.[n.ID] ? " · isolated" : ""}</option>)}
                    </select>
                  </label>
                  <details className="text-xs text-gray-500">
                    <summary className="cursor-pointer">Sample data</summary>
                    <button className={`${secondary} mt-2`} disabled={busy || !!pending} onClick={() => void addPracticeRows()}>
                      {practiceBusy ? "Saving practice users…" : "Add five practice users"}
                    </button>
                    <p className="mt-2">Saves keys 101 to 105 through the real API. Existing values for these keys are replaced.</p>
                  </details>
                </div>
              </details>
            )}
            <div className="flex flex-wrap items-center justify-between gap-3 text-ui-13 text-gray-500">
              <p>
                {leaders.length > 1
                  ? `${leaders.length} servers report leadership · Server ${leader.ID + 1} has the highest term`
                  : leader
                    ? `Server ${leader.ID + 1} is the leader`
                    : "Waiting for a leader"}
                <span className="mx-2 text-gray-300">·</span>
                <span className="tabular-nums">
                  {cluster?.Live
                    ? "Live · refreshes automatically"
                    : `Tick ${cluster?.Tick || 0}`}
                </span>
              </p>
              {!cluster?.Live && (
                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 text-xs">
                    <input
                      type="checkbox"
                      className="accent-brand"
                      checked={showHeartbeats}
                      disabled={cluster?.Live}
                      onChange={(e) => setShowHeartbeats(e.target.checked)}
                    />
                    Show heartbeats
                  </label>
                  <select
                    aria-label="Playback speed"
                    className="rounded-md bg-white py-1 text-xs text-gray-500"
                    value={speed}
                    disabled={cluster?.Live}
                    onChange={(e) => setSpeed(Number(e.target.value))}
                  >
                    <option value={1000}>0.5× speed</option>
                    <option value={600}>1× speed</option>
                    <option value={300}>2× speed</option>
                  </select>
                </div>
              )}
            </div>
            <KubernetesRecovery cluster={cluster} />
          <NetworkCanvas
              cluster={cluster}
              exchange={exchange}
              playback={playback}
              onReplay={replay}
              onSkipPlayback={skip}
              rowBusy={busy}
              onEdit={(storedKey, storedValue) => {
                setKey(storedKey);
                setValue(storedValue);
                document
                  .getElementById("row-editor")
                  ?.scrollIntoView({ behavior: "smooth", block: "center" });
              }}
              busy={cluster?.Live ? false : busy}
              speed={speed}
              showHeartbeats={showHeartbeats}
              onToggle={(id) => void request({ Action: "toggle", ID: id })}
              onSelect={(id) => {
                setSelected(id);
                const details = document.getElementById("server-details") as HTMLDetailsElement | null;
                if (details) {
                  details.open = true;
                  details.scrollIntoView({ behavior: "smooth", block: "start" });
                }
              }}
            />
            {cluster?.Live ? (
              <p className="text-center text-xs text-gray-400">
                Three copies of Shard {selectedShard + 1}. Saved rows stay here after a restart.
              </p>
            ) : (
              <div className="flex justify-center gap-5 text-ui-11 text-gray-400 sm:text-xs">
                <span>
                  <span className="mr-1.5 inline-block h-0.5 w-3 align-middle bg-brand" />
                  Request
                </span>
                <span>
                  <span className="mr-1.5 inline-block h-0.5 w-3 align-middle bg-reply" />
                  Reply
                </span>
                <span>
                  <span className="mr-1.5 inline-block h-0.5 w-3 align-middle bg-dropped" />
                  Dropped
                </span>
              </div>
            )}
            <div
              className="mt-5 rounded-xl border border-gray-200 px-5 py-4 text-sm leading-relaxed sm:text-ui-15"
              aria-live="polite"
            >
              <p className="text-gray-700">
                {latest ? explain(latest.Text) : "Connecting to the cluster."}
              </p>
              {drop && (
                <p className="mt-1.5 text-ui-13 text-warning-muted">
                  Server {drop.Message.From + 1} → Server {drop.Message.To + 1}:
                  destination offline. Message dropped.
                </p>
              )}
            </div>
            {progress && (
              <div
                className="mt-3 flex flex-wrap items-baseline gap-x-3 gap-y-1 text-ui-13 text-gray-500"
                role="status"
              >
                <span className="max-w-full overflow-x-auto whitespace-nowrap font-mono text-gray-800">
                  {progress.Delete
                    ? `Remove ${progress.Key}`
                    : `${progress.Key} = ${progress.Value}`}
                </span>
                <span className={progress.Committed ? "text-brand" : ""}>
                  {progress.Phase === "superseded"
                    ? "Replaced before commitment"
                    : progress.Committed
                      ? "Committed"
                      : "Pending ,  waiting for a majority"}
                </span>
                <span>
                  {progress.Confirmed.length}/3 recorded copies
                  {progress.Committed
                    ? ` · observed applied on ${progress.Applied.length}/3`
                    : ""}
                </span>
              </div>
            )}
          </section>

          <details id="server-details" className="mt-6 scroll-mt-28 rounded-xl border border-gray-200 p-4">
            <summary className="cursor-pointer text-sm font-medium text-gray-700">Server details and logs{sharded ? ` · Shard ${selectedShard + 1}` : ""}</summary>
            <section className="mt-4" aria-labelledby="state-title">
            <h2
              id="state-title"
              className="text-ui-21 font-semibold tracking-heading-small"
            >
              Server state{sharded ? ` · Shard ${selectedShard + 1}` : ""}
            </h2>
            <div className="mt-3 overflow-x-auto">
              <table className="w-full min-w-162.5 border-collapse text-left text-sm">
                <thead className="border-b border-gray-200 text-ui-13 text-gray-500">
                  <tr>
                    {[
                      "Server",
                      "Role",
                      "Term",
                      "Snapshot through",
                      "Retained log",
                      "Committed",
                      "Applied values",
                      "",
                    ].map((h, i) => (
                      <th
                        key={i}
                        className="px-3 py-3 font-medium first:pl-0 last:pr-0"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {cluster?.Nodes.map((n) => (
                    <tr key={n.ID} className="border-b border-gray-100">
                      <td className="py-4 pr-3 font-medium">
                        Server {n.ID + 1}
                        {cluster.Live && (
                          <span className="mt-1 block text-ui-11 font-normal text-gray-400">
                            {cluster.PIDs?.[n.ID]
                              ? `PID ${cluster.PIDs[n.ID]}`
                              : "Process stopped"}
                          </span>
                        )}
                      </td>
                      <td
                        className={`px-3 py-4 capitalize ${n.Role === "leader" ? "text-brand" : "text-gray-500"}`}
                      >
                        {n.Role}
                      </td>
                      <td className="px-3 py-4 tabular-nums text-gray-600">
                        {cluster.Live && !n.Up ? "n/a" : n.Term}
                      </td>
                      <td className="px-3 py-4 tabular-nums text-gray-600">
                        {cluster.Live && !n.Up ? "n/a" : n.Snapshot?.Index || 0}
                      </td>
                      <td className="px-3 py-4 tabular-nums text-gray-600">
                        {cluster.Live && !n.Up ? "n/a" : n.Log?.length || 0}
                      </td>
                      <td className="px-3 py-4 tabular-nums text-gray-600">
                        {cluster.Live && !n.Up ? "n/a" : n.Commit}
                      </td>
                      <td className="max-w-62.5 px-3 py-4 font-mono text-ui-13 text-gray-700">
                        {Object.entries(n.KV).length
                          ? Object.entries(n.KV).map(
                              ([storedKey, storedValue], i, entries) => (
                                <span
                                  key={storedKey}
                                  className="inline-block whitespace-nowrap"
                                >
                                  {storedKey} = {storedValue}
                                  {i < entries.length - 1 ? ",\u00a0" : ""}
                                </span>
                              ),
                            )
                          : "No rows"}
                      </td>
                      <td className="py-4 pl-3 text-right">
                        <button
                          className="whitespace-nowrap text-ui-13 text-brand underline-offset-4 hover:underline"
                          aria-expanded={selected === n.ID}
                          disabled={cluster.Live && !n.Up}
                          title={
                            cluster.Live && !n.Up
                              ? "The stopped process cannot report its log. Restart it to inspect recovered state."
                              : undefined
                          }
                          onClick={() =>
                            setSelected(selected === n.ID ? null : n.ID)
                          }
                        >
                          Inspect log
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {node && !(cluster?.Live && !node.Up) && (
              <div className="mt-5 rounded-xl border border-gray-200 p-5">
                <div className="flex items-center justify-between gap-4">
                  <h3 className="font-medium">Server {node.ID + 1} log</h3>
                  <button
                    onClick={() => setSelected(null)}
                    className="rounded px-2 text-xl text-gray-400 hover:text-gray-800"
                    aria-label="Close log"
                  >
                    ×
                  </button>
                </div>
                <div className="mt-3 flex flex-wrap items-center justify-between gap-3 text-ui-13 text-gray-500">
                  <p>
                    {node.Snapshot?.Index
                      ? `Snapshot covers entries 1-${node.Snapshot.Index}.`
                      : "No snapshot yet."}{" "}
                    Retained log begins at entry{" "}
                    {(node.Snapshot?.Index || 0) + 1}.
                  </p>
                  {cluster?.Live && (
                    <button
                      className={secondary}
                      disabled={
                        busy || node.Commit <= (node.Snapshot?.Index || 0)
                      }
                      onClick={() =>
                        void request({ Action: "snapshot", ID: node.ID })
                      }
                    >
                      Snapshot committed log
                    </button>
                  )}
                  <p className="w-full text-xs">
                    A snapshot keeps the applied values and retry receipts; it
                    discards only the committed log prefix. Pending entries
                    remain.
                  </p>
                  {!!node.Snapshot?.Index && (
                    <p className="w-full text-xs">
                      Saved values: {Object.keys(node.Snapshot.KV || {}).length}{" "}
                      · Saved retry receipts:{" "}
                      {Object.keys(node.Snapshot.Requests || {}).length} ·
                      Boundary term: {node.Snapshot.Term}
                    </p>
                  )}
                  {cluster?.Live && (
                    <p className="w-full text-xs">
                      Current WAL size:{" "}
                      {(cluster.WALBytes?.[node.ID] || 0).toLocaleString()}{" "}
                      bytes. Snapshotting rewrites it with one checkpoint and
                      the remaining suffix.
                    </p>
                  )}
                </div>
                <div className="mt-3 overflow-x-auto">
                  <table className="w-full min-w-87.5 text-left text-ui-13">
                    <thead className="border-b border-gray-200 text-gray-500">
                      <tr>
                        {["Entry", "Term", "Command", "State"].map((h) => (
                          <th className="py-2 pr-3 font-normal" key={h}>
                            {h}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {!node.Log?.length && (
                        <tr>
                          <td colSpan={4} className="py-4 text-gray-500">
                            {node.Snapshot?.Index
                              ? `No retained entries. The saved values are in the snapshot; the next write will use entry ${node.Snapshot.Index + 1}.`
                              : "No entries yet. Wait for a leader, then write a value."}
                          </td>
                        </tr>
                      )}
                      {(node.Log || []).map((e, i) => (
                        <tr key={i} className="border-b border-gray-100">
                          <td className="py-3 pr-3">
                            {(node.Snapshot?.Index || 0) + i + 1}
                          </td>
                          <td className="py-3 pr-3">{e.Term}</td>
                          <td className="max-w-62.5 py-3 pr-3 font-mono whitespace-nowrap">
                            {e.ReadKey
                              ? `Read barrier: ${e.ReadKey}`
                              : e.Delete
                                ? `Remove row: ${e.Key}`
                                : e.Key
                                  ? `${e.Key} = ${e.Value}`
                                  : "Leader no-op"}
                            {e.RequestID && (
                              <span className="mt-1 block text-ui-10 text-gray-400">
                                {e.RequestID}
                              </span>
                            )}
                          </td>
                          <td className="py-3 text-gray-500">
                            {(node.Snapshot?.Index || 0) + i < node.Commit
                              ? "Committed"
                              : "Pending"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
            </section>
          </details>

          <details className="mt-7 border-t border-gray-100 pt-5">
            <summary className="w-fit text-sm text-gray-500">
              Activity history
            </summary>
            <ol className="mt-3 max-h-64 space-y-3 overflow-auto text-ui-13 leading-relaxed text-gray-500">
              {cluster?.Events.slice()
                .reverse()
                .map((e, i) => (
                  <li className="flex gap-4" key={`${e.At}-${i}-${e.Text}`}>
                    <time className="w-8 shrink-0 tabular-nums text-gray-400">
                      {e.At}
                    </time>
                    <span>{explain(e.Text)}</span>
                  </li>
                ))}
            </ol>
          </details>
          {cluster?.Deployment === "kubernetes" && (
            <p className="mt-5 text-sm text-gray-600">
              Kubernetes deployment: six Pods across two fixed Raft groups.
              Kubernetes replaces Pods and reattaches their persistent volumes.
              Use kubectl for Pod replacement; changing the StatefulSet replica
              count does not change the voting group.
            </p>
          )}
          {cluster?.Transport === "grpc" && (
            <p className="mt-5 text-xs text-gray-500">
              This group’s received gRPC traffic since each process started:{" "}
              {(cluster.PeerRPCs || []).reduce((sum, n) => sum + n, 0)} peer
              calls ·{" "}
              {(cluster.ClientRPCs || []).reduce((sum, n) => sum + n, 0)} client
              commands. A received RPC does not itself prove commitment.
            </p>
          )}
          <footer className="mt-6 text-xs leading-relaxed text-gray-400">
            {sharded
              ? "Two fixed shards run as six independent Go processes. Peer and client-command traffic uses protobuf over gRPC. Each group owns its elections, WALs, snapshots, and retry receipts. Table values are observations; use Read row for a verified read."
              : cluster?.Live
                ? "Three independent Go processes exchange protobuf RPCs over localhost gRPC; browser and administration calls use HTTP. Stop kills a process; Restart recovers its own WAL. State is polled separately from each server, so this table is not an atomic cluster snapshot. No sharding yet."
                : cluster?.Durable
                  ? "Disk-backed simulation. Three logical servers share one process. Stop discards memory; Restart reloads the WAL. Reset is disabled to protect data."
                  : "In-memory protocol fixture."}
          </footer>
        </>
      )}
    </main>
  );
}
