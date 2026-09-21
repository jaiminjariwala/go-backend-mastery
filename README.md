# Go Backend Mastery

I'm learning backend engineering and distributed systems **in depth, in Go** - one day at a time.
Each day: learn the topic, **type the code myself** (no copy-paste), commit it under `day-NNN/`.

## How I'm using this repo

1. I'm following the [63-day plan](#-63-day-plan) in order - later phases build on earlier ones.
2. Each day has a folder `day-NNN-<slug>/`. I fill it in as I learn.
3. Every phase ends with a **project**.
4. **Pace is flexible:** "63 days" is just the ordering. With free time, do 3-4 days' worth per day and finish in ~3 weeks. The checkboxes track real progress, not calendar days.

---



## Phase overview


| Phase | Days  | Focus                                       |
| ----- | ----- | ------------------------------------------- |
| 1     | 1-10  | Go language fundamentals                    |
| 2     | 11-18 | Go concurrency deep dive                    |
| 3     | 19-21 | Go applied: testing, profiling              |
| 4     | 22-27 | Networking & API design                     |
| 5     | 28-33 | SQL & relational databases                  |
| 6     | 34-36 | NoSQL, Redis & caching                      |
| 7     | 37-41 | System design fundamentals                  |
| 8     | 42-47 | Distributed systems core                    |
| 9     | 48-49 | Messaging & event-driven architecture       |
| 10    | 50-51 | Docker                                      |
| 11    | 52-57 | Kubernetes                                  |
| 12    | 58-60 | Microservices, security, observability, CI/CD |
| 13    | 61-63 | System design interview drills              |


---



## Phase 1 - Go language fundamentals (Days 1-10)

- [ ] Toolchain: `go` CLI, `go.mod`/`go.sum`, modules, versioning
- [ ] Variables, constants, `iota`, zero values, short declaration `:=`, shadowing pitfalls
- [ ] Basic types, type conversions, aliases vs defined types
- [ ] Control flow: `if` with init statement, `for` (all forms), `switch`
- [ ] Functions: multiple/named returns, variadic params, closures, function values
- [ ] Pointers: semantics, pointer receivers vs value receivers, when each is right
- [ ] Structs: fields, tags, embedding (composition over inheritance), promotion rules
- [ ] Arrays vs slices: slice header internals (`ptr/len/cap`), `append` growth strategy, slicing pitfalls, `copy`
- [ ] Maps: internals (buckets), iteration order randomness, comma-ok idiom, concurrent access danger
- [ ] Strings, runes, bytes, `utf8` package, `strings.Builder`
- [ ] Interfaces: implicit satisfaction, interface segregation, `any`, type assertions & type switches, common stdlib interfaces (`error`, `io.Reader`, `fmt.Stringer`)
- [ ] Errors: sentinel errors, custom error types, wrapping with `%w`, `errors.Is`/`errors.As`, `panic`/`recover` (and when *not* to use them)
- [ ] `defer`: LIFO order, argument evaluation timing, defer in loops
- [ ] Packages: visibility rules, `init()` order, import cycles, project layout
- [ ] `io` package: `Reader`/`Writer`/`Closer`, `io.Copy`, `bufio`
- [ ] `encoding/json`: marshaling, struct tags, custom `MarshalJSON`, streaming with `Decoder`
- [ ] `time`: durations, timers vs tickers, monotonic clocks, time zones
- [ ] OS interaction: signals (`os/signal`), exit codes, environment, file I/O
- [ ] `go vet`, `gofmt` basics



## Phase 2 - Go concurrency deep dive (Days 11-18)

