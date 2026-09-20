# Go Backend Mastery - My 100-Day Backend Engineering Journey

I'm learning backend engineering and distributed systems **in depth, in Go** - one day at a time.
Each day: learn the topic, **type the code myself** (no copy-paste), commit it under `day-NNN/`.

## How I'm using this repo

1. I'm following the [100-day plan](#-100-day-plan) in order - later phases build on earlier ones.
2. Each day has a folder `day-NNN-<slug>/` with a `NOTES.md` template. I fill it in as I learn.
3. Every phase ends with a **project**. No project = I haven't learned the topic.
4. I check off topics in the phase checklists below as I master them.
5. **Pace is flexible:** "100 days" is just the ordering. With free time, do 3-4 days' worth per day and finish in \~4 weeks. The checkboxes track real progress, not calendar days.

> My rule: for every topic, be able to **explain it in 2 minutes** and **describe its trade-offs**.

---

## Phase overview

| Phase | Days | Focus |
| --- | --- | --- |
| 1 | 1-12 | Go language fundamentals |
| 2 | 13-22 | Go concurrency deep dive |
| 3 | 23-26 | Go advanced: testing, profiling, GC |
| 4 | 27-34 | Networking & API design |
| 5 | 35-42 | SQL & relational databases |
| 6 | 43-48 | NoSQL, Redis & caching |
| 7 | 49-56 | System design fundamentals |
| 8 | 57-70 | Distributed systems core |
| 9 | 71-76 | Messaging & event-driven architecture |
| 10 | 77-82 | Docker |
| 11 | 83-92 | Kubernetes |
| 12 | 93-97 | Microservices, security, observability, CI/CD |
| 13 | 98-100 | System design interview drills |

---

## Phase 1 - Go language fundamentals (Days 1-12)

- [ ] Toolchain: `go` CLI, `go.mod`/`go.sum`, workspaces, `GOPATH` vs modules, versioning & semantic import versioning
- [ ] Variables, constants, `iota`, zero values, short declaration `:=`, shadowing pitfalls
- [ ] Basic types, type conversions, aliases vs defined types
- [ ] Control flow: `if` with init statement, `for` (all forms), `switch`/`select`, `goto` (and why not)
- [ ] Functions: multiple/named returns, variadic params, closures, function values
- [ ] Pointers: semantics, pointer receivers vs value receivers, when each is right
- [ ] Structs: fields, tags, embedding (composition over inheritance), promotion rules
- [ ] Arrays vs slices: slice header internals (`ptr/len/cap`), `append` growth strategy, slicing pitfalls, `copy`
- [ ] Maps: internals (buckets), iteration order randomness, comma-ok idiom, concurrent access danger
- [ ] Strings, runes, bytes, `utf8` package, string interning costs, `strings.Builder`
- [ ] Interfaces: implicit satisfaction, interface segregation, empty interface/`any`, type assertions & type switches, interface internals (itable), common stdlib interfaces (`error`, `io.Reader`, `fmt.Stringer`)
- [ ] Errors: sentinel errors, custom error types, wrapping with `%w`, `errors.Is`/`errors.As`/`errors.Join`, error handling philosophy, `panic`/`recover` (and when *not* to use them)
- [ ] `defer`: LIFO order, argument evaluation timing, defer in loops, performance cost
- [ ] Packages: visibility rules, `init()` order, import cycles, internal packages
- [ ] Generics: type parameters, constraints (`comparable`, `any`, custom), type inference, when generics hurt readability
- [ ] `io` package: `Reader`/`Writer`/`Closer`/`Seeker`, `io.Copy`, `bufio`, `io.Pipe`
- [ ] `encoding/json`: marshaling, struct tags, custom `MarshalJSON`, streaming with `Decoder`
- [ ] `time`: durations, timers vs tickers, monotonic clocks, time zones
- [ ] OS interaction: signals (`os/signal`), exit codes, environment, file I/O
- [ ] Build: build tags, cross-compilation (`GOOS`/`GOARCH`), `ldflags`
- [ ] `go vet`, `gofmt`/`gofumpt`, `staticcheck` basics
- [ ] **Project:** CLI tool (e.g. todo manager / log analyzer) with subcommands, flags, and tests

