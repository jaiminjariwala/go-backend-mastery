# My URL shortener: cache, limits, and measurement

I am extending my existing URL shortener, not the separate simulation project. The dashboard reports actual Go lookups and process counters. It does not invent cache hits, latency, or benchmark numbers.

## Setup I need once

1. I start Docker Desktop or my Colima Docker runtime. The Docker command alone is not enough; its daemon must be running.
2. From the repository root I run:

```sh
docker compose -f 15-url-shortener/compose.yaml up -d
```

This starts one disposable local Redis container, bound only to 127.0.0.1:6379. No public port or password is needed for this isolated local lab. Cache/limit contents are not persisted. I must not reuse these settings on a public host. Redis uses a 128MB limit with no eviction, so it cannot silently evict rate-limit keys to make room for cache entries; cache writes can instead fail and the application uses PostgreSQL.

3. I privately set my existing PostgreSQL connection string. In zsh:

```sh
read -rs 'DATABASE_URL?Paste PostgreSQL connection string (hidden): '; echo
export DATABASE_URL
export REDIS_URL='redis://127.0.0.1:6379/0'
go run ./15-url-shortener
```

I paste the connection string only at the hidden prompt, not into the command itself. I do not paste passwords into chat or commit them. I use the existing database schema in `schema.sql`; this upgrade does not modify it. The program does not load `.env` files automatically.

I open http://127.0.0.1:8081/. Without REDIS_URL, the original database-only behavior still works, and the UI explicitly says that cache and limiting are disabled. A configured Redis URL is not a guarantee of health: a lookup tests availability.

## What I see in the UI

I create a link or use one from the existing list. Each row has two actions:

- **Trace lookup** runs the same cache/database lookup used by a redirect, but returns its trace as JSON so I stay on the dashboard.
- **Baseline lookup** deliberately skips Redis for this lookup, reads PostgreSQL, and does not warm the cache.

The trace shows cache state, source, database-read flag, cache-write success, elapsed lookup time, instance ID, and the rate-limit decision. It is not the complete browser navigation time. The rate-limit check is outside this lookup timing; an end-to-end load test includes it.

The cache-hit, cache-miss, lookup-database-read, and rejection counters are cumulative for this Go process since startup. They are not global cluster counters. Listing and creating links use SQL too, but are deliberately not counted as destination-lookup SELECTs.

## My first lookup: cache-aside

For illustration, my database could contain:

| PostgreSQL code (primary key) | original_url |
| --- | --- |
| aB3kP9xQ | https://example.com/articles/learn-go |

```text
Person opens /s/aB3kP9xQ
          |
          v
Go API: shared rate allowance available?
          |
          +-- No --> HTTP 429, Retry-After header
          |
         Yes
          v
Redis GET shortener:v1:link:aB3kP9xQ
          |
          +-- HIT --> destination returned without a SQL lookup
          |
         MISS
          v
PostgreSQL SELECT original_url WHERE code = 'aB3kP9xQ'
          |
          v
Redis SET mapping with a 60-second expiry
          |
          v
Go returns HTTP 302 and Location: original URL
          |
          v
Browser visits the original website
```

After a miss, Redis might contain:

| Redis key | Value | TTL |
| --- | --- | --- |
| shortener:v1:link:aB3kP9xQ | https://example.com/articles/learn-go | Up to 60 seconds |

Redis is an extra, fast copy, not the owner of the data. PostgreSQL remains the source of truth. Expiry or a Redis restart must not delete the actual link. This is cache-aside: Go checks Redis, loads missing data from PostgreSQL, then fills Redis. Missing links are not cached. No edit/delete operation exists yet, so update invalidation is a future requirement, not implemented here. Concurrent cold misses can still duplicate database reads; stampede protection is not implemented yet.

I trace the same code twice. If it was not cached, the first says miss/database read/cache saved and the second says hit/no database read. Waiting 60 seconds without rewriting that entry causes it to expire. If cache save fails, a subsequent hit is not guaranteed. I can compare a baseline lookup, but one pair of timings does not establish a reliable speedup.

## My rate limiter

Redis stores a counter for each client IP and route group. Link creation has its own budget; redirect and trace lookups share another. Each budget permits 20 requests in a 10-second window that starts at the first request. The atomic Lua script increments the counter and sets the expiry together. Requests beyond the allowance return 429 with Retry-After. Rejections are counted too until the window expires.

This version is a fixed-window limiter, not a token bucket or sliding window. It can allow a burst around window boundaries. It uses the actual remote connection IP, ignores spoofable forwarded headers, and assumes the server is reached directly. A reverse proxy deployment needs trusted-proxy handling; otherwise all users may appear to have the proxy's IP. Local browsers usually share 127.0.0.1 and therefore the same allowance. It is not authentication or complete abuse prevention. Listing and diagnostics are not rate limited in this local lab.

If Redis fails, the limiter returns 503 (fail closed). This protects admission rules but sacrifices availability. Cache failures alone fall back to PostgreSQL (fail open for caching). I must choose and explain these different policies rather than call every failure a cache miss.

## What a baseline load test means

A baseline is my measured starting point before claiming an optimization helped. I use the same saved code, hardware, database location, request count, concurrency, and success criteria in both runs. I deliberately disable limiting during a controlled localhost throughput test so quick 429s do not masquerade as successful redirects.

