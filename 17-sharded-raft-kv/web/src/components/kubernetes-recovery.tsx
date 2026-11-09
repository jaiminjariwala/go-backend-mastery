"use client";

import { useEffect, useRef, useState } from "react";
import type { Cluster, ReplicaRuntime } from "@/lib/types";

type Replacement = {
  name: string;
  oldUID: string;
  newUID: string;
  claim: string;
  sameClaim: boolean;
  at: string;
};

// Only show actual Kubernetes identities. A missing response means unavailable,
// not that a Pod was replaced. Replacement requires a changed observed Pod UID.
export function KubernetesRecovery({ cluster }: { cluster: Cluster | null }) {
  const previous = useRef(new Map<string, ReplicaRuntime>());
  const [replacements, setReplacements] = useState<Replacement[]>([]);
  useEffect(() => {
    if (cluster?.Deployment !== "kubernetes") return;
    for (const node of cluster.ReplicaRuntime || []) {
      if (!node.PodName || !node.PodUID) continue;
      const old = previous.current.get(node.PodName);
      if (old?.PodUID && old.PodUID !== node.PodUID) {
        const replacement = {
          name: node.PodName,
          oldUID: old.PodUID,
          newUID: node.PodUID,
          claim: node.VolumeClaim,
          sameClaim: !!node.VolumeClaim && old.VolumeClaim === node.VolumeClaim,
          at: new Date().toLocaleTimeString(),
        };
        setReplacements((items) => [replacement, ...items].slice(0, 8));
      }
      previous.current.set(node.PodName, node);
    }
  }, [cluster]);
  if (cluster?.Deployment !== "kubernetes") return null;
  return (
    <section
      className="mt-5 rounded-xl border border-gray-200 p-4"
      aria-label="Kubernetes recovery"
    >
      <h2 className="text-base font-semibold">Pods and persistent storage</h2>
      <p className="mt-2 text-xs leading-5 text-gray-500">
        Kubernetes replaces a Pod. Raft elects the leader. Keep this view open
        while replacing one Pod, then use Read row to verify the saved value.
      </p>
      <div className="mt-4 grid gap-3 sm:grid-cols-3">
        {cluster.Nodes.map((node, index) => {
          const runtime = cluster.ReplicaRuntime?.[index];
          return (
            <div
              key={node.ID}
              className="rounded-lg border border-gray-200 p-3 text-xs leading-5"
            >
              <p className="font-semibold">
                {runtime?.PodName || `Server ${node.ID + 1}`}
              </p>
              <p className={node.Up ? "text-brand" : "text-gray-500"}>
                {node.Up ? `${node.Role} · responding` : "No response observed"}
              </p>
              <p className="mt-2 break-all">
                Pod UID: {runtime?.PodUID || "not observed"}
              </p>
              <p className="mt-1 break-all">
                Volume claim: {runtime?.VolumeClaim || "not reported"}
              </p>
            </div>
          );
        })}
      </div>
      <details className="mt-3 text-xs text-gray-500">
        <summary>Replace one Pod from your terminal</summary>
        <code className="mt-2 block break-all">
          kubectl --kubeconfig /tmp/raft-lab-kubeconfig -n raft-lab delete pod{" "}
          {cluster.ReplicaRuntime?.find((n) => n.PodName)?.PodName ||
            "POD_NAME"}
        </code>
        <p className="mt-2">
          Replace only one voter at a time. This removes the Pod, not its PVC.
        </p>
      </details>
      {replacements.map((item) => (
        <div
          key={item.newUID}
          className="mt-3 rounded-lg bg-blue-50 p-3 text-xs leading-5"
        >
          <p className="font-semibold">
            {item.name} replaced · {item.at}
          </p>
          <p className="break-all">
            {item.oldUID} → {item.newUID}
          </p>
          <p>
            {item.sameClaim
              ? `Same volume claim reported: ${item.claim}`
              : "Volume claim continuity not confirmed."}
          </p>
          <p className="mt-1 text-gray-500">
            A matching claim name does not verify the volume’s UID or saved
            rows. Verify the value with Read row; the recovery smoke script also
            checks PVC identity.
          </p>
        </div>
      ))}
    </section>
  );
}