- [ ] Goroutines: what they cost, GMP scheduler model (G/M/P, work stealing), `GOMAXPROCS`
- [ ] Channels: unbuffered (synchronous handoff) vs buffered semantics, blocking rules, `nil` channel blocks forever
- [ ] Channel idioms: closing channels, `for range` over channels, who closes?
- [ ] `select`: multiplexing, `default` for non-blocking ops, timeouts with `time.After`
- [ ] Pipelines: stages as goroutines, fan-out/fan-in, cancellation propagation, bounded parallelism (worker pools, semaphore pattern)
- [ ] `sync.Mutex` / `RWMutex`: when RWMutex helps and when it hurts, lock ordering & deadlocks
- [ ] `sync.WaitGroup` (add-before-go, no copying), `sync.Once`
- [ ] `context`: cancellation, deadlines/timeouts, values (request-scoped data only)
- [ ] Go memory model: happens-before relationships
- [ ] Data races: `-race` detector, common race patterns (loop variable capture, concurrent map access, slice sharing)
- [ ] Goroutine leaks: how to detect, common causes
- [ ] `errgroup`, `singleflight` (`golang.org/x/sync`)
- [ ] Graceful shutdown: signal handling + `http.Server.Shutdown` + draining workers
- [ ] **Project:** concurrent web crawler with dedup and graceful shutdown



## Phase 3 - Go applied: testing, profiling (Days 19-21)

- [ ] Table-driven tests, subtests, test helpers (`t.Helper()`), `testify` vs stdlib
- [ ] `httptest` for handlers, mocking interfaces, dependency injection in Go
- [ ] Benchmarks: `testing.B`, `b.N`, avoiding compiler optimizations fooling you
- [ ] `pprof`: CPU, heap, goroutine profiles; `go tool pprof`
- [ ] **Project:** in-memory key-value store with HTTP API, benchmarks + a pprof-driven optimization



## Phase 4 - Networking & API design (Days 22-27)

- [ ] TCP vs UDP, 3-way handshake - the theory every backend dev needs
- [ ] DNS: resolution flow, record types, TTL, caching issues
- [ ] TLS: handshake (1.2 vs 1.3), certificates, SNI
- [ ] HTTP/1.1: methods, status codes, headers, cookies, keep-alive, chunked encoding
- [ ] `net/http` server internals: `ServeMux` (1.22+ routing), `Handler`/`HandlerFunc`, request lifecycle
- [ ] Middleware pattern: logging, auth, recovery, CORS, request IDs, timeouts
- [ ] REST: resource naming, idempotency (keys), pagination (offset vs cursor), filtering/sorting, versioning strategies, OpenAPI
- [ ] gRPC: Protocol Buffers, unary/server-streaming/client-streaming/bidi, deadlines, interceptors
- [ ] Load balancing algorithms: round-robin, weighted, least-connections, consistent hashing
- [ ] Reverse proxy (`httputil.ReverseProxy`)
- [ ] **Projects:** (1) REST API in **stdlib only** - no frameworks; (2) gRPC service with streaming; (3) L7 load balancer in Go



## Phase 5 - SQL & relational databases (Days 28-33)

- [ ] Data modeling, normalization (1NF -> BCNF), when to denormalize
- [ ] Postgres setup (in Docker); `psql` essentials
- [ ] Indexes: B-tree internals, composite index column order, covering indexes, partial indexes
- [ ] `EXPLAIN` / `EXPLAIN ANALYZE`: reading query plans
- [ ] ACID, transactions, isolation levels (read committed / repeatable read / serializable), anomalies (dirty read, non-repeatable read, phantom)
- [ ] Row locking (`FOR UPDATE`), deadlocks - reproduce and resolve one
- [ ] Joins (all types), window functions, CTEs (incl. recursive)
- [ ] Query optimization: N+1 problem, `IN` vs joins, keyset pagination
- [ ] Connection pooling in Go (`pgx` pool), prepared statements
- [ ] Migrations (`golang-migrate`), schema versioning discipline
- [ ] **Project:** URL shortener backend on Postgres - schema, migrations, indexes, transactional click counting



## Phase 6 - NoSQL, Redis & caching (Days 34-36)

