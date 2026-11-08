"use client";
import { useEffect, useRef, useState } from "react";
import type { Cluster, Message } from "./types";

export interface Pulse {
  message: Message;
  progress: number;
  status: "pending" | "delivered" | "dropped";
}
export function usePulses(cluster: Cluster | null) {
  const [pulses, setPulses] = useState<Pulse[]>([]);
  const entries = useRef(new Map<number, Pulse>());
  const previous = useRef<{
    tick: number;
    serial: number;
    delivery: number;
  } | null>(null);
  const timers = useRef(new Set<ReturnType<typeof setTimeout>>());
  const epoch = useRef(0);
  useEffect(
    () => () => {
      for (const timer of timers.current) clearTimeout(timer);
    },
    [],
  );
  useEffect(() => {
    if (!cluster) return;
    const old = previous.current;
    const reset =
      !!old && (cluster.Tick < old.tick || cluster.Serial < old.serial);
    if (reset) {
      epoch.current++;
      entries.current.clear();
      for (const timer of timers.current) clearTimeout(timer);
      timers.current.clear();
    }
    for (const message of cluster.Messages || []) {
      const progress = Math.max(
        0,
        Math.min(
          1,
          (cluster.Tick - message.Sent) / (message.Due - message.Sent),
        ),
      );
      entries.current.set(message.ID, { message, progress, status: "pending" });
    }
    if (old && !reset)
      for (const delivery of cluster.Deliveries || []) {
        if (delivery.Sequence <= old.delivery) continue;
        const id = delivery.Message.ID;
        const original = entries.current.get(id)?.message || delivery.Message;
        entries.current.set(id, {
          message: original,
          progress: delivery.Status === "dropped" ? 0.9 : 1,
          status: delivery.Status,
        });
        const generation = epoch.current;
        const timer = setTimeout(() => {
          timers.current.delete(timer);
          if (generation !== epoch.current) return;
          entries.current.delete(id);
          setPulses([...entries.current.values()]);
        }, 1150);
        timers.current.add(timer);
      }
    previous.current = {
      tick: cluster.Tick,
      serial: cluster.Serial,
      delivery: cluster.DeliverySerial,
    };
    setPulses([...entries.current.values()]);
  }, [cluster]);
  return pulses;
}
