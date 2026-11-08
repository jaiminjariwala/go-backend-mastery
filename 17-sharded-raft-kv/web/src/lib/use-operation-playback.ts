"use client";

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { ClientExchange, Cluster, PeerObservation } from "./types";
import { collectPlaybackEvents } from "./playback-events";

export type RowSamples = Record<number, Record<string, string>>;
export interface OperationPlayback {
  run: number;
  exchangeID: number;
  shardID: number;
  key: string;
  events: PeerObservation[];
  starts: Record<string, number>;
  applyAt: number;
  endAt: number;
  before: RowSamples;
  after: RowSamples;
  elapsed: number;
  active: boolean;
}

// Playback is a display of recorded messages and sampled rows. It never drives
// Raft or delays a database response. One clock survives shard/view switches.
export function useOperationPlayback(cluster: Cluster | null, exchange: ClientExchange | null) {
  const seen = useRef(new Set<number>(exchange ? [exchange.ID] : []));
  const [playback, setPlayback] = useState<OperationPlayback | null>(null);
  const startTime = useRef(0);
  const run = useRef(0);
  useLayoutEffect(() => {
    if (!cluster || !exchange?.BeforeRows || !exchange.RequestID ||
        exchange.Operation === "read" || exchange.ShardID !== (cluster.ShardID || 0)) return;
    if (seen.current.has(exchange.ID)) {
      // Polling can observe a follower before it applies the committed entry.
      // Merge late observations without restarting the clock. Finished runs collect
      // evidence for an explicit replay, but never restart themselves.
      setPlayback((current) => {
        if (!current || current.exchangeID !== exchange.ID) return current;
        const collected = collectPlaybackEvents(current, cluster.PeerObservations || [], exchange.RequestID!);
        return { ...current, ...collected,
          after: current.active && current.elapsed < current.applyAt
            ? Object.fromEntries(cluster.Nodes.map((node) => [node.ID, { ...node.KV }]))
            : current.after };
      });
      return;
    }
    const collected = collectPlaybackEvents(null, cluster.PeerObservations || [], exchange.RequestID);
    if (!collected.events.length) return;
    seen.current.add(exchange.ID);
    // Keep this session's display bookkeeping bounded.
    if (seen.current.size > 100) seen.current.delete(seen.current.values().next().value!);
    startTime.current = performance.now();
    setPlayback({
      run: ++run.current, exchangeID: exchange.ID, shardID: exchange.ShardID || 0,
      key: exchange.Key, ...collected, before: exchange.BeforeRows,
      after: Object.fromEntries(cluster.Nodes.map((node) => [node.ID, { ...node.KV }])),
      elapsed: 0, active: true,
    });
  }, [cluster, exchange]);

  useEffect(() => {
    if (!playback?.active) return;
    let frame = 0;
    const tick = () => {
      const clock = (performance.now() - startTime.current) / 1000;
      setPlayback((current) => current && ({ ...current,
        elapsed: Math.min(current.endAt, clock), active: clock < current.endAt }));
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [playback?.run, playback?.active]);

  // A new operation supersedes the old replay immediately. Switching shards
  // does not create a new run and cannot restart finished animation components.
  const current = playback?.exchangeID === exchange?.ID ? playback : null;
  return {
    playback: current,
    replay: () => {
      if (!current) return;
      startTime.current = performance.now();
      setPlayback({ ...current, ...collectPlaybackEvents(null, current.events, exchange!.RequestID!), run: ++run.current, elapsed: 0, active: true });
    },
    skip: () => setPlayback((value) => value && ({ ...value, elapsed: value.endAt, active: false })),
  };
}