- [ ] Redis data structures: strings, hashes, lists, sets, sorted sets, bitmaps, HyperLogLog, geospatial
- [ ] TTL, eviction policies (LRU/LFU variants), persistence (RDB/AOF)
- [ ] Pipelines, transactions (`MULTI`/`EXEC`)
- [ ] Caching patterns: cache-aside, read-through, write-through, write-behind, refresh-ahead
- [ ] Cache stampede / thundering herd: `singleflight`, request coalescing, probabilistic early refresh
- [ ] Invalidation strategies, TTL jitter, negative caching
- [ ] **Projects:** (1) rate limiter service on Redis; (2) distributed cache with consistent hashing in Go



## Phase 7 - System design fundamentals (Days 37-41)

- [ ] Back-of-envelope estimation: QPS, storage, bandwidth math; powers of 10 & latency numbers every engineer should know
- [ ] CAP theorem - what it actually says (and common misquotes); PACELC
- [ ] Consistency models: strong, eventual, causal, read-your-writes, monotonic reads; linearizability vs serializability
- [ ] Availability math: "nines", error budgets; latency percentiles p50/p95/p99/p999 and tail latency causes
- [ ] Reliability patterns: timeouts, retries with exponential backoff + jitter, hedging, deadlines
- [ ] Circuit breaker, bulkhead, load shedding, backpressure
- [ ] Rate limiting: token bucket, leaky bucket, fixed window, sliding window log/counter (implement all in Go)
- [ ] DB scaling: read replicas, replication topologies (leader-follower vs leaderless), sharding, choosing partition keys
- [ ] Async architectures: queues, workers, schedulers
- [ ] **Drills:** estimate capacity for TinyURL, WhatsApp, YouTube (numbers on paper)



## Phase 8 - Distributed systems core (Days 42-47)

- [ ] Quorums: W + R > N, sloppy quorums, read repair, anti-entropy
- [ ] Consensus: Raft - leader election, log replication, safety guarantees - **implement a minimal Raft**
- [ ] Distributed locks done right: fencing tokens
- [ ] Distributed transactions: 2PC (and why it blocks), Saga (choreography vs orchestration - implement one)
- [ ] Transactional outbox pattern + idempotent consumers (implement with Postgres)
- [ ] **Capstone:** 3-node replicated key-value store with leader election + log replication in Go



## Phase 9 - Messaging & event-driven architecture (Days 48-49)

- [ ] Kafka: topics, partitions, offsets, consumer groups, rebalancing
- [ ] Kafka internals: ISR, replication, log segments, exactly-once semantics (idempotent producer + transactions), log compaction
- [ ] Partitioning strategies & ordering guarantees
- [ ] Queue vs pub/sub, push vs pull models
- [ ] Poison messages & DLQ, message deduplication
- [ ] **Project:** order service with Kafka in Go



## Phase 10 - Docker (Days 50-51)

- [ ] Images & layers, Dockerfile instructions, BuildKit, layer caching, `.dockerignore`
- [ ] Multi-stage builds; distroless & scratch images for Go (static binaries with `CGO_ENABLED=0`)
- [ ] Volumes, bind mounts; networking: bridge/host/none, port mapping, embedded DNS
- [ ] Docker Compose: full local stack (Go API + Postgres + Redis + Kafka)
- [ ] Resource limits (`--memory`, `--cpus`), healthchecks, restart policies
- [ ] Security: non-root users, read-only filesystems, no secrets in layers
- [ ] **Project:** production-grade container setup for your Phase 4 API - hardened image, Compose-orchestrated



## Phase 11 - Kubernetes (Days 52-57)

- [ ] Architecture: API server, scheduler, controller manager, etcd; kubelet, kube-proxy, CNI/CRI; `kind` local cluster
- [ ] Pods: lifecycle, labels/selectors/annotations, namespaces, init containers
- [ ] Deployments & ReplicaSets; Jobs, CronJobs
- [ ] Services: ClusterIP/NodePort/LoadBalancer, headless services
- [ ] Ingress & IngressClass
- [ ] ConfigMaps, Secrets, env vs mounted config
- [ ] Probes: liveness/readiness/startup; resource requests/limits & QoS classes
- [ ] Autoscaling: HPA
- [ ] Deployment strategies: rolling (maxSurge/maxUnavailable), rollbacks
- [ ] Helm: charts, values, templating - package your Go app
- [ ] `kubectl` essentials, `k9s`
- [ ] **Project:** deploy your microservice stack to K8s - manifests + Helm chart + HPA + Ingress



