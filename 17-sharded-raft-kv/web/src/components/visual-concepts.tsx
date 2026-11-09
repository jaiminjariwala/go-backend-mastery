"use client";

import { useState } from "react";
import { ReplicaArtwork } from "./replica-artwork";

const topics = [
  "Rows",
  "Replication",
  "Agreement",
  "Elections",
  "Network split",
  "Snapshots",
  "Sharding",
  "Reads & retries",
  "gRPC",
  "Kubernetes",
];
const names = ["Maya", "Noah", "Ava", "Leo", "Ivy"];
const colors = ["#9a83e7", "#5078ed", "#9a83e7", "#56b77a", "#1caaa8"];
type Row = { id: string; name: string; color: string };
const original: Row[] = names.map((name, i) => ({
  id: String(101 + i),
  name,
  color: colors[i],
}));
const updated = original.map((row, i) =>
  i === 0 ? { ...row, name: "Maya Chen" } : row,
);
const button =
  "rounded-xl bg-brand px-5 py-3 text-sm font-medium text-white disabled:opacity-40";

// These scenes are illustrations, not fabricated observations of live servers.
// Each action changes only its scene; no database request is sent.
export function VisualConcepts({ onOpenLab }: { onOpenLab?: () => void }) {
  const [topic, setTopic] = useState(0);
  const [step, setStep] = useState(0);
  const [selected, setSelected] = useState("101");
  const changeTopic = (index: number) => {
    setTopic(index);
    setStep(0);
  };
  const next = (maximum: number) => (
    <button
      className={button}
      onClick={() => setStep((n) => (n + 1) % (maximum + 1))}
    >
      {step === maximum ? "Replay" : "Next step →"}
    </button>
  );
  let diagram: React.ReactNode;
  let caption = "";
  let detail = "";
  let controls: React.ReactNode = null;

  if (topic === 0) {
    diagram = (
      <div className="mx-auto max-w-lg">
        <Table rows={original} highlight="101" />
        <div className="mt-5 flex justify-center gap-3">
          <Tag>Key: 101</Tag>
          <Tag>Value: Maya</Tag>
        </div>
      </div>
    );
    caption = "A key is a name tag. A value is what we save.";
  } else if (topic === 1) {
    diagram = (
      <Columns>
        {[0, 1, 2].map((id) => (
          <Server
            key={id}
            id={id}
            rows={step ? updated : original}
            highlight={step ? "101" : undefined}
          />
        ))}
      </Columns>
    );
    caption = "Three copies. Same rows.";
    detail =
      "These three servers form one group: a team storing copies of the same data.";
    controls = (
      <button className={button} onClick={() => setStep((n) => (n ? 0 : 1))}>
        {step ? "Restore Maya" : "Change Maya to Maya Chen"}
      </button>
    );
  } else if (topic === 2) {
    diagram = (
      <>
        <div className="mb-7 flex flex-wrap items-center justify-center gap-4">
          <Tag>Save 101 Maya Chen</Tag>
          <span className="text-3xl text-brand">→</span>
          <Tag>
            {step === 0
              ? "Waiting"
              : step === 1
                ? "1 of 3 recorded"
                : "2 of 3 recorded"}
          </Tag>
        </div>
        <Columns>
          {[0, 1, 2].map((id) => (
            <Server
              key={id}
              id={id}
              role={id === 0 ? "Leader" : "Follower"}
              rows={step >= 2 && (id < 2 || step === 3) ? updated : original}
              highlight={step >= 2 ? "101" : undefined}
              note={
                (step >= 1 && id === 0) || (step >= 2 && id === 1) || step === 3
                  ? "✓ Instruction recorded on disk"
                  : "Waiting for instruction"
              }
            />
          ))}
        </Columns>
      </>
    );
    caption =
      step < 2
        ? "The leader records it. A follower records a copy."
        : "Saved means two of three agreed.";
    detail =
      step === 3
        ? "The third copy has caught up too."
        : "The leader replies after agreement and after applying the change to its rows.";
    controls = next(3);
  } else if (topic === 3 || topic === 4) {
    const split = topic === 4;
    const separated = split && step > 0 && step < 3;
    diagram = (
      <div className="relative">
        {separated && (
          <>
            <svg
              role="img"
              aria-label="Network split separates Server 1 from Servers 2 and 3"
              viewBox="0 0 8 520"
              preserveAspectRatio="none"
              className="pointer-events-none absolute -top-2 left-1/3 hidden h-[calc(100%+8px)] w-2 -translate-x-3 text-red-500 md:block"
            >
              <path
                d="M4 4V516"
                fill="none"
                stroke="currentColor"
                strokeWidth="6"
                strokeDasharray="24 16"
                strokeLinecap="round"
              />
            </svg>
            <div className="pointer-events-none absolute left-[calc(66.6667%+8px)] top-16 z-10 hidden -translate-x-1/2 text-center text-sm font-medium leading-tight text-brand md:block">
              <span>
                Still
                <br />
                together
              </span>
              <svg
                viewBox="0 0 176 18"
                className="mt-1 h-5 w-32 lg:w-44"
                aria-hidden="true"
              >
                <path d="M7 9H169" stroke="currentColor" strokeWidth="4" />
                <circle cx="7" cy="9" r="5" fill="currentColor" />
                <circle cx="169" cy="9" r="5" fill="currentColor" />
              </svg>
            </div>
            <p className="mb-4 text-center text-sm font-medium text-brand md:hidden">
              Server 1 is alone. Servers 2 and 3 are still together.
            </p>
          </>
        )}
        <div
          className={`grid gap-6 md:grid-cols-3 ${split ? "md:gap-x-12" : ""}`}
        >
          {[0, 1, 2].map((id) => (
            <div key={id}>
              <Server
                id={id}
                offline={!split && step > 0 && step < 3 && id === 0}
                rows={
                  step === 3 || (step === 2 && id !== 0) ? updated : original
                }
                role={
                  step === 0
                    ? id === 0
                      ? "Leader"
                      : "Follower"
                    : id === 0
                      ? step === 3
                        ? "Follower · caught up"
                        : split
                          ? "Alone · old leader"
                          : "Stopped"
                      : id === 1
                        ? "New leader"
                        : "Follower"
                }
              />
            </div>
          ))}
        </div>
      </div>
    );
    caption =
      step === 3
        ? "Three connected copies. Same saved rows."
        : step === 0
          ? "Three servers. One leader."
          : split
            ? "Running does not mean connected."
            : "Leader stopped. The others pick a new one.";
    detail =
      split && step === 2
        ? "The connected pair saved the new value. The isolated copy still has Maya."
        : "Two connected servers can save a change. One server alone cannot.";
    controls = (
      <>
        <button
          className={button}
          onClick={() => setStep((n) => (n === 0 ? 1 : n === 3 ? 0 : 3))}
        >
          {step === 3
            ? "Reset example"
            : step
              ? split
                ? "Heal network"
                : "Restart server"
              : split
                ? "Isolate leader"
                : "Stop leader"}
        </button>
        {step > 0 && step < 3 && (
          <button className={button} onClick={() => setStep(2)}>
            Save Maya Chen on the pair
          </button>
        )}
      </>
    );
  } else if (topic === 5) {
    diagram = (
      <div className="grid items-center gap-8 md:grid-cols-[1fr_auto_1fr]">
        <LogPages compacted={step > 0} />
        <Arrow />
        <div>
          <Table rows={updated} highlight="101" />
          <p className="mt-4 text-center text-lg text-brand">
            {step
              ? "Snapshot through 12 · entries 13 and 14 retained"
              : "Current rows"}
          </p>
        </div>
      </div>
    );
    caption = "Keep the result. Remove the covered log pages.";
    detail =
      "The snapshot also keeps completed request IDs. Newer log entries stay.";
    controls = (
      <button className={button} onClick={() => setStep((n) => (n ? 0 : 1))}>
        {step ? "Show full log" : "Take snapshot through 12"}
      </button>
    );
  } else if (topic === 6) {
    const owner = shard(selected);
    diagram = (
      <div className="grid items-center gap-8 lg:grid-cols-[1fr_auto_1fr]">
        <Table rows={original} highlight={selected} onSelect={setSelected} />
        <Arrow />
        <div className="space-y-5">
          {[0, 1].map((group) => (
            <div
              key={group}
              className={`rounded-3xl border-4 bg-white p-5 ${owner === group ? "border-brand" : "border-gray-200"}`}
            >
              <h3
                className={`mb-4 text-center text-2xl font-semibold ${group === 0 ? "text-brand" : "text-emerald-600"}`}
              >
                Shard {group + 1}
              </h3>
              <Table
                rows={original.filter((row) => shard(row.id) === group)}
                highlight={selected}
                compact
              />
              <div className="mt-4 flex justify-center gap-3">
                {[0, 1, 2].map((id) => (
                  <div key={id} className="h-16 w-16">
                    <ReplicaArtwork compact />
                  </div>
                ))}
              </div>
              <p className="mt-2 text-center text-sm text-gray-500">
                Group {group + 1} · three copies of these rows
              </p>
            </div>
          ))}
        </div>
      </div>
    );
    caption = "Each key has one home group.";
    detail = `Selected key ${selected}: FNV-1a hash % 2 = ${owner}. It belongs to shard ${owner + 1}. Click a row to follow its route.`;
  } else if (topic === 7) {
    if (step === 0)
      diagram = (
        <Columns>
          {[0, 1, 2].map((id) => (
            <Server
              key={id}
              id={id}
              rows={id === 2 ? original : updated}
              highlight="101"
              staleKey={id === 2 ? "101" : undefined}
              role={id === 0 ? "Leader" : "Follower"}
              note={id === 2 ? "Behind the other copies" : "Current: Maya Chen"}
            />
          ))}
        </Columns>
      );
    else
      diagram = (
        <div className="grid items-center gap-8 md:grid-cols-[1fr_auto_1fr]">
          <Panel title="What the client sees">
            <img
              src="/client-computer.png"
              alt="Client"
              className="mx-auto h-36 object-contain"
            />
            <div className="mt-6 text-center text-xl">
              {step === 1
                ? "Read 101?"
                : step === 2
                  ? "Save 101 Maya Chen · ID 42"
                  : "Retry Save 101 Maya Chen · ID 42"}
            </div>
            <p className="mt-5 text-center text-2xl text-brand">
              {step === 2
                ? "? No reply"
                : step === 3
                  ? "✓ Same saved answer"
                  : "Maya Chen"}
            </p>
          </Panel>
          <Arrow />
          <Panel title="What the server knows">
            <Table rows={updated} highlight="101" compact />
            <p className="mt-6 text-center text-lg text-emerald-700">
              {step === 1
                ? "Leader checked with the group"
                : "✓ ID 42 already saved"}
            </p>
          </Panel>
        </div>
      );
    caption = [
      "Server 3 is behind. Do not trust its old value.",
      "The leader checks first, then returns a fresh value.",
      "No reply means unknown. The save may have worked.",
      "Same ID again. No second execution. Same answer.",
    ][step];
    detail =
      step >= 2
        ? "42 is a ticket for this request, not the row key. A new ticket means a new operation; another save to key 101 still updates the same row."
        : "The displayed tables are samples. A verified read checks with the group.";
    controls = next(3);
  } else if (topic === 8) {
    diagram = (
      <>
        <div className="grid items-center gap-5 lg:grid-cols-[1fr_auto_1fr_auto_1fr]">
          <Panel title="Browser">
            <img
              src="/client-computer.png"
              alt="Client browser"
              className="mx-auto h-40 object-contain"
            />
          </Panel>
          <Arrow label="HTTP" />
          <Panel title="Controller">
            <p className="py-12 text-center text-xl text-gray-500">
              Routes your request
            </p>
          </Panel>
          <Arrow label="gRPC" />
          <div>
            <div className="mx-auto h-40 w-40">
              <ReplicaArtwork />
            </div>
            <p className="mt-3 text-center text-xl">Raft servers</p>
          </div>
        </div>
        <div className="mt-8 grid gap-5 sm:grid-cols-2">
          <Panel title="Execute">
            <p className="text-center text-xl">Your work: Save, Read, Delete</p>
          </Panel>
          <Panel title="Deliver">
            <p className="text-center text-xl">
              Server messages: votes, logs, snapshots
            </p>
          </Panel>
        </div>
      </>
    );
    caption = "gRPC carries messages. Raft decides agreement.";
  } else {
    diagram = (
      <>
        <Columns>
          {[0, 1, 2].map((id) => (
            <Panel key={id} title={`Pod shard-1-${id}`}>
              <div
                className={`mx-auto h-40 w-40 transition-opacity ${step === 1 && id === 0 ? "opacity-20" : ""}`}
              >
                <ReplicaArtwork />
              </div>
              <p className="mt-5 text-center text-xl">
                {id === 0 && step === 1
                  ? "Old process stopped"
                  : id === 0 && step === 2
                    ? "New process · same name"
                    : "Running server"}
              </p>
              <div className="mt-6 rounded-xl bg-gray-100 px-4 py-2 text-center text-lg">
                PVC data-shard-1-{id}
                <br />
                <span className="text-sm text-gray-500">Saved disk stays</span>
              </div>
            </Panel>
          ))}
        </Columns>
        <div className="mx-auto mt-7 max-w-md">
          <Table rows={original} compact />
        </div>
      </>
    );
    caption =
      step === 0
        ? "A Pod runs a server. Its saved disk stays."
        : step === 1
          ? "Old Pod stops. Its stored rows remain."
          : "New Pod. Same name. Same saved disk.";
    detail =
      "Kubernetes replaces the process. Raft still picks the leader and checks for a majority.";
    controls = next(2);
  }
  return (
    <section className="mt-7" aria-label="Visual database concepts">
      <nav
        className="mb-8 flex flex-wrap justify-center gap-2"
        aria-label="Concepts"
      >
        {topics.map((name, i) => (
          <button
            key={name}
            onClick={() => changeTopic(i)}
            aria-current={topic === i ? "page" : undefined}
            className={`rounded-full border px-4 py-2 text-sm ${topic === i ? "border-brand bg-brand text-white" : "border-gray-200 bg-white text-gray-500"}`}
          >
            {i + 1}. {name}
          </button>
        ))}
      </nav>
      <div className="rounded-[32px] bg-slate-50/70 px-5 py-6 sm:px-8">
        <div style={topic === 0 ? { zoom: 0.94 } : undefined}>
          <h2 className="mb-5 text-center text-3xl font-semibold tracking-tight sm:text-4xl">
            {topics[topic]}
          </h2>
          {controls && (
            <div className="sticky top-3 z-10 mx-auto mb-5 flex w-fit max-w-full flex-wrap justify-center gap-3 rounded-2xl border border-gray-200 bg-white/95 p-2 shadow-sm backdrop-blur">
              {controls}
            </div>
          )}
          {diagram}
          <p
            className="mx-auto mt-6 max-w-4xl text-center text-xl font-semibold leading-tight tracking-tight sm:text-2xl"
            role="status"
          >
            {caption}
          </p>
          {detail && (
            <p className="mx-auto mt-4 max-w-3xl text-center text-base leading-relaxed text-gray-500">
              {detail}
            </p>
          )}
        </div>
      </div>
      <div className="mt-4 flex flex-wrap justify-between gap-3 text-xs text-gray-400">
        <span>Concept illustration · example data</span>
        {onOpenLab && (
          <button
            onClick={onOpenLab}
            className="text-brand underline underline-offset-4"
          >
            Open live cluster →
          </button>
        )}
      </div>
    </section>
  );
}