## Phase 2 - Go concurrency deep dive (Days 13-22)

- [ ] Goroutines: what they cost (\~2KB stacks), GMP scheduler model (G/M/P, work stealing), `GOMAXPROCS`
- [ ] Channels: unbuffered (synchronous handoff) vs buffered semantics, blocking rules, zero value (`nil` channel blocks forever)
- [ ] Channel idioms: signaling with `chan struct{}`, closing channels, `for range` over channels, who closes?
- [ ] `select`: multiplexing, `default` for non-blocking ops, timeouts with `time.After`, random selection among ready cases
- [ ] Pipelines: stages as goroutines, fan-out/fan-in, cancellation propagation, bounded parallelism (worker pools, semaphore pattern)
- [ ] `sync.Mutex` / `RWMutex`: when RWMutex helps and when it hurts, lock ordering & deadlocks
- [ ] `sync.WaitGroup` (add-before-go, no copying), `sync.Once`, `sync.Cond`, `sync.Pool`, `sync.Map`, `atomic` package
- [ ] `context`: cancellation, deadlines/timeouts, values (and why values are for request-scoped data only), `context.WithoutCancel`
- [ ] Go memory model: happens-before relationships, what "correctly synchronized" means
- [ ] Data races: `-race` detector, common race patterns (loop variable capture - pre/post Go 1.22, map concurrent access, slice sharing)
- [ ] Goroutine leaks: how to detect (pprof goroutine dump), common causes
- [ ] `errgroup`, `singleflight` (`golang.org/x/sync`) - the interview-favorite patterns
- [ ] Graceful shutdown: signal handling + `http.Server.Shutdown` + draining workers
- [ ] **Project:** concurrent web crawler with rate limiting, dedup, and graceful shutdown

## Phase 3 - Go advanced: testing, profiling, GC (Days 23-26)

- [ ] Table-driven tests, subtests, test helpers (`t.Helper()`), test fixtures, `testify` vs stdlib
- [ ] `httptest` for handlers, mocking interfaces, dependency injection in Go
- [ ] Fuzzing (`go test -fuzz`), examples as docs
- [ ] Benchmarks: `testing.B`, `b.N`, `benchstat`, avoiding compiler optimizations fooling you
- [ ] `pprof`: CPU, heap, goroutine, mutex/block profiles; `go tool pprof`, flame graphs
- [ ] `runtime/trace`, execution tracer for latency investigations
- [ ] Escape analysis (`go build -gcflags=-m`), reducing allocations, inlining
- [ ] GC: tri-color mark & sweep, write barriers, pacer, `GOGC`, `GOMEMLIMIT`, GC latency vs throughput
- [ ] `reflect` basics and costs; `unsafe` - what it is, why you (mostly) avoid it
- [ ] **Project:** in-memory key-value store with HTTP API, benchmarks + pprof profile showing an optimization you made

## Phase 4 - Networking & API design (Days 27-34)

- [ ] OSI vs TCP/IP model, TCP 3-way handshake/teardown, flow control, congestion control basics, UDP vs TCP
- [ ] Sockets in Go: `net` package, TCP echo server, chat server with goroutine-per-connection
- [ ] DNS: resolution flow, record types, TTL, DNS caching issues
- [ ] TLS: handshake (1.2 vs 1.3), certificates, SNI, mTLS concept
- [ ] HTTP/1.1: methods, status codes, headers, cookies, keep-alive, chunked encoding
- [ ] HTTP/2: multiplexing, HPACK, server push (deprecated), binary framing
- [ ] HTTP/3 & QUIC: why (head-of-line blocking), 0-RTT
- [ ] `net/http` server internals: `ServeMux` (1.22+ routing), `Handler`/`HandlerFunc`, request lifecycle
- [ ] Middleware pattern: logging, auth, recovery, CORS, request IDs, timeouts
- [ ] REST: resource naming, maturity model, idempotency (keys), pagination (offset vs cursor), filtering/sorting, versioning strategies, OpenAPI/Swagger
- [ ] gRPC: Protocol Buffers, unary/server-streaming/client-streaming/bidi, deadlines, interceptors, `grpc-gateway`
- [ ] WebSockets, Server-Sent Events, long polling - when to use each
- [ ] Load balancing algorithms: round-robin, weighted, least-connections, IP hash, consistent hashing
- [ ] Reverse proxy (`httputil.ReverseProxy`), forward proxy, CDN behavior
- [ ] API gateways & BFF pattern
- [ ] **Projects:** (1) REST API in **stdlib only** - no frameworks; (2) gRPC service with streaming; (3) L7 load balancer in Go

