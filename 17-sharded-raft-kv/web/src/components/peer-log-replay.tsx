"use client";

import { useReducedMotion } from "motion/react";
import { playbackEventKey } from "@/lib/playback-events";
import type { OperationPlayback } from "@/lib/use-operation-playback";

import { replicaRoute, type ReplicaBox as Box } from "@/lib/replica-route";

// Position is driven by one parent-owned clock. Remounting on a shard switch
// cannot restart a log page or an acknowledgement that has already finished.
export function PeerLogReplay({
  playback,
  boxes,
}: {
  playback: OperationPlayback | null;
  boxes: Box[];
}) {
  const reduce = useReducedMotion();
  if (!playback?.active || reduce || boxes.length !== 3) return null;
  return (
    <>
      {playback.events.map((event) => {
        const a = boxes[event.From],
          b = boxes[event.To];
        if (!a || !b) return null;
        const ack = event.Kind === "ack";
        const start = playback.starts[playbackEventKey(event)];
        const duration = ack ? 1.3 : 2;
        const progress = (playback.elapsed - start) / duration;
        if (progress < 0 || progress > 1) return null;
        const path = replicaRoute(a, b);
        return (
          <div
            key={`${playback.run}:${event.ID}`}
            aria-label={`${ack ? "Acknowledgement" : event.Delete ? "Delete log instruction" : "Save log instruction"} from Server ${event.From + 1} to Server ${event.To + 1} for ${event.Key}`}
            className="pointer-events-none absolute left-0 top-0 z-20"
            style={{
              offsetPath: `path("${path}")`,
              offsetRotate: "0deg",
              offsetAnchor: "center",
              offsetDistance: `${progress * 100}%`,
            }}
          >
            {ack ? (
              <div className="rounded-full bg-reply px-2 py-0.5 text-xs font-medium text-white">
                ACK
              </div>
            ) : (
              <div className="w-24 rounded-lg border border-stone-300 bg-stone-50 px-2 py-2 text-center shadow-md">
                <div className="mb-1 text-sm font-semibold text-brand">LOG</div>
                <div className="text-xs font-semibold">
                  {event.Delete ? "Delete" : "Save"}
                </div>
                <div
                  className="truncate text-xs text-gray-700"
                  title={event.Key}
                >
                  {event.Key}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </>
  );
}