function Panel({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-3xl border-2 border-gray-200 bg-white p-5 sm:p-7">
      <h3 className="mb-5 text-center text-xl font-semibold sm:text-2xl">
        {title}
      </h3>
      {children}
    </div>
  );
}
function Columns({ children }: { children: React.ReactNode }) {
  return <div className="grid gap-6 md:grid-cols-3">{children}</div>;
}
function Tag({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded-xl bg-blue-50 px-4 py-2 text-base text-brand">
      {children}
    </span>
  );
}
function Arrow({ label }: { label?: string }) {
  return (
    <div className="text-center text-brand">
      <p className="text-base font-medium">{label}</p>
      <span className="inline-block text-5xl lg:text-6xl" aria-hidden="true">
        →
      </span>
    </div>
  );
}
function Server({
  id,
  rows,
  role,
  offline,
  highlight,
  staleKey,
  note,
}: {
  id: number;
  rows: Row[];
  role?: string;
  offline?: boolean;
  highlight?: string;
  staleKey?: string;
  note?: string;
}) {
  return (
    <div>
      <div
        className={`mx-auto h-44 w-36 transition-opacity sm:h-48 sm:w-40 ${offline ? "opacity-25" : ""}`}
      >
        <ReplicaArtwork />
      </div>
      <h3 className="mt-2 text-center text-xl font-semibold">
        Server {id + 1}
      </h3>
      {role && (
        <p
          className={`mt-1 text-center text-base ${role.includes("leader") || role === "Leader" ? "text-brand" : "text-gray-500"}`}
        >
          {role}
        </p>
      )}
      <div className={`mt-3 transition-opacity ${offline ? "opacity-40" : ""}`}>
        <Table rows={rows} highlight={highlight} staleKey={staleKey} compact />
      </div>
      {note && <p className="mt-4 text-center text-sm text-brand">{note}</p>}
    </div>
  );
}
function Table({
  rows,
  highlight,
  staleKey,
  compact,
  onSelect,
}: {
  rows: Row[];
  highlight?: string;
  staleKey?: string;
  compact?: boolean;
  onSelect?: (id: string) => void;
}) {
  return (
    <div className="reference-table">
      <h3
        className={`reference-table-title text-center font-medium text-gray-600 ${compact ? "py-2 text-sm" : "py-4 text-xl"}`}
      >
        USERS TABLE
      </h3>
      <div className="reference-table-grid">
        <table
          className={`w-full border-collapse text-left ${compact ? "text-base" : "text-xl sm:text-2xl"}`}
        >
          <thead className="bg-gray-100 text-gray-500">
            <tr>
              <th className="border-r border-gray-200 px-4 py-2 font-medium">
                ID
              </th>
              <th className="px-4 py-2 font-medium">STUDENT</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr
                key={row.id}
                className={`border-t border-gray-200 transition-colors ${highlight === row.id ? "bg-violet-50" : ""}`}
              >
                <td className="border-r border-gray-200 px-4 py-2">
                  {onSelect ? (
                    <button
                      onClick={() => onSelect(row.id)}
                      className="text-brand underline underline-offset-4"
                    >
                      {row.id}
                    </button>
                  ) : (
                    row.id
                  )}
                </td>
                <td className="px-4 py-2">
                  <span
                    className="mr-2 inline-block size-3 rounded-full sm:size-4"
                    style={{ background: row.color }}
                  />
                  {row.name}
                  {staleKey === row.id && (
                    <span className="ml-2 inline-block rounded-md bg-red-500 px-1.5 py-0.5 align-middle text-xs font-semibold leading-none text-white">
                      OLD
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
// Match the backend's stable FNV-1a rule, rather than inventing an even split.
function shard(key: string) {
  let hash = 2166136261;
  for (const byte of new TextEncoder().encode(key))
    hash = Math.imul(hash ^ byte, 16777619) >>> 0;
  return hash % 2;
}

// Draw each instruction as a sheet, so compaction removes pages without
// confusing the log history with the rows produced by applying that history.
function LogPages({ compacted }: { compacted: boolean }) {
  return (
    <div className="mx-auto w-full max-w-sm">
      <svg
        viewBox="0 0 400 460"
        className="w-full"
        role="img"
        aria-label={
          compacted
            ? "Log pages 13 and 14 remain after snapshot through 12"
            : "Fourteen stacked log pages"
        }
      >
        <defs>
          <linearGradient id="log-paper" x1="0" y1="0" x2="1" y2="1">
            <stop stopColor="#faf9f5" />
            <stop offset=".6" stopColor="#eeece4" />
            <stop offset="1" stopColor="#dcd8cc" />
          </linearGradient>
          <filter
            id="log-paper-shadow"
            x="-20%"
            y="-30%"
            width="150%"
            height="180%"
          >
            <feDropShadow
              dx="0"
              dy="5"
              stdDeviation="4"
              floodColor="#635b48"
              floodOpacity=".22"
            />
          </filter>
        </defs>
        {Array.from({ length: 14 }, (_, i) => {
          const covered = compacted && i < 12;
          const y = compacted ? (i - 12) * 28 + 128 : i * 26 + 8;
          return (
            <g
              key={i}
              transform={`translate(0 ${y})`}
              style={{
                opacity: covered ? 0 : 1,
                transition: "opacity 650ms ease",
                pointerEvents: "none",
              }}
            >
              <path d="M92 7 L367 15 L313 105 L27 91 Z" fill="#d3cec0" />
              <path
                d="M92 4 L367 12 L313 102 L27 88 Z"
                fill="#f4f1e8"
                stroke="#d0cbbe"
                strokeWidth=".6"
              />
              <path
                d="M92 1 L367 9 L313 99 L27 85 Z"
                fill="url(#log-paper)"
                stroke="#e3dfd3"
                strokeWidth=".6"
                filter="url(#log-paper-shadow)"
              />
              <text
                x="207"
                y="43"
                textAnchor="middle"
                fill="#8b887f"
                fontSize="17"
                letterSpacing="3"
                transform="rotate(2 207 43)"
              >
                LOGS
              </text>
              <text
                x="184"
                y="94"
                textAnchor="middle"
                fill="#555650"
                fontSize="16"
              >
                {i + 1} · saved instruction
              </text>
            </g>
          );
        }).reverse()}
        {compacted && (
          <text
            x="200"
            y="300"
            textAnchor="middle"
            fill="#657083"
            fontSize="16"
          >
            Pages 1–12 covered by the snapshot
          </text>
        )}
      </svg>
    </div>
  );
}