## Phase 5 - SQL & relational databases (Days 35-42)

- [ ] Data modeling, normalization (1NF → BCNF), when to denormalize
- [ ] Postgres setup (in Docker); `psql` essentials
- [ ] Indexes: B-tree internals, composite index column order, covering indexes, partial indexes, index-only scans
- [ ] `EXPLAIN` / `EXPLAIN ANALYZE`: reading query plans
- [ ] ACID, transactions, isolation levels (read committed / repeatable read / serializable), anomalies (dirty read, non-repeatable read, phantom)
- [ ] MVCC in Postgres, `VACUUM`, row locking (`FOR UPDATE`), deadlocks, advisory locks
- [ ] Joins (all types), window functions, CTEs (incl. recursive)
- [ ] Query optimization: N+1 problem, `IN` vs joins, pagination with keyset
- [ ] Connection pooling in Go (`pgx` pool), prepared statements
- [ ] Migrations (`golang-migrate`), schema versioning discipline
- [ ] Replication: streaming/logical, read replicas, replication lag handling
- [ ] Partitioning & sharding strategies (range/hash), `pgBouncer`
- [ ] **Project:** URL shortener backend on Postgres - schema, migrations, indexes, transactional click counting

## Phase 6 - NoSQL, Redis & caching (Days 43-48)

- [ ] Redis data structures: strings, hashes, lists, sets, sorted sets, streams, bitmaps, HyperLogLog, geospatial
- [ ] TTL, eviction policies (LRU/LFU variants), persistence (RDB/AOF)
- [ ] Transactions (`MULTI`/`EXEC`), Lua scripts, pipelines
- [ ] Pub/Sub vs Streams (consumer groups, PEL, claiming)
- [ ] Redis Cluster (hash slots), Sentinel, failover behavior
- [ ] Distributed locks: Redlock algorithm **and** the Martin Kleppmann critique, fencing tokens done right
- [ ] Caching patterns: cache-aside, read-through, write-through, write-behind, refresh-ahead
- [ ] Cache stampede / thundering herd: `singleflight`, request coalescing, probabilistic early refresh
- [ ] Invalidation strategies, TTL jitter, negative caching
- [ ] MongoDB: documents, indexes, aggregation pipeline basics
- [ ] Wide-column (Cassandra/DynamoDB model): partition keys, clustering keys, tunable consistency, hinted handoff
- [ ] Elasticsearch basics (inverted index) - for search-type interview problems
- [ ] **Projects:** (1) rate limiter service on Redis; (2) distributed cache with consistent hashing in Go

## Phase 7 - System design fundamentals (Days 49-56)

- [ ] Back-of-envelope estimation: QPS, storage, bandwidth math; powers of 10 & latency numbers every engineer should know
- [ ] CAP theorem - what it actually says (and common misquotes); PACELC
- [ ] Consistency models: strong, eventual, causal, read-your-writes, monotonic reads, session consistency; linearizability vs serializability
- [ ] Availability math: "nines", error budgets; latency percentiles p50/p95/p99/p999 and tail latency causes
- [ ] Caching at every layer; CDN push vs pull
- [ ] Reliability patterns: timeouts, retries with exponential backoff + jitter, hedging, deadlines
- [ ] Circuit breaker, bulkhead, load shedding, backpressure
- [ ] Rate limiting: token bucket, leaky bucket, fixed window, sliding window log/counter (implement all in Go)
- [ ] Load balancers (L4 vs L7), API gateways
- [ ] DB scaling: read replicas, sharding, federation; choosing partition keys
- [ ] Async architectures: queues, workers, schedulers
- [ ] SLOs/SLIs/SLAs; RED and USE observability methods
- [ ] **Drills:** estimate capacity for TinyURL, WhatsApp, YouTube (numbers on paper)

