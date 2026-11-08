import type { PeerObservation } from "./types";

// Slow display-only hops leave enough time to read each travelling label.
export const REQUEST_HOP_SECONDS = 2.4;
export const REQUEST_TRAVEL_SECONDS = REQUEST_HOP_SECONDS * 2;

export const playbackEventKey = (event: PeerObservation) => `${event.From}:${event.To}:${event.Kind}`;

type EventSchedule = {
  events: PeerObservation[];
  starts: Record<string, number>;
  applyAt: number;
  endAt: number;
};

// Polls sample different servers at different moments. Keep each observed link
// once, including ACKs received after playback began. These are display times,
// not measured Raft timestamps or a requirement to wait for every follower.
export function collectPlaybackEvents(
  current: (EventSchedule & { elapsed: number; active: boolean }) | null,
  observations: PeerObservation[],
  requestID: string,
): EventSchedule {
  const events = new Map((current?.events || []).map((event) => [playbackEventKey(event), event]));
  const starts = { ...current?.starts };
  const incoming = observations.filter((event) => event.RequestID === requestID)
    .sort((a, b) => Number(a.Kind === "ack") - Number(b.Kind === "ack"));
  for (const event of incoming) {
    const key = playbackEventKey(event);
    if (events.has(key)) continue;
    events.set(key, event);
    const ack = event.Kind === "ack";
    let start = (ack ? REQUEST_TRAVEL_SECONDS + 2.4 : REQUEST_TRAVEL_SECONDS) + (ack ? event.From : event.To) * 0.1;
    if (ack) {
      const copyStart = starts[`${event.To}:${event.From}:append`];
      if (copyStart !== undefined) start = Math.max(start, copyStart + 2.15);
    }
    // A late packet still gets a visible journey in an active replay.
    if (current?.active) start = Math.max(start, current.elapsed + 0.15);
    starts[key] = start;
  }
  const lastArrival = Math.max(0, ...[...events.values()].map((event) =>
    starts[playbackEventKey(event)] + (event.Kind === "ack" ? 1.3 : 2)));
  const applyAt = Math.max(REQUEST_TRAVEL_SECONDS + 4, lastArrival + 0.15);
  return { events: [...events.values()], starts, applyAt, endAt: applyAt + REQUEST_TRAVEL_SECONDS + 0.4 };
}
