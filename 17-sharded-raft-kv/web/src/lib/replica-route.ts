export type ReplicaBox = { x: number; y: number; width: number; height: number };

// The rack SVG fits a 780:1110 viewBox inside its button. Anchor to the visible
// rack edge, rather than the wider button's empty margin.
export function replicaLeftEdge(box: ReplicaBox) {
  return box.x - Math.min(box.width, box.height * 780 / 1110) / 2 - 2;
}

// The long top-to-bottom link has its own outer lane. Use exactly the same
// geometry for the background wire and moving log/ACK packets in either direction.
export function replicaRoute(a: ReplicaBox, b: ReplicaBox) {
  const start = replicaLeftEdge(a), end = replicaLeftEdge(b);
  const bend = Math.abs(a.y - b.y) > 360 ? 68 : 36;
  return `M ${start} ${a.y} C ${start - bend} ${a.y}, ${end - bend} ${b.y}, ${end} ${b.y}`;
}
