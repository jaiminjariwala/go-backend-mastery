# Go Backend Mastery

I'm learning backend engineering and distributed systems **in depth, in Go**.



---

## Phase overview


| Phase | Focus                                         |
| ----- | --------------------------------------------- |
| 1     | Go language fundamentals                      |
| 2     | Go concurrency deep dive                      |
| 3     | Go applied: testing, profiling                |
| 4     | Networking & API design                       |
| 5     | SQL & relational databases                    |
| 6     | NoSQL, Redis & caching                        |
| 7     | System design fundamentals                    |
| 8     | Distributed systems core                      |
| 9     | Messaging & event-driven architecture         |
| 10    | Docker                                        |
| 11    | Kubernetes                                    |
| 12    | Microservices, security, observability, CI/CD |
| 13    | System design interview drills                |




---




| Day | Focus                                   | Build / Do                                     |
| --- | --------------------------------------- | ---------------------------------------------- |
| 1   | Go setup, modules, toolchain            | Hello world CLI + `go vet` clean               |
| 2   | Types, zero values, constants, iota     | Exercises: iota enum, shadowing traps          |
| 3   | Functions, closures, variadic           | Higher-order function utilities                |
| 4   | Pointers, structs, methods, embedding   | Shape hierarchy via embedding                  |
| 5   | Slices internals                        | Implement `Filter`/`Map` without `append` bugs |
| 6   | Maps, strings, runes, utf8              | Word-frequency counter                         |
| 7   | Interfaces                              | `io.Writer` implementations; stringer          |
| 8   | Errors: wrap, Is/As                     | Error package with domain errors               |
| 9   | Packages, project layout, defer         | Multi-package layout + defer patterns          |
| 10  | io, encoding/json, time                 | JSON log parser CLI                            |
| 11  | Goroutines + GMP model                  | Goroutine fan-out benchmark                    |
| 12  | Channels: buffered/unbuffered + idioms  | Blocking-behavior experiments                  |
| 13  | select + timeouts                       | Timeout/cancel patterns                        |
| 14  | Pipelines, fan-out/fan-in               | Parallel pipeline                              |
| 15  | Mutex/RWMutex, WaitGroup, Once          | Concurrent counter variants                    |
| 16  | context                                 | Request-scoped timeout propagation             |
| 17  | Memory model + race detector            | Find & fix races with `-race`                  |
| 18  | **Project:** concurrent web crawler     | Crawler with dedup + graceful shutdown         |
| 19  | Testing: table-driven                   | Table tests for your KV store                  |
| 20  | Benchmarks + pprof                      | Profile & optimize a hot path                  |
| 21  | **Project:** in-memory KV + HTTP API    | With benchmarks                                |
| 22  | DNS, TLS, HTTP/1.1                      | Inspect real handshakes (`openssl s_client`)   |
| 23  | REST with stdlib only                   | CRUD API, no frameworks                        |
| 24  | Middleware                              | Logging/auth/recovery/request-ID chain         |
| 25  | API design: idempotency, pagination     | Idempotency-key middleware                     |
| 26  | gRPC + protobuf                         | Streaming gRPC service                         |
| 27  | **Project:** L7 load balancer           | Round-robin, least-conn, consistent hash       |
| 28  | Data modeling, normalization            | Schema for URL shortener                       |
| 29  | Indexes + EXPLAIN ANALYZE               | Index experiments                              |
| 30  | Transactions + isolation levels         | Money-transfer with tx                         |
| 31  | Joins, window functions, CTEs           | Analytics queries                              |
| 32  | Pooling (pgx), N+1                      | Fix N+1 in an API                              |
| 33  | **Project:** URL shortener on Postgres  | Full backend + migrations                      |
| 34  | Redis data structures                   | Cache + leaderboard exercises                  |
| 35  | Caching patterns + singleflight         | Cache-aside with stampede protection           |
| 36  | **Project:** distributed cache          | Consistent-hashing nodes                       |
| 37  | Estimation math + latency numbers       | Estimate TinyURL/WhatsApp/YouTube              |
| 38  | CAP/PACELC, consistency models          | 2-minute explanations, recorded                |
| 39  | Retries/backoff/jitter, circuit breaker | `gobreaker`-style breaker from scratch         |
| 40  | Rate limiting algorithms                | All 4 algorithms in Go                         |
| 41  | DB scaling: replicas, sharding          | Shard-key design exercises                     |
| 42  | Quorums W+R>N                           | Quorum KV prototype                            |
| 43  | Raft: election + log replication        | Minimal Raft in Go                             |
| 44  | Distributed locks + fencing             | Fencing-token lock service                     |
| 45  | 2PC vs Saga                             | Saga orchestrator                              |
| 46  | Outbox + idempotent consumers           | Outbox with Postgres                           |
| 47  | **Capstone:** 3-node replicated KV      | Raft-lite in Go                                |
| 48  | Kafka: topics, partitions, EOS          | Local Kafka via Docker                         |
| 49  | **Project:** event-driven order service | Go + Kafka                                     |
| 50  | Dockerfile, multi-stage, hardening      | Distroless hardened Go image                   |
| 51  | Docker Compose                          | Full stack: Go + PG + Redis + Kafka            |
| 52  | K8s architecture; kind cluster          | Cluster up, deploy nginx                       |
| 53  | Pods, labels, namespaces, config        | Deploy Go API + ConfigMaps/Secrets             |
| 54  | Deployments, rolling updates            | Rollout + rollback drill                       |
| 55  | Services, Ingress                       | Expose API via Ingress                         |
| 56  | Probes, resources, HPA                  | Load-test autoscaling                          |
| 57  | **Project:** full stack on K8s          | Manifests + Helm + HPA + Ingress               |
| 58  | Auth: JWT in Go                         | Auth service issuing JWTs                      |
| 59  | Observability: slog, Prometheus, OTel   | Instrumented service                           |
| 60  | CI/CD: GitHub Actions, ArgoCD           | Pipeline: test -> scan -> deploy               |
| 61  | Drill: URL shortener + rate limiter     | Timed, out loud                                |
| 62  | Drill: Twitter feed + chat system       | Timed, out loud                                |
| 63  | Drill: distributed KV + review          | Polish & retrospective                         |


---