In the terminal where DATABASE_URL and REDIS_URL are set:

```sh
CACHE_ENABLED=false RATE_LIMIT_ENABLED=false go run ./15-url-shortener
```

In another terminal, I replace the example code with an existing link:

```sh
go run ./15-url-shortener/cmd/loadtest -url http://127.0.0.1:8081/s/aB3kP9xQ -n 200 -c 10
```

The tool does not follow the 302 to the external website. It reports all HTTP status counts, transport errors, total request rate, successful-redirect rate, and p50/p95 latency across all responses. A 404, 429, or 500 is not a successful redirect. It bounds requests to 10,000 and concurrency to 100 and only accepts direct localhost HTTP short-link URLs.

I stop the Go server with Ctrl+C, then restart with cache enabled:

```sh
CACHE_ENABLED=true RATE_LIMIT_ENABLED=false go run ./15-url-shortener
```

I warm the chosen code once using Trace lookup and rerun the same load command. I repeat trials rather than select one flattering result. I record warm versus cold cache, hardware, PostgreSQL/Redis locations, limits, status counts, p50/p95, and cache/database counter changes. A warm repeated-key test is not representative of millions of distinct cold links.

| Measurement | Database-only baseline | Warm Redis run |
| --- | --- | --- |
| Successful redirects/second | Not measured yet | Not measured yet |
| p95 latency | Not measured yet | Not measured yet |
| Error/status distribution | Not measured yet | Not measured yet |
| Lookup database reads | Not measured yet | Not measured yet |

I do not claim 5,000 RPS or a 10× improvement without reproducible evidence. A closed-loop concurrency test is a first measurement, not a complete overload or capacity study. More advanced tests must include arrival rate, saturation, CPU, memory, database load, and realistic key distributions.

## Multiple Go instances

An instance is a separate running copy of the same application, not another goroutine. The code now accepts a localhost HTTP_ADDR and an INSTANCE_ID. In separate terminals with the same DATABASE_URL and REDIS_URL I can run:

```sh
HTTP_ADDR=127.0.0.1:8081 INSTANCE_ID=go-1 go run ./15-url-shortener
HTTP_ADDR=127.0.0.1:8083 INSTANCE_ID=go-2 go run ./15-url-shortener
HTTP_ADDR=127.0.0.1:8084 INSTANCE_ID=go-3 go run ./15-url-shortener
```

```text
Browser --> Go instance 1 --+
Browser --> Go instance 2 --+--> Same Redis: cached links and atomic limit counters
Browser --> Go instance 3 --+--> Same PostgreSQL: durable links
```

This upgrade supports configuring instances; it does not start three processes or include a real reverse-proxy load balancer. Direct requests to different instances can reuse a Redis cache entry and consume the same IP budget. Each process still has its own metrics and database pool. More instances also mean more possible database connections; scaling application processes is not a free database upgrade.

## Failure experiments

I intentionally stop one dependency and check an expected behavior. I do this only in my disposable local setup, never by deleting actual link data.

| Experiment | Expected result |
| --- | --- |
| Stop Redis with rate limiting enabled | Protected requests return 503; limiter errors increase |
| Restart Go with RATE_LIMIT_ENABLED=false, then stop Redis | Cache lookups time out quickly and fall back to PostgreSQL; redirects can still succeed |
| Restart Redis | Cache/limit state is empty in this disposable setup; durable links remain in PostgreSQL |
| PostgreSQL unavailable, cache entry still valid, Redis healthy | Warm cached redirects can still succeed until expiry; misses fail |
| Restart one Go instance | Its counters reset; shared cache and database data remain |

```sh
docker compose -f 15-url-shortener/compose.yaml stop redis
docker compose -f 15-url-shortener/compose.yaml start redis
```

Redis operations use short timeouts with retries disabled to bound dependency delay. Database lookup errors return generic messages, not credentials. This is not a circuit breaker or complete resilience design. A database outage experiment needs a dedicated local database or controlled test environment; I do not disrupt my shared Neon project.

## Which file does what?

| File | Responsibility |
| --- | --- |
| `main.go` | Loads environment settings, connects PostgreSQL and Redis, configures instance/port and shutdown |
| `handler.go` | Validates/creates/lists links and sends browser redirects |
| `redis.go` | Redis cache GET/SET and atomic shared fixed-window limit script |
| `observe.go` | Cache-aside lookup, trace/stats endpoints, rate-limit wrapper, and process counters |
| `index.html` | Existing shortener UI plus live diagnostic cards, Trace and Baseline actions |
| `compose.yaml` | Local disposable Redis service only |
| `cmd/loadtest/main.go` | Bounded localhost redirect load generator with no external navigation |
| `handler_test.go`, `cache_test.go` | Existing handler tests and mocked dependency tests for cache, bypass, fallback, and limiting |
| `schema.sql` | Durable PostgreSQL table definition, unchanged by this upgrade |

The automated tests use SQL mocks and a fake infrastructure adapter. They verify control flow, not actual Redis Lua behavior, network latency, or the live Neon connection. I still need live integration verification and benchmark runs after setting up the dependencies.

References: [official Go Redis client guide](https://redis.io/docs/latest/develop/clients/go/) and [Redis INCR rate-limiter patterns](https://redis.io/docs/latest/commands/incr/).