## Phase 8 - Distributed systems core (Days 57-70)

- [ ] The 8 fallacies of distributed computing; partial failure as the norm
- [ ] Time: Lamport timestamps, vector clocks, Hybrid Logical Clocks; Google Spanner/TrueTime concept
- [ ] Ordering: total vs causal vs FIFO; idempotency as the practical answer
- [ ] Quorums: W + R > N, sloppy quorums, read repair, anti-entropy (Merkle trees)
- [ ] Replication topologies: single-leader, multi-leader, leaderless (Dynamo-style)
- [ ] Partitioning: range vs hash, consistent hashing with virtual nodes (implement the ring)
- [ ] Consensus: Paxos intuition (proposers/acceptors/learners), why it's hard
- [ ] Raft: leader election, log replication, safety guarantees - **implement a minimal Raft**
- [ ] Zab (ZooKeeper's protocol) awareness
- [ ] Gossip protocols: SWIM membership & failure detection, rumor mongering (implement simple gossip)
- [ ] Leader election: Bully algorithm, Raft-based election
- [ ] Distributed locks done right: fencing tokens; why "Redis Redlock" is debated
- [ ] Distributed transactions: 2PC (and why it blocks), 3PC, Saga (choreography vs orchestration - implement one), TCC
- [ ] Transactional outbox pattern + idempotent consumers (implement with Postgres)
- [ ] CRDTs: G-Counter, PN-Counter, LWW-Register, OR-Set (implement)
- [ ] Service discovery: client-side vs server-side, etcd/Consul/ZooKeeper roles
- [ ] Distributed tracing: OpenTelemetry, spans/traces, sampling strategies, correlation IDs
- [ ] Byzantine fault tolerance basics (PBFT awareness); Jepsen-style thinking
- [ ] **Capstone:** 3-node replicated key-value store with leader election + log replication in Go

## Phase 9 - Messaging & event-driven architecture (Days 71-76)

- [ ] Kafka: topics, partitions, offsets, consumer groups, rebalancing
- [ ] Kafka internals: ISR, replication, log segments, exactly-once semantics (idempotent producer + transactions), log compaction
- [ ] Partitioning strategies & ordering guarantees
- [ ] RabbitMQ: exchanges (direct/fanout/topic/headers), queues, bindings, ACKs, prefetch, dead-letter exchanges
- [ ] NATS core + JetStream: subjects, streams, consumers
- [ ] Queue vs pub/sub, push vs pull models
- [ ] Event-driven patterns: event notification, event-carried state transfer, event sourcing, CQRS (implement a small event-sourced aggregate)
- [ ] Outbox + CDC (Debezium concept), poison messages & DLQ, message deduplication
- [ ] **Projects:** (1) order service with Kafka in Go; (2) notification fan-out service (email/SMS/push) with retries & DLQ

## Phase 10 - Docker (Days 77-82)

- [ ] Images & layers, Dockerfile instructions, BuildKit, layer caching, `.dockerignore`
- [ ] Multi-stage builds; distroless & scratch images for Go (static binaries with `CGO_ENABLED=0`)
- [ ] How containers work: namespaces, cgroups - isolation is not virtualization
- [ ] Volumes, bind mounts, storage drivers
- [ ] Networking: bridge/host/none/overlay, port mapping, embedded DNS
- [ ] Docker Compose: full local stack (Go API + Postgres + Redis + Kafka)
- [ ] Registries, tagging strategy, image scanning (Trivy)
- [ ] Resource limits (`--memory`, `--cpus`), healthchecks, restart policies
- [ ] Security: non-root users, read-only filesystems, no secrets in layers, secret mounts
- [ ] **Project:** production-grade container setup for your Phase 4 API - hardened, scanned, Compose-orchestrated

## Phase 11 - Kubernetes (Days 83-92)

- [ ] Architecture: API server, scheduler, controller manager, etcd; kubelet, kube-proxy, CNI/CRI; `kind` local cluster
- [ ] Pods: lifecycle, labels/selectors/annotations, namespaces, init containers, multi-container patterns (sidecar/adapter/ambassador)
- [ ] Deployments & ReplicaSets; StatefulSets, DaemonSets, Jobs, CronJobs
- [ ] Services: ClusterIP/NodePort/LoadBalancer, headless services, Endpoints/EndpointSlices
- [ ] Ingress & IngressClass; Gateway API awareness
- [ ] ConfigMaps, Secrets, downward API, env vs mounted config
- [ ] Probes: liveness/readiness/startup; resource requests/limits & QoS classes
- [ ] Autoscaling: HPA, VPA, Cluster Autoscaler, KEDA awareness
- [ ] Storage: PV/PVC, StorageClass, CSI; run Postgres on K8s
- [ ] RBAC: Roles/ClusterRoles, bindings, ServiceAccounts
- [ ] NetworkPolicies (default-deny posture)
- [ ] Deployment strategies: rolling (maxSurge/maxUnavailable), rollbacks, blue-green, canary (with Istio/Argo Rollouts awareness)
- [ ] Helm: charts, values, templating - package your Go app
- [ ] Operators: controller pattern, CRDs, reconciliation loops (`kubebuilder`/`operator-sdk` awareness)
- [ ] Service mesh: Istio/Linkerd - sidecars, mTLS, traffic splitting, observability
- [ ] etcd deep dive (Raft in production); kube-scheduler basics
- [ ] `kubectl` essentials, `k9s`; GitOps preview (ArgoCD)
- [ ] **Project:** deploy your microservice stack to K8s - manifests + Helm chart + HPA + Ingress + NetworkPolicies

## Phase 12 - Microservices, security, observability, CI/CD (Days 93-97)

- [ ] Decomposition: DDD bounded contexts, strangler fig pattern, database-per-service vs shared
- [ ] Resilience in Go: `sony/gobreaker`, `singleflight`, timeouts via context - wire into a service
- [ ] AuthN/AuthZ: OAuth2 flows (authorization code, client credentials), OIDC, JWT structure/validation/rotation/revocation, sessions vs tokens, API keys, mTLS between services
- [ ] OWASP API Top 10, input validation, SQL injection prevention, secrets management (Vault/SOPS/ESO awareness)
- [ ] Observability in Go: structured logging (`log/slog`, `zerolog`), Prometheus metrics (counters/gauges/histograms), OpenTelemetry tracing - instrument a service end to end
- [ ] Grafana dashboards, Alertmanager, log aggregation (Loki/ELK concept)
- [ ] SRE: SLOs & error budgets, incident response, blameless postmortems, chaos engineering (Netflix-style)
- [ ] CI/CD: GitHub Actions pipelines (test → build → scan → push → deploy), semantic versioning, trunk-based development, feature flags
- [ ] IaC & GitOps: Terraform basics, ArgoCD
- [ ] 12-factor app methodology
- [ ] **Project:** production-readiness checklist applied to one service - auth, metrics, traces, dashboards, alerts, pipeline

## Phase 13 - System design interview drills (Days 98-100)

Classic FAANG system design problems - I'll do each **timed (35-45 min)**, out loud, with trade-offs:

- [ ] URL shortener (TinyURL)
- [ ] Rate limiter (distributed)
- [ ] Pastebin
- [ ] Twitter / Instagram feed & stories
- [ ] WhatsApp / Messenger chat system
- [ ] YouTube / Netflix video upload & streaming
- [ ] Uber/Lyft: location tracking, matching, geospatial indexing (geohash, quadtrees)
- [ ] Dropbox/Google Drive: file sync, chunking, delta sync
- [ ] Web crawler (distributed)
- [ ] Notification system (multi-channel)
- [ ] Search autocomplete / typeahead (tries)
- [ ] Distributed cache (design Memcached/Redis)
- [ ] Distributed lock service
- [ ] Unique ID generator (Snowflake-style)
- [ ] Metrics & monitoring pipeline
- [ ] Ad click aggregator
- [ ] Distributed key-value store (Dynamo-style)
- [ ] Distributed file system (GFS/HDFS concepts)
- [ ] Consensus/etcd-like service
- [ ] ML model/inference serving platform (NVIDIA-relevant: batching, GPU scheduling, Triton concepts)

**System design interview framework:** clarify requirements → back-of-envelope → high-level design → deep dive (pick 2-3 components) → bottlenecks & failure modes → wrap up trade-offs.

---

## 🗓 100-day plan

| Day | Focus | Build / Do |
| --- | --- | --- |
| 1 | Go setup, modules, toolchain | Hello world CLI + `go vet` clean |
| 2 | Types, zero values, constants, iota | Exercises: iota enum, shadowing traps |
| 3 | Functions, closures, variadic | Higher-order function utilities |
| 4 | Pointers, structs, methods, embedding | Shape hierarchy via embedding |
| 5 | Slices internals | Implement `Filter`/`Map` without `append` bugs |
| 6 | Maps, strings, runes, utf8 | Word-frequency counter |
| 7 | Interfaces | `io.Writer` implementations; stringer |
| 8 | Errors: wrap, Is/As | Error package with domain errors |
| 9 | Generics | Generic stack, map/filter with constraints |
| 10 | Packages, init, defer deep dive | Multi-package project layout |
| 11 | io, encoding/json, time | JSON log parser CLI |
| 12 | **Project:** CLI tool | Todo CLI with tests |
| 13 | Goroutines + GMP model | Goroutine fan-out benchmark |
| 14 | Channels: buffered vs unbuffered | Blocking-behavior experiments |
| 15 | Channel idioms: close, range, nil | Pipeline stage helpers |
| 16 | select + timeouts | Timeout/cancel patterns |
| 17 | Pipelines, fan-out/fan-in | Parallel image-resize-style pipeline |
| 18 | Mutex/RWMutex, WaitGroup, Once | Concurrent counter variants |
| 19 | sync.Map, Pool, atomics | Object pool benchmark |
| 20 | context | Request-scoped timeout propagation |
| 21 | Memory model + race detector | Find & fix races with `-race` |
| 22 | **Project:** concurrent web crawler | Crawler with dedup + graceful shutdown |
| 23 | Testing: table-driven, fuzzing | Fuzz a parser you wrote |
| 24 | Benchmarks + pprof | Profile & optimize the KV store |
| 25 | GC internals, escape analysis | Reduce allocations in hot path |
| 26 | **Project:** in-memory KV + HTTP API | With benchmarks |
| 27 | TCP/IP, sockets | TCP echo + chat server |
| 28 | DNS, TLS, HTTP/1.1 | Inspect real handshakes (Wireshark/`openssl s_client`) |
| 29 | REST with stdlib only | CRUD API, no frameworks |
| 30 | Middleware | Logging/auth/recovery/request-ID chain |
| 31 | API design: idempotency, pagination, OpenAPI | Idempotency-key middleware |
| 32 | gRPC + protobuf | Streaming gRPC service |
| 33 | WebSockets + SSE | Realtime feed endpoint |
| 34 | **Project:** L7 load balancer | Round-robin, least-conn, consistent hash |
| 35 | Data modeling, normalization | Schema for URL shortener |
| 36 | Indexes + EXPLAIN ANALYZE | Index experiments on 1M rows |
| 37 | Transactions + isolation levels | Money-transfer with tx |
| 38 | MVCC, locking, deadlocks | Reproduce & resolve a deadlock |
| 39 | Joins, window functions, CTEs | Analytics queries |
| 40 | Pooling (pgx), N+1 | Fix N+1 in an API |
| 41 | Migrations | golang-migrate workflow |
| 42 | **Project:** URL shortener on Postgres | Full backend |
| 43 | Redis data structures | Rebuild Phase-6 mini apps |
| 44 | Redis Streams, Lua, pipelines | Job queue on Streams |
| 45 | Caching patterns + singleflight | Cache-aside with stampede protection |
| 46 | Cluster/Sentinel, eviction | Failover experiment |
| 47 | MongoDB / Cassandra data models | Model a feed in each |
| 48 | **Project:** distributed cache | Consistent-hashing nodes |
| 49 | Estimation math + latency numbers | Estimate TinyURL/WhatsApp/YouTube |
| 50 | CAP/PACELC, consistency models | 2-minute explanations, recorded |
| 51 | Caching layers, CDN | CDN behavior notes |
| 52 | Retries/backoff/jitter, circuit breaker | `gobreaker`-style breaker from scratch |
| 53 | Rate limiting algorithms | All 4 algorithms in Go |
| 54 | LBs, API gateways | Compare L4 vs L7 |
| 55 | DB scaling: replicas, sharding | Shard-key design exercises |
| 56 | Async, SLOs, p99 | Define SLIs for your API |
| 57 | Fallacies, Lamport/vector clocks | Implement vector clocks |
| 58 | Ordering, idempotency | Idempotent worker design |
| 59 | Quorums W+R>N | Quorum KV prototype |
| 60 | Replication topologies | Compare leader-follower vs leaderless |
| 61 | Consistent hashing | Ring implementation + tests |
| 62 | Paxos intuition | Write the 2-minute explanation |
| 63 | Raft: leader election | Implement election |
| 64 | Raft: log replication | Replicate a log |
| 65 | Gossip/SWIM | Membership prototype |
| 66 | Distributed locks + fencing | Fencing-token lock service |
| 67 | 2PC vs Saga | Saga orchestrator |
| 68 | Outbox + idempotent consumers | Outbox with Postgres |
| 69 | CRDTs | G-Counter, OR-Set |
| 70 | **Capstone:** 3-node replicated KV | Raft-lite in Go |
| 71 | Kafka: topics, partitions, groups | Local Kafka via Docker |
| 72 | Kafka: ISR, EOS semantics | Idempotent producer demo |
| 73 | Event-driven order service | Go + Kafka |
| 74 | RabbitMQ: exchanges, DLX | Routing topologies |
| 75 | NATS JetStream; event sourcing + CQRS | Event-sourced aggregate |
| 76 | **Project:** notification fan-out | Retries + DLQ |
| 77 | Dockerfile, layers, multi-stage | Distroless Go image |
| 78 | Volumes, networking | Multi-network experiments |
| 79 | Docker Compose | Full stack: Go + PG + Redis + Kafka |
| 80 | Registries, scanning | Trivy scan your images |
| 81 | Namespaces/cgroups | Inspect isolation primitives |
| 82 | **Project:** hardened container setup | Non-root, read-only FS, scanned |
| 83 | K8s architecture; kind cluster | Cluster up, deploy nginx |
| 84 | Pods, labels, namespaces | Deploy your Go API |
| 85 | Deployments, rolling updates | Rollout + rollback drill |
| 86 | Services, Ingress | Expose API via Ingress |
| 87 | ConfigMaps, Secrets | 12-factor config |
| 88 | Probes, resources, HPA | Load-test autoscaling |
| 89 | StatefulSets, PV/PVC | Postgres on K8s |
| 90 | RBAC, NetworkPolicies | Default-deny policy |
| 91 | Helm | Chart for your app |
| 92 | **Project:** full stack on K8s | Manifests + Helm + HPA + Ingress |
| 93 | Microservices decomposition | DDD context map of a shop |
| 94 | Auth: OAuth2/OIDC, JWT in Go | Auth service issuing JWTs |
| 95 | mTLS, OWASP API Top 10 | Threat-model your API |
| 96 | Observability: slog, Prometheus, OTel | Instrumented service |
| 97 | CI/CD: GitHub Actions, ArgoCD | Pipeline: test → scan → deploy |
| 98 | Drill: URL shortener + rate limiter | Timed, out loud |
| 99 | Drill: Twitter feed + chat system | Timed, out loud |
| 100 | Drill: distributed KV + STAR stories | Polish & retrospective |

---

## 🏗 Capstone projects

1. **Concurrent web crawler** - worker pools, politeness, dedup, graceful shutdown
2. **URL shortener** - Postgres, Redis cache, base62, analytics
3. **Distributed cache** - consistent hashing, replication, failure handling
4. **Raft-lite KV store** - 3-node consensus, leader election, log replication
5. **Event-driven order system** - Kafka, outbox pattern, saga, notification fan-out
6. **L7 load balancer** - health checks, multiple algorithms, observability
7. **K8s-deployed microservice** - Helm chart, HPA, mTLS, dashboards, CI/CD

---

## 📌 Backlog - topics to add later

When I find a topic that's missing, I add it here (or under the right phase above) as a checkbox and work through it. This roadmap grows with me.

- [ ] *(example) eBPF basics for observability*
- [ ] *add yours here...*