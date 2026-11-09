"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { shardOwner } from "./shard-owner";
import type {
  Action,
  Cluster,
  Receipt,
  PendingWrite,
  ClientExchange,
} from "./types";

export function useCluster(key = "") {
  const shardCountRef = useRef(1);
  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [running, setRunning] = useState(false);
  const [speed, setSpeed] = useState(600);
  const lock = useRef(false);
  const [selectedShard, setSelectedShard] = useState(0);
  const shardRef = useRef(0);
  const keyRef = useRef(key);
  keyRef.current = key;
  const selectShard = useCallback((id: number) => {
    shardRef.current = id;
    setSelectedShard(id);
  }, []);
  const clientLock = useRef(false);
  const [exchange, setExchange] = useState<ClientExchange | null>(null);
  const [clientBusy, setClientBusy] = useState(false);
  const [clientError, setClientError] = useState("");
  const [receipt, setReceipt] = useState<Receipt | null>(null);
  const [pending, setPending] = useState<PendingWrite | null>(null);
  const pendingRef = useRef<PendingWrite | null>(null);
  useEffect(() => {
    try {
      const stored = JSON.parse(
        sessionStorage.getItem("raft-pending-write") || "null",
      ) as PendingWrite | null;
      if (
        stored &&
        typeof stored.Key === "string" &&
        typeof stored.Value === "string" &&
        typeof stored.RequestID === "string" &&
        (stored.Operation === undefined ||
          stored.Operation === "write" ||
          stored.Operation === "delete")
      ) {
        pendingRef.current = stored;
        setPending(stored);
      }
    } catch {
      /* Storage may be unavailable; retries still work in this tab. */
    }
  }, []);
  const operate = useCallback(
    async (
      operation: "write" | "read" | "delete",
      key: string,
      value = "",
      retry = false,
      target?: number,
    ) => {
      if (clientLock.current) return;
      clientLock.current = true;
      setClientBusy(true);
      setClientError("");
      setReceipt(null);
      let write: PendingWrite | null = null;
      try {
        if (operation !== "read") {
          if (operation === "delete") value = "";
          const previous = pendingRef.current;
          if (
            previous &&
            !retry &&
            (previous.Key !== key ||
              previous.Value !== value ||
              (previous.Operation || "write") !== operation)
          ) {
            throw new Error(
              `An earlier ${previous.Operation === "delete" ? "removal" : "save"} for ${previous.Key} is unconfirmed. Use Retry the earlier request below before saving different data.`,
            );
          }
          write = previous || {
            Operation: operation,
            Key: key,
            Value: value,
            RequestID: crypto.randomUUID(),
          };
          pendingRef.current = write;
          setPending(write);
          try {
            sessionStorage.setItem("raft-pending-write", JSON.stringify(write));
          } catch {
            /* Memory retains the ID. */
          }
        }
        // Take a before-sample from the owning group, even if another shard is
        // selected. Failure to observe must not block the real client request.
        const owner = shardOwner(write?.Key || key, shardCountRef.current);
        let beforeRows: ClientExchange["BeforeRows"];
        if (operation !== "read") {
          try {
            const observation = await fetch(`/api/cluster?shard=${owner}`, {
              cache: "no-store",
              signal: AbortSignal.timeout(1800),
            });
            if (observation.ok) {
              const sample = (await observation.json()) as Cluster;
              beforeRows = Object.fromEntries(
                sample.Nodes.map((node) => [node.ID, { ...node.KV }]),
              );
            }
          } catch {
            /* No sample means no row playback; the operation still runs. */
          }
        }
        // Animate only actual client operations, never background observation polls.
        const exchangeID = Date.now();
        setExchange({
          ID: exchangeID,
          BeforeRows: beforeRows,
          RequestID: write?.RequestID,
          Operation: operation,
          Key: write?.Key || key,
          ShardID: owner,
          Target: target,
          Phase: "waiting",
        });
        const response = await fetch("/api/kv", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          // Routing may change on retry; the operation's identity must not.
          body: JSON.stringify({
            Operation: operation,
            ...(write || { Key: key, Value: value }),
            Target: target,
          }),
          cache: "no-store",
          signal: AbortSignal.timeout(6000),
        });
        if (!response.ok) throw new Error(await response.text());
        const result = (await response.json()) as Receipt;
        setReceipt(result);
        setExchange({
          ID: exchangeID,
          BeforeRows: beforeRows,
          RequestID: write?.RequestID,
          Operation: operation,
          Key: result.Key,
          Phase: "returned",
          ShardID: result.ShardID,
          ServerID: result.ServerID,
        });
        if (result.ShardID !== undefined) selectShard(result.ShardID);
        if (write) {
          pendingRef.current = null;
          setPending(null);
          try {
            sessionStorage.removeItem("raft-pending-write");
          } catch {
            /* No persistent storage. */
          }
        }
        return result;
      } catch (e) {
        setExchange((previous) =>
          previous?.Phase === "waiting"
            ? { ...previous, Phase: "failed" }
            : previous,
        );
        setClientError(
          e instanceof Error
            ? e.message
            : "Outcome unknown; retry the same request ID.",
        );
        return null;
      } finally {
        clientLock.current = false;
        setClientBusy(false);
      }
    },
    [selectShard],
  );
  const request = useCallback(async (action?: Action) => {
    // Background observation may skip a refresh, but must not swallow a click.
    if (action) setBusy(true);
    while (lock.current) {
      if (!action) return false;
      await new Promise((resolve) => setTimeout(resolve, 25));
    }
    lock.current = true;
    if (action) setBusy(true);
    try {
      const requestedShard = shardRef.current;
      const response = await fetch(`/api/cluster?shard=${requestedShard}`, {
        method: action ? "POST" : "GET",
        headers: action ? { "Content-Type": "application/json" } : undefined,
        body: action ? JSON.stringify(action) : undefined,
        cache: "no-store",
        signal: AbortSignal.timeout(10000),
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      const snapshot = (await response.json()) as Cluster;
      shardCountRef.current = snapshot.ShardCount || 1;
      let overview: Pick<Cluster, "Shards" | "Route"> = {};
      if ((snapshot.ShardCount || 1) > 1) {
        const all = await fetch(
          `/api/shards?key=${encodeURIComponent(keyRef.current)}`,
          { cache: "no-store", signal: AbortSignal.timeout(4000) },
        );
        if (!all.ok)
          throw new Error("Cannot observe shard routing. Refresh state.");
        overview = await all.json();
      }
      // Ignore a slow result from a shard the student has already left.
      if (requestedShard === shardRef.current)
        setCluster({ ...snapshot, ...overview });
      setError("");
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed");
      setRunning(false);
      return false;
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }, []);
  useEffect(() => {
    void request();
  }, [request, selectedShard]);
  useEffect(() => {
    const timer = setTimeout(() => void request(), 180);
    return () => clearTimeout(timer);
  }, [key, request]);
  useEffect(() => {
    if (!cluster?.Live) return;
    const timer = setInterval(() => void request(), 600);
    return () => clearInterval(timer);
  }, [cluster?.Live, request]);
  useEffect(() => {
    if (!running || cluster?.Live) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;
    const advance = async () => {
      await request({ Action: "step" });
      if (!cancelled) timer = setTimeout(advance, speed);
    };
    timer = setTimeout(advance, speed);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [running, speed, request, cluster?.Live]);
  const recover = useCallback(async () => {
    const restored = await request({ Action: "recover" });
    if (restored) setClientError("");
    return restored;
  }, [request]);
  return {
    recover,
    cluster,
    selectedShard,
    selectShard,
    error: clientError || error,
    busy: busy || clientBusy,
    clientBusy,
    exchange,
    receipt,
    pending,
    operate,
    running,
    speed,
    setSpeed,
    setRunning,
    request,
  };
}
