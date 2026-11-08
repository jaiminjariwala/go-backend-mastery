#!/usr/bin/env python3
"""Exercise the local lab's routing, retained PVC, WAL and deduplication.

Run after applying the manifests and port-forwarding raft-router to :8086.
Only the dedicated kind-raft-lab context and raft-lab namespace are accepted.
The test replaces one Pod at a time. It never deletes PVCs or local Go data.
"""
import argparse
import json
import subprocess
import time
import urllib.error
import urllib.request
import uuid

parser = argparse.ArgumentParser()
parser.add_argument("--kubeconfig", required=True)
parser.add_argument("--api", default="http://127.0.0.1:8086")
args = parser.parse_args()
if args.api not in ("http://127.0.0.1:8086", "http://localhost:8086"):
    parser.error("use the local raft-router port-forward on :8086")
kubectl = ["kubectl", "--kubeconfig", args.kubeconfig, "--context", "kind-raft-lab", "-n", "raft-lab"]


def kube(*command):
    return subprocess.check_output(kubectl + list(command), text=True)


def api(path, body=None):
    data = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(args.api + path, data=data,
                                     headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=6) as response:
        return json.load(response)


def eventually(label, operation, seconds=45):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        try:
            result = operation()
            if result:
                return result
        except urllib.error.HTTPError as error:
            if error.code not in (409, 503, 504):
                raise
        except (urllib.error.URLError, TimeoutError):
            pass
        time.sleep(0.25)
    raise AssertionError("timed out: " + label)


def ready_new_pod(name, old_uid):
    pod = json.loads(kube("get", "pod", name, "-o", "json"))
    conditions = pod.get("status", {}).get("conditions", [])
    return pod if pod["metadata"]["uid"] != old_uid and any(
        c["type"] == "Ready" and c["status"] == "True" for c in conditions) else None


run_id = str(uuid.uuid4())
receipts = []
commands = []
for key in ("a", "b"):
    command = {"Operation": "write", "Key": "kubernetes-check:" + key,
               "Value": "retained-" + run_id, "RequestID": run_id + "-" + key}
    commands.append(command)
    result = eventually("write both shards", lambda: api("/api/kv", command))
    assert result["Committed"]
    receipts.append(result)
assert {r["ShardID"] for r in receipts} == {0, 1}, "test must cover both shards"

evidence = {"RunID": run_id, "Checks": [], "Receipts": receipts}
for receipt, command in zip(receipts, commands):
    shard = receipt["ShardID"]
    state = api("/api/cluster?shard=" + str(shard))
    assert state["Deployment"] == "kubernetes" and state["Transport"] == "grpc"
    assert all(n["Up"] for n in state["Nodes"])
    leader = max((n for n in state["Nodes"] if n["Role"] == "leader"), key=lambda n: n["Term"])
    pod_name = f"shard-{shard+1}-{leader['ID']}"
    pod = json.loads(kube("get", "pod", pod_name, "-o", "json"))
    claim_name = "data-" + pod_name
    claim_before = json.loads(kube("get", "pvc", claim_name, "-o", "json"))

    # Compact before replacement too: retry receipts must survive snapshots,
    # not merely remain available in an untrimmed in-memory log.
    api("/api/cluster?shard=" + str(shard), {"Action": "snapshot", "ID": leader["ID"]})
    kube("delete", "pod", pod_name, "--wait=false")

    def replacement():
        try:
            return ready_new_pod(pod_name, pod["metadata"]["uid"])
        except subprocess.CalledProcessError:
            return None

    new_pod = eventually("replacement Pod ready", replacement)
    claim_after = json.loads(kube("get", "pvc", claim_name, "-o", "json"))
    assert claim_after["metadata"]["uid"] == claim_before["metadata"]["uid"]
    assert claim_after["spec"]["volumeName"] == claim_before["spec"]["volumeName"]

    def caught_up():
        current = api("/api/cluster?shard=" + str(shard))
        node = current["Nodes"][leader["ID"]]
        return current if node["Up"] and node["KV"].get(command["Key"]) == command["Value"] else None

    eventually("replacement recovered and caught up", caught_up)
    result = eventually("verified read after replacement", lambda: api("/api/kv", {
        "Operation": "read", "Key": command["Key"]}))
    assert result["Committed"] and result["Found"] and result["Value"] == command["Value"]
    replay = eventually("deduplicated receipt after replacement", lambda: api("/api/kv", command))
    assert replay["Replayed"] and replay["Index"] == receipt["Index"] and replay["Term"] == receipt["Term"]
    evidence["Checks"].append({"Pod": pod_name, "OldUID": pod["metadata"]["uid"],
        "NewUID": new_pod["metadata"]["uid"], "PVC": claim_name,
        "Volume": claim_after["spec"]["volumeName"], "ReadVerified": True, "RetryReceiptRetained": True})

print(json.dumps(evidence, indent=2))