## Phase 12 - Microservices, security, observability, CI/CD (Days 58-60)

- [ ] AuthN/AuthZ: OAuth2 flows awareness, OIDC, JWT structure/validation/rotation/revocation, sessions vs tokens, API keys
- [ ] Input validation, SQL injection prevention, secrets management
- [ ] Observability in Go: structured logging (`log/slog`), Prometheus metrics (counters/gauges/histograms), OpenTelemetry tracing - instrument a service end to end
- [ ] CI/CD: GitHub Actions pipelines (test -> build -> scan with Trivy -> push -> deploy), semantic versioning, trunk-based development, feature flags
- [ ] GitOps: ArgoCD
- [ ] 12-factor app methodology
- [ ] **Project:** production-readiness checklist applied to one service - auth, metrics, traces, dashboards, pipeline



## Phase 13 - System design interview drills (Days 61-63)

Classic system design problems - I'll do each **timed (35-45 min)**, out loud, with trade-offs:

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

**System design interview framework:** clarify requirements -> back-of-envelope -> high-level design -> deep dive (pick 2-3 components) -> bottlenecks & failure modes -> wrap up trade-offs.

---



## 63-day plan


| Day | Focus                                        | Build / Do                                             |
| --- | -------------------------------------------- | ------------------------------------------------------ |
| 1   | Go setup, modules, toolchain                 | Hello world CLI + `go vet` clean                       |
| 2   | Types, zero values, constants, iota          | Exercises: iota enum, shadowing traps                  |
| 3   | Functions, closures, variadic                | Higher-order function utilities                        |
| 4   | Pointers, structs, methods, embedding        | Shape hierarchy via embedding                          |
| 5   | Slices internals                             | Implement `Filter`/`Map` without `append` bugs         |
| 6   | Maps, strings, runes, utf8                   | Word-frequency counter                                 |
| 7   | Interfaces                                   | `io.Writer` implementations; stringer                  |
| 8   | Errors: wrap, Is/As                          | Error package with domain errors                       |
| 9   | Packages, project layout, defer              | Multi-package layout + defer patterns                  |
| 10  | io, encoding/json, time                      | JSON log parser CLI                                    |
| 11  | Goroutines + GMP model                       | Goroutine fan-out benchmark                            |
| 12  | Channels: buffered/unbuffered + idioms       | Blocking-behavior experiments                          |
| 13  | select + timeouts                            | Timeout/cancel patterns                                |
| 14  | Pipelines, fan-out/fan-in                    | Parallel pipeline                                      |
| 15  | Mutex/RWMutex, WaitGroup, Once               | Concurrent counter variants                            |
| 16  | context                                      | Request-scoped timeout propagation                     |
| 17  | Memory model + race detector                 | Find & fix races with `-race`                          |
| 18  | **Project:** concurrent web crawler          | Crawler with dedup + graceful shutdown                 |
| 19  | Testing: table-driven                        | Table tests for your KV store                          |
| 20  | Benchmarks + pprof                           | Profile & optimize a hot path                          |
| 21  | **Project:** in-memory KV + HTTP API         | With benchmarks                                        |
| 22  | DNS, TLS, HTTP/1.1                           | Inspect real handshakes (`openssl s_client`)           |
| 23  | REST with stdlib only                        | CRUD API, no frameworks                                |
| 24  | Middleware                                   | Logging/auth/recovery/request-ID chain                 |
| 25  | API design: idempotency, pagination          | Idempotency-key middleware                             |
| 26  | gRPC + protobuf                              | Streaming gRPC service                                 |
| 27  | **Project:** L7 load balancer                | Round-robin, least-conn, consistent hash               |
| 28  | Data modeling, normalization                 | Schema for URL shortener                               |
| 29  | Indexes + EXPLAIN ANALYZE                    | Index experiments                                      |
| 30  | Transactions + isolation levels              | Money-transfer with tx                                 |
| 31  | Joins, window functions, CTEs                | Analytics queries                                      |
| 32  | Pooling (pgx), N+1                           | Fix N+1 in an API                                      |
| 33  | **Project:** URL shortener on Postgres       | Full backend + migrations                              |
| 34  | Redis data structures                        | Cache + leaderboard exercises                          |
| 35  | Caching patterns + singleflight              | Cache-aside with stampede protection                   |
| 36  | **Project:** distributed cache               | Consistent-hashing nodes                               |
| 37  | Estimation math + latency numbers            | Estimate TinyURL/WhatsApp/YouTube                      |
| 38  | CAP/PACELC, consistency models               | 2-minute explanations, recorded                        |
| 39  | Retries/backoff/jitter, circuit breaker      | `gobreaker`-style breaker from scratch                 |
| 40  | Rate limiting algorithms                     | All 4 algorithms in Go                                 |
| 41  | DB scaling: replicas, sharding               | Shard-key design exercises                             |
| 42  | Quorums W+R>N                                | Quorum KV prototype                                    |
| 43  | Raft: election + log replication             | Minimal Raft in Go                                     |
| 44  | Distributed locks + fencing                  | Fencing-token lock service                             |
| 45  | 2PC vs Saga                                  | Saga orchestrator                                      |
| 46  | Outbox + idempotent consumers                | Outbox with Postgres                                   |
| 47  | **Capstone:** 3-node replicated KV           | Raft-lite in Go                                        |
| 48  | Kafka: topics, partitions, EOS               | Local Kafka via Docker                                 |
| 49  | **Project:** event-driven order service      | Go + Kafka                                             |
| 50  | Dockerfile, multi-stage, hardening           | Distroless hardened Go image                           |
| 51  | Docker Compose                               | Full stack: Go + PG + Redis + Kafka                    |
| 52  | K8s architecture; kind cluster               | Cluster up, deploy nginx                               |
| 53  | Pods, labels, namespaces, config             | Deploy Go API + ConfigMaps/Secrets                     |
| 54  | Deployments, rolling updates                 | Rollout + rollback drill                               |
| 55  | Services, Ingress                            | Expose API via Ingress                                 |
| 56  | Probes, resources, HPA                       | Load-test autoscaling                                  |
| 57  | **Project:** full stack on K8s               | Manifests + Helm + HPA + Ingress                       |
| 58  | Auth: JWT in Go                              | Auth service issuing JWTs                              |
| 59  | Observability: slog, Prometheus, OTel        | Instrumented service                                   |
| 60  | CI/CD: GitHub Actions, ArgoCD                | Pipeline: test -> scan -> deploy                       |
| 61  | Drill: URL shortener + rate limiter          | Timed, out loud                                        |
| 62  | Drill: Twitter feed + chat system            | Timed, out loud                                        |
| 63  | Drill: distributed KV + review               | Polish & retrospective                                 |




---



## Capstone projects

1. **Concurrent web crawler** - worker pools, politeness, dedup, graceful shutdown
2. **URL shortener** - Postgres, Redis cache, base62, analytics
3. **Distributed cache** - consistent hashing, replication, failure handling
4. **Raft-lite KV store** - 3-node consensus, leader election, log replication
5. **Event-driven order system** - Kafka, outbox pattern, saga, notification fan-out
6. **L7 load balancer** - health checks, multiple algorithms, observability
7. **K8s-deployed microservice** - Helm chart, HPA, mTLS, dashboards, CI/CD

---



## Backlog - topics to add later

When I find a topic that's missing, I add it here (or under the right phase above) as a checkbox and work through it. This roadmap grows with me.

- [ ] *(example) eBPF basics for observability*
- [ ] *add yours here...*
