// Preview only: replicas independently enforce the same persisted routing rule.
// Math.imul preserves the 32-bit multiplication used by Go's FNV-1a implementation.
export function shardOwner(key: string, count: number) {
  let hash = 2166136261;
  for (const byte of new TextEncoder().encode(key))
    hash = Math.imul(hash ^ byte, 16777619) >>> 0;
  return hash % count;
}
