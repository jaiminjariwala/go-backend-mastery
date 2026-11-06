// Reuse one rack illustration everywhere. The view box trims only the empty
// transparent margin, keeping the entire cabinet visible at every size.
export function ReplicaArtwork({ compact = false }: { compact?: boolean }) {
  return (
    <svg
      viewBox="270 40 780 1110"
      className="size-full"
      role="img"
      aria-label={compact ? "Server replica" : "Rack server"}
    >
      <image href="/server-rack.png" width="1312" height="1199" />
    </svg>
  );
}
