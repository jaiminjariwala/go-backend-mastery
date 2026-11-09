"use client";
import type { ShardSummary } from "@/lib/types";

// Shard selection changes observations, never the key's ownership.
export function LiveShardMap({
  shards,
  selected,
  onSelect,
}: {
  shards: ShardSummary[];
  selected: number;
  onSelect: (id: number) => void;
}) {
  const current = shards.find((shard) => shard.ID === selected);
  return (
    <nav
      className="mt-5 flex flex-wrap items-center gap-3"
      aria-label="Select a shard"
    >
      <div className="flex gap-1 rounded-lg bg-gray-100 p-1">
        {shards.map((shard) => (
          <button
            key={shard.ID}
            type="button"
            aria-pressed={selected === shard.ID}
            onClick={() => onSelect(shard.ID)}
            className={`rounded-md px-4 py-2 text-sm transition-colors ${selected === shard.ID ? "bg-white font-medium text-brand shadow-sm" : "text-gray-500 hover:text-gray-900"}`}
          >
            Shard {shard.ID + 1}
          </button>
        ))}
      </div>
      {current && (
        <p className="text-xs text-gray-500">
          {current.Online}/3 servers running · {current.Connected}/3 not
          isolated
        </p>
      )}
      <p className="text-xs text-gray-400">
        Each shard has three copies of its rows.
      </p>
    </nav>
  );
}
