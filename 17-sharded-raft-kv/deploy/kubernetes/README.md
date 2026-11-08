# Local Kubernetes lab

This is a runnable deployment of the same Go Raft engine. It uses two fixed
three-voter groups, six StatefulSet Pods with separate persistent volumes,
a stateless router, and the existing Next.js UI. It does not change membership
or move shards. The default localhost process mode remains available.

## What each part does

- `shard-1.yaml` and `shard-2.yaml`: stable voter names, per-Pod PVCs, probes,
  and a disruption budget allowing one voluntary eviction per group.
- `peers.yaml`: immutable fixed peer DNS configuration. Array positions are
  voter IDs. DNS names identify individual Pods, not a load-balanced peer service.
- `gateway.yaml`: router and UI Deployments and ClusterIP Services. The router
  starts no children and owns no replica storage.
- `network.yaml`: namespace-local traffic policy. Enforcement requires a CNI
  that supports NetworkPolicy; the default kind CNI may not enforce it.
- `../../deployment.go`: configuration validation, persisted PVC/voter binding,
  Pod ordinal parsing, and externally managed router mode.
- `smoke.py`: real write, snapshot, leader Pod replacement, verified read,
  retained volume and duplicate-request checks across both shards.

A replacement Pod gets a new UID and process but the same voter name and PVC.
The WAL and snapshots recover local state; Raft catches up any missing changes.
A PVC containing another voter/shard identity is rejected. Existing local lab
WAL directories are never imported automatically into these volumes.

## Run with Docker Desktop and kind

From the repository root, with Docker Desktop running:

```sh
mkdir -p /tmp/raft-kube-bin
GOBIN=/tmp/raft-kube-bin go install sigs.k8s.io/kind@v0.33.0
export RAFT_KUBECONFIG=/tmp/raft-lab-kubeconfig
/tmp/raft-kube-bin/kind create cluster --name raft-lab --kubeconfig "$RAFT_KUBECONFIG" --wait 90s

docker build -t raft-kv:lab -f 17-sharded-raft-kv/deploy/Dockerfile .
docker build -t raft-web:lab 17-sharded-raft-kv/web
/tmp/raft-kube-bin/kind load docker-image raft-kv:lab raft-web:lab --name raft-lab

kubectl --kubeconfig "$RAFT_KUBECONFIG" apply -k 17-sharded-raft-kv/deploy/kubernetes
kubectl --kubeconfig "$RAFT_KUBECONFIG" -n raft-lab get pods,pvc
```

Use a separate terminal for each port-forward:

```sh
kubectl --kubeconfig "$RAFT_KUBECONFIG" -n raft-lab port-forward --address 127.0.0.1 service/raft-web 8085:3000
```

```sh
kubectl --kubeconfig "$RAFT_KUBECONFIG" -n raft-lab port-forward --address 127.0.0.1 service/raft-router 8086:8084
```

Open <http://127.0.0.1:8085/> for the live cluster. The Concepts tab includes
gRPC and Kubernetes diagrams. The lab reports `Deployment: kubernetes`.
Local mode still uses ports 8083/8084 and its separate data directories.
The Next.js image bakes `RAFT_API_ORIGIN=http://raft-router:8084` into its API
rewrite. Local builds use the default loopback origin.

The kubeconfig above is separate from your default context. Reuse the existing
`raft-lab` cluster on later runs; creating a second cluster with the same name
is unnecessary. Images use `IfNotPresent`; after rebuilding the same tag,
reload it into kind, restart the stateless Deployments, and deliberately
replace replica Pods one at a time, checking recovery between replacements.
The StatefulSets use `OnDelete` updates to avoid automatic rolling replacements.

## Verify actual recovery

With both port-forwards running:

```sh
python3 17-sharded-raft-kv/deploy/kubernetes/smoke.py --kubeconfig "$RAFT_KUBECONFIG"
```

The script writes two test rows, one per shard, and takes a checkpoint. It
replaces the observed leader Pod in each group, one at a time. It checks that
the Pod UID changes while its PVC UID and bound volume stay unchanged, waits
for recovered rows, performs a verified read, and retries the original ID.
The retry must return the original committed log index and term.
It accepts only the dedicated `kind-raft-lab` context and `raft-lab` namespace.
Its two `kubernetes-check:*` rows remain available to inspect afterward.

`validation.json` records a successful local run on November 5, 2026. Both
shards passed saved-row and retry-receipt checks after leader Pod replacement.
All six PVCs were bound and all eight application Pods were Ready afterward.
This is evidence for that recovery scenario, not a correctness proof or a
performance benchmark.

## Health and storage semantics

`/healthz` checks process responsiveness. `/readyz` checks local storage health.
Neither depends on being leader or on having quorum. Quorum loss must not
cause all healthy voters to restart at once. Headless Services publish
not-ready addresses to let peers find one another during bootstrap. Readiness
is not a certificate that a client write has committed.

PVC retention is `Retain` on StatefulSet deletion and scale-down. Deleting a
Pod retains its claim. Deleting a PVC or the entire kind cluster can destroy
this lab's stored data. A retained PVC is persistence, not an independent backup.

The UI disables process Stop/Restart in Kubernetes mode. Use `kubectl` to
replace Pods; partition/heal and snapshot controls remain available. Changing
a StatefulSet replica count is not a safe Raft membership change. Ordinals
above two are rejected, and the fixed peer configuration remains unchanged.

## Deliberate limits

This local lab uses one kind machine. It demonstrates Pod replacement and
persistent-volume recovery, not availability after losing that whole machine.
Volumes use the cluster's default StorageClass. Routing state and peer identity
are fixed; DNS allows replacement Pod IPs to change without changing voters.

Services are ClusterIP only, with localhost port-forwarding. RPCs are cleartext
and administrative endpoints are unauthenticated. Use this on a trusted local
cluster. A remote/public deployment needs authenticated TLS, access control,
and a storage/failure model appropriate to that environment. There is no
operator, Helm chart, autoscaler, automatic shard balancing, or membership API.

References: [StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/),
[probes](https://kubernetes.io/docs/concepts/workloads/pods/probes/), and
[kind quick start](https://kind.sigs.k8s.io/docs/user/quick-start/).
