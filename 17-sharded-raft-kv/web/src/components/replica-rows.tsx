"use client";

import { motion, useReducedMotion } from "motion/react";
import { useEffect, useRef, useState } from "react";

interface Props {
  values: Record<string, string>;
  online?: boolean;
  busy?: boolean;
  onEdit?: (key: string, value: string) => void;
  label: string;
  applied?: boolean;
  shardNumber?: number;
  playbackLabel?: string;
}

// These are current applied rows, not a history of the last request.
// Compare values across observations so a refresh never highlights old rows as new.
export function ReplicaRows({
  values,
  online = true,
  busy,
  onEdit,
  label,
  applied = true,
  shardNumber,
  playbackLabel,
}: Props) {
  const reduce = useReducedMotion();
  const previous = useRef(values);
  const scroller = useRef<HTMLDivElement>(null);
  const clearHighlight = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [changedKeys, setChangedKeys] = useState<Set<string>>(new Set());
  const all = Object.entries(values).sort(([a], [b]) => a.localeCompare(b));
  const rows = online ? all : [];
  useEffect(() => {
    if (!online) return;
    const changed = Object.keys(values).filter((key) => previous.current[key] !== values[key]);
    previous.current = values;
    if (!changed.length) return;
    setChangedKeys(new Set(changed));
    if (clearHighlight.current) clearTimeout(clearHighlight.current);
    clearHighlight.current = setTimeout(() => setChangedKeys(new Set()), 2600);
    // Bring the actual changed row into view inside its table, not the whole page.
    const container = scroller.current;
    const row = container && Array.from(container.querySelectorAll<HTMLElement>("[data-row-key]"))
      .find((element) => element.dataset.rowKey === changed[0]);
    if (container && row) {
      const bounds = container.getBoundingClientRect();
      const cell = row.getBoundingClientRect();
      if (cell.bottom > bounds.bottom || cell.top < bounds.top + 30)
        container.scrollTop += cell.top - bounds.top - 30;
    }
  }, [values, online]);
  useEffect(() => () => {
    if (clearHighlight.current) clearTimeout(clearHighlight.current);
  }, []);
  return (
    <div className="reference-table live-replica-table min-w-0 text-left" aria-label={label}>
      <h3 className="reference-table-title py-2 text-center text-xs font-medium text-gray-600">
        USERS TABLE
      </h3>
      <p className="pb-2 text-center text-ui-11 text-gray-400">
        {playbackLabel || (shardNumber ? `Shard ${shardNumber} · stored rows` : "Stored rows")}
      </p>
      <p className="sr-only">
        {online
          ? `${all.length} ${applied ? "applied " : ""}row${all.length === 1 ? "" : "s"}`
          : "Offline · current rows unavailable"}
      </p>
      <div ref={scroller} className="reference-table-grid max-h-32 overflow-auto" tabIndex={0} aria-label={`${label}, scroll to see all rows`}>
        <table className="w-full table-fixed text-sm">
          <thead className="sticky top-0 z-10 bg-gray-100 text-gray-600">
            <tr>
              <th className="px-3 py-1.5 font-normal">KEY</th>
              <th className="px-2 py-1.5 font-normal">VALUE</th>
            </tr>
          </thead>
          <tbody>
              {rows.map(([key, value]) => (
                <motion.tr
                  key={key}
                  data-row-key={key}
                  initial={false}
                  animate={{ backgroundColor: changedKeys.has(key) ? "#eaf2ff" : "#ffffff" }}
                  exit={{ opacity: 0, backgroundColor: "#fee2e2" }}
                  transition={{ duration: reduce ? 0 : 0.6 }}
                >
                  <td className="border-t border-gray-50 px-3 py-1.5">
                    <span className="block overflow-x-auto whitespace-nowrap ">
                      {onEdit ? (
                        <button
                          className="text-brand underline-offset-2 hover:underline"
                          disabled={busy}
                          onClick={() => onEdit(key, value)}
                          title="Load this row into the editor"
                        >
                          {key}
                        </button>
                      ) : (
                        key
                      )}
                    </span>
                  </td>
                  <td className="border-t border-gray-50 px-2 py-1.5">
                    <motion.span
                      key={value}
                      initial={false}
                      animate={{ opacity: 1 }}
                      transition={{ duration: reduce ? 0 : 0.9 }}
                      className="block overflow-x-auto whitespace-nowrap "
                    >
                      <span
                        className="mr-2 inline-block size-3 rounded-full"
                        style={{
                          background: [
                            "#9a83e7",
                            "#5078ed",
                            "#9a83e7",
                            "#56b77a",
                            "#1caaa8",
                          ][
                            Math.max(
                              0,
                              Number(key.match(/\d+$/)?.[0] || 101) - 101,
                            ) % 5
                          ],
                        }}
                      />
                      {value}
                    </motion.span>
                  </td>
                </motion.tr>
              ))}
            {!rows.length && (
              <tr><td colSpan={2} className="px-3 py-6 text-center text-xs text-gray-400">
                {online ? "No rows stored yet" : "Server offline"}
              </td></tr>
            )}
          </tbody>
        </table>
      </div>
      <p className="px-1 pt-2 text-xs text-gray-400">
        {online ? `${all.length} stored row${all.length === 1 ? "" : "s"}` : "Current rows unavailable"}
      </p>
    </div>
  );
}
