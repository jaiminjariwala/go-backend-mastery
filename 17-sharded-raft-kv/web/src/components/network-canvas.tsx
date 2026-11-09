"use client";

import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { useLayoutEffect, useRef, useState } from "react";
import { ReplicaArtwork } from "./replica-artwork";
import { PeerLogReplay } from "./peer-log-replay";
import { ReplicaRows } from "./replica-rows";
import type { Cluster, ClientExchange } from "@/lib/types";
import {
  REQUEST_HOP_SECONDS,
  REQUEST_TRAVEL_SECONDS,
} from "@/lib/playback-events";
import { type OperationPlayback } from "@/lib/use-operation-playback";
import { usePulses } from "@/lib/use-pulses";

import {
  replicaRoute as peerRoute,
  replicaLeftEdge,
  type ReplicaBox as Box,
} from "@/lib/replica-route";
interface Props {
  cluster: Cluster | null;
  exchange: ClientExchange | null;
  playback: OperationPlayback | null;
  onReplay: () => void;
  onSkipPlayback: () => void;
  busy: boolean;
  rowBusy: boolean;
  speed: number;
  showHeartbeats: boolean;
  onToggle: (id: number) => void;
  onSelect: (id: number) => void;
  onEdit: (key: string, value: string) => void;
}
export function NetworkCanvas({
  cluster,
  exchange,
  playback,
  onReplay,
  onSkipPlayback,
  busy,
  rowBusy,
  speed,
  showHeartbeats,
  onToggle,
  onSelect,
  onEdit,
}: Props) {
  const canvas = useRef<HTMLDivElement>(null);
  const anchors = useRef<(HTMLButtonElement | null)[]>([]);
  const client = useRef<HTMLDivElement>(null);
  const controller = useRef<HTMLDivElement>(null);
  const [boxes, setBoxes] = useState<Box[]>([]);
  const [clientBox, setClientBox] = useState<Box | null>(null);
  const [controllerBox, setControllerBox] = useState<Box | null>(null);
  const pulses = usePulses(cluster);
  const reduce = useReducedMotion();
  const servers = cluster?.Nodes || [];
  const reportedLeader = servers
    .filter((n) => n.Up && n.Role === "leader")
    .sort((a, b) => b.Term - a.Term)[0];
  const sameShard =
    exchange?.ShardID === undefined ||
    exchange.ShardID === (cluster?.ShardID || 0);
  const replayHere =
    playback?.active && playback.shardID === (cluster?.ShardID || 0);
  const target =
    exchange && sameShard
      ? (exchange?.ServerID ?? exchange?.Target ?? reportedLeader?.ID)
      : undefined;
  useLayoutEffect(() => {
    if (!canvas.current) return;
    const element = canvas.current;
    const measure = () => {
      const base = element.getBoundingClientRect();
      const box = (node: HTMLElement | null): Box | null => {
        if (!node) return null;
        const r = node.getBoundingClientRect();
        return {
          x: r.left + r.width / 2 - base.left,
          y: r.top + r.height / 2 - base.top,
          width: r.width,
          height: r.height,
        };
      };
      const result = anchors.current.map(box);
      if (result.length === 3 && result.every((v): v is Box => v !== null))
        setBoxes((previous) =>
          JSON.stringify(previous) === JSON.stringify(result)
            ? previous
            : result,
        );
      setClientBox((previous) => {
        const next = box(client.current);
        return JSON.stringify(previous) === JSON.stringify(next)
          ? previous
          : next;
      });
      setControllerBox((previous) => {
        const next = box(controller.current);
        return JSON.stringify(previous) === JSON.stringify(next)
          ? previous
          : next;
      });
    };
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    // The canvas can stay the same size while its right-aligned servers move.
    // Observe their layout containers too, so lines follow the actual artwork.
    for (const anchor of anchors.current) {
      if (anchor?.parentElement?.parentElement)
        observer.observe(anchor.parentElement.parentElement);
    }
    if (controller.current) observer.observe(controller.current);
    measure();
    return () => observer.disconnect();
  }, [cluster, exchange]);
  const targetBox = target !== undefined ? boxes[target] : undefined;
  // Centre the controller in the free space between the client artwork and
  // the leftmost bend of the replica links, rather than using a fixed offset.
  const controllerCenterX =
    clientBox && boxes.length === 3
      ? (clientBox.x +
          clientBox.width / 2 +
          Math.min(...boxes.map(replicaLeftEdge)) -
          51) /
        2
      : undefined;
  // HTTP reaches the controller first. Execute then carries the command over
  // gRPC to the selected server, where Raft peer replication begins.
  const showingRequest =
    exchange && sameShard && (exchange.Phase === "waiting" || replayHere);
  const requestPaths =
    showingRequest && clientBox && controllerBox && targetBox
      ? [
          `M ${clientBox.x + clientBox.width / 2 + 8} ${clientBox.y} L ${controllerBox.x - controllerBox.width / 2 - 8} ${controllerBox.y}`,
          `M ${controllerBox.x + controllerBox.width / 2 + 8} ${controllerBox.y} C ${controllerBox.x + controllerBox.width / 2 + 55} ${controllerBox.y}, ${targetBox.x - targetBox.width / 2 - 90} ${targetBox.y}, ${replicaLeftEdge(targetBox)} ${targetBox.y}`,
        ]
      : [];
  const clientPath = requestPaths.length ? requestPaths.join(" ") : null;
  const duration = reduce ? 0 : Math.min(0.48, (speed / 1000) * 0.8);
  return (
    <div className="mt-3">
      {playback && playback.shardID === (cluster?.ShardID || 0) && (
        <div
          data-playback-run={playback.run}
          data-playback-phase={
            replayHere
              ? playback.elapsed < playback.applyAt
                ? "before-apply"
                : "applied"
              : "live"
          }
          className={`mb-2 flex flex-wrap items-center gap-2 text-xs text-gray-600 ${replayHere ? "justify-between rounded-lg bg-blue-50/60 px-3 py-2" : "justify-end"}`}
          role="status"
        >
          {replayHere && (
            <span>
              {`Recorded operation replay · ${playback.elapsed < REQUEST_TRAVEL_SECONDS ? "Client request" : playback.elapsed < REQUEST_TRAVEL_SECONDS + 2.4 ? "Copying log instructions" : playback.elapsed < playback.applyAt ? "Followers acknowledge stored instructions" : "Showing observed applied rows"}`}
              <span className="ml-2 text-gray-400">
                Slowed playback with before/after samples. The database runs at
                real speed.
              </span>
            </span>
          )}
          <button
            className="shrink-0 text-brand underline underline-offset-2"
            onClick={replayHere ? onSkipPlayback : onReplay}
          >
            {replayHere ? "Show live rows" : "Replay last operation"}
          </button>
        </div>
      )}
      <div className="mt-3 overflow-x-auto">
        <div
          ref={canvas}
          className="relative h-180 min-w-240"
          aria-label="Browser to Go controller to the selected shard’s servers"
        >
          <svg
            className="pointer-events-none absolute inset-0 size-full overflow-visible"
            aria-hidden="true"
          >
            {boxes.length === 3 &&
              [
                [0, 1],
                [0, 2],
                [1, 2],
              ].map(([a, b]) => (
                <path
                  key={`${a}-${b}`}
                  d={peerRoute(boxes[a], boxes[b])}
                  fill="none"
                  stroke={
                    cluster?.Isolated?.[a] || cluster?.Isolated?.[b]
                      ? "#b78264"
                      : "#dce2ea"
                  }
                  strokeWidth="1.5"
                  strokeDasharray={
                    !servers[a]?.Up ||
                    !servers[b]?.Up ||
                    cluster?.Isolated?.[a] ||
                    cluster?.Isolated?.[b]
                      ? "4 5"
                      : undefined
                  }
                />
              ))}
            {clientPath && (
              <path
                d={clientPath}
                fill="none"
                stroke="#a7bfe9"
                strokeWidth="1.5"
                strokeDasharray={
                  target !== undefined && !servers[target]?.Up
                    ? "4 5"
                    : undefined
                }
              />
            )}
            {replayHere &&
              playback &&
              clientPath &&
              exchange &&
              !reduce &&
              (() => {
                const responseAt = playback.applyAt + 0.2;
                const returning = playback.elapsed >= responseAt;
                if (
                  (playback.elapsed >= REQUEST_TRAVEL_SECONDS && !returning) ||
                  (returning && exchange.Phase !== "returned")
                )
                  return null;
                const elapsed = returning
                  ? playback.elapsed - responseAt
                  : playback.elapsed;
                return requestPaths.map((path, leg) => {
                  const progress =
                    (elapsed -
                      (returning ? 1 - leg : leg) * REQUEST_HOP_SECONDS) /
                    REQUEST_HOP_SECONDS;
                  if (progress < 0 || progress > 1) return null;
                  return (
                    <path
                      key={leg}
                      d={path}
                      pathLength={1}
                      fill="none"
                      stroke={returning ? "#648e9c" : "#2962cd"}
                      strokeWidth="3"
                      strokeLinecap="round"
                      strokeDasharray="0.06 1"
                      strokeDashoffset={returning ? -1 + progress : -progress}
                    />
                  );
                });
              })()}
            <AnimatePresence>
              {!cluster?.Live && clientPath && exchange && sameShard && (
                <motion.path
                  key={`client-send-${exchange.ID}`}
                  d={clientPath}
                  pathLength={1}
                  fill="none"
                  stroke="#2962cd"
                  strokeWidth="3"
                  strokeLinecap="round"
                  strokeDasharray="0.06 1"
                  initial={{ strokeDashoffset: 0.06, opacity: 0 }}
                  animate={{ strokeDashoffset: -1, opacity: [0, 1, 1, 0] }}
                  transition={{ duration: reduce ? 0 : 0.8, ease: "easeInOut" }}
                />
              )}
              {!cluster?.Live &&
                clientPath &&
                exchange &&
                exchange.Phase !== "waiting" &&
                sameShard && (
                  <motion.path
                    key={`client-return-${exchange.ID}`}
                    d={clientPath}
                    pathLength={1}
                    fill="none"
                    stroke={
                      exchange.Phase === "returned" ? "#648e9c" : "#c28b72"
                    }
                    strokeWidth="3"
                    strokeLinecap="round"
                    strokeDasharray="0.06 1"
                    initial={{ strokeDashoffset: -1, opacity: 0 }}
                    animate={{ strokeDashoffset: 0.06, opacity: [0, 1, 1, 0] }}
                    transition={{
                      duration: reduce ? 0 : 0.8,
                      ease: "easeInOut",
                    }}
                  />
                )}
              {boxes.length === 3 &&
                pulses
                  .filter((p) => showHeartbeats || !p.message.Quiet)
                  .map((p) => {
                    const m = p.message;
                    return (
                      <motion.path
                        key={`peer-${m.ID}`}
                        d={peerRoute(boxes[m.From], boxes[m.To])}
                        pathLength={100}
                        fill="none"
                        stroke={
                          p.status === "dropped"
                            ? "#c28b72"
                            : m.Kind === "ack" || m.Kind === "ballot"
                              ? "#648e9c"
                              : "#2962cd"
                        }
                        strokeWidth="2"
                        strokeDasharray="4 105"
                        initial={{ strokeDashoffset: 4 }}
                        animate={{
                          strokeDashoffset: 4 - 104 * p.progress,
                          opacity: p.status === "pending" ? 1 : 0,
                        }}
                        transition={{ duration, ease: "linear" }}
                      />
                    );
                  })}
            </AnimatePresence>
          </svg>
          {replayHere &&
            playback &&
            !reduce &&
            requestPaths.map((path, leg) => {
              const returning = playback.elapsed >= playback.applyAt + 0.2;
              const elapsed = returning
                ? playback.elapsed - playback.applyAt - 0.2
                : playback.elapsed;
              const progress =
                (elapsed - (returning ? 1 - leg : leg) * REQUEST_HOP_SECONDS) /
                REQUEST_HOP_SECONDS;
              if (
                progress < 0 ||
                progress > 1 ||
                (returning && exchange?.Phase !== "returned")
              )
                return null;
              return (
                <div
                  key={`request-label-${leg}`}
                  className="pointer-events-none absolute left-0 top-0 z-30"
                  style={{
                    offsetPath: `path("${path}")`,
                    offsetRotate: "0deg",
                    offsetAnchor: "center",
                    offsetDistance: `${(returning ? 1 - progress : progress) * 100}%`,
                  }}
                >
                  <div className="-translate-y-6 whitespace-nowrap rounded-md border border-blue-100 bg-white px-2 py-1 text-ui-11 font-semibold text-brand shadow-sm">
                    {returning
                      ? leg === 0
                        ? "HTTP response"
                        : "gRPC response"
                      : leg === 0
                        ? "HTTP POST request"
                        : "gRPC Execute"}
                  </div>
                </div>
              );
            })}
          <PeerLogReplay
            playback={replayHere ? playback : null}
            boxes={boxes}
          />
          <div className="absolute left-2 top-[calc(50%-56px)] w-28 text-center">
            <div ref={client}>
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src="/client-computer.png"
                alt="Client computer"
                className="mx-auto size-28 object-contain"
              />
            </div>
            <p className="mt-2 text-sm font-semibold">Client</p>
            <p className="mt-1 text-xs text-gray-400">Your browser</p>
          </div>
          <div
            className="absolute top-1/2 w-52 -translate-x-1/2 -translate-y-1/2 text-center"
            style={{ left: controllerCenterX ?? "calc(50% - 179px)" }}
          >
            <div
              ref={controller}
              className="rounded-2xl border border-[#c7c9d3] bg-white px-3 py-3 shadow-sm"
            >
              <h3 className="text-sm font-semibold">Go controller</h3>
              <div className="mt-3 space-y-2 text-left text-xs leading-5 text-gray-600">
                <p>Receives your HTTP request.</p>
                <p>
                  <strong className="font-medium text-gray-800">
                    Hashes the key
                  </strong>{" "}
                  to pick a shard.
                </p>
                <p>
                  Finds that shard’s leader and sends your request by{" "}
                  <strong className="font-medium text-gray-800">gRPC</strong>.
                </p>
                <p>Returns the response to your browser.</p>
              </div>
              {exchange && sameShard && (
                <div
                  className="mt-2 border-t border-gray-200 pt-1 text-xs leading-4"
                  aria-live="polite"
                >
                  <p
                    className="truncate font-mono text-gray-700"
                    title={exchange.Key}
                  >
                    {exchange.Key}
                  </p>
                  <p className="mt-0.5 text-brand">
                    Shard {(cluster?.ShardID || 0) + 1}
                    {target !== undefined
                      ? ` · Server ${target + 1}`
                      : " · finding leader"}
                  </p>
                </div>
              )}
            </div>
          </div>
          {servers.map((n) => (
            <article
              key={n.ID}
              className={`absolute right-0 flex w-116 items-center gap-4 ${n.ID === 0 ? "top-1" : n.ID === 1 ? "top-61" : "top-121"}`}
            >
              <div className="w-32 shrink-0 text-center">
                <button
                  ref={(el) => {
                    anchors.current[n.ID] = el;
                  }}
                  onClick={() => onSelect(n.ID)}
                  aria-label={`Inspect Server ${n.ID + 1}`}
                  className="relative mx-auto block h-26 w-24 rounded-md"
                >
                  <motion.span
                    className="absolute inset-0"
                    animate={{
                      opacity: n.Up ? 1 : 0.3,
                      filter: n.Up ? "grayscale(0)" : "grayscale(0.8)",
                    }}
                    transition={{ duration: reduce ? 0 : 0.35 }}
                  >
                    <ReplicaArtwork />
                  </motion.span>
                </button>
                <h3 className="mt-2 text-sm font-semibold">
                  Server {n.ID + 1}
                </h3>
                <p
                  className={`mt-1 whitespace-nowrap text-xs capitalize ${n.Role === "leader" ? "text-brand" : "text-gray-500"}`}
                >
                  {n.Role}
                  {n.Up ? ` · term ${n.Term}` : ""}
                </p>
                <button
                  onClick={() => onToggle(n.ID)}
                  disabled={busy || cluster?.Deployment === "kubernetes"}
                  aria-label={
                    cluster?.Deployment === "kubernetes"
                      ? `Server ${n.ID + 1} Pod managed by Kubernetes`
                      : `${n.Up ? "Stop" : "Restart"} Server ${n.ID + 1}`
                  }
                  className="mt-1 rounded px-2 py-1 text-xs text-gray-400 hover:text-gray-700 disabled:opacity-40"
                >
                  {cluster?.Deployment === "kubernetes"
                    ? "Pod managed by Kubernetes"
                    : n.Up
                      ? "Stop"
                      : "Restart"}
                </button>
                {cluster?.Isolated?.[n.ID] && (
                  <p className="mt-1 text-ui-11 text-partition">
                    Isolated · still running
                  </p>
                )}
              </div>
              <div className="min-w-0 max-w-80 flex-1">
                <ReplicaRows
                  key={`${cluster?.ShardID || 0}:${n.ID}`}
                  label={`Shard ${(cluster?.ShardID || 0) + 1} Server ${n.ID + 1} rows`}
                  shardNumber={(cluster?.ShardID || 0) + 1}
                  playbackLabel={
                    replayHere
                      ? playback.elapsed < playback.applyAt
                        ? "Replay · before apply"
                        : "Replay · sampled result"
                      : undefined
                  }
                  values={
                    replayHere
                      ? (playback.elapsed < playback.applyAt
                          ? playback.before[n.ID]
                          : playback.after[n.ID]) || n.KV
                      : n.KV
                  }
                  online={n.Up}
                  busy={rowBusy}
                  onEdit={onEdit}
                />
              </div>
            </article>
          ))}
          {!cluster && (
            <p className="absolute inset-0 flex items-center justify-center text-sm text-gray-400">
              Connecting to the Go backend…
            </p>
          )}
        </div>
      </div>
      {cluster?.Live && (
        <p className="mt-2 text-center text-sm text-gray-500" role="status">
          {!exchange
            ? "Save, read, or remove a row to follow its request."
            : !sameShard
              ? `Last request belongs to Shard ${(exchange.ShardID || 0) + 1}. Select that group to see its destination.`
              : exchange.Phase === "waiting"
                ? "Request sent · waiting for commitment."
                : exchange.Phase === "returned"
                  ? `${exchange.Operation === "delete" ? "Row removed" : exchange.Operation === "write" ? "Saved" : "Verified read"} · response received from Server ${(exchange.ServerID || 0) + 1}.`
                  : "No success confirmed. A timed-out mutation may still commit; retain its request ID."}
        </p>
      )}
    </div>
  );
}
