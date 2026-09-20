# Phase 2 - Go concurrency deep dive

Days 13-22. Full topic checklist lives in the [main README](../README.md).

| Day | Focus | Build / Do | Folder |
|---|---|---|---|
| 13 | Goroutines + GMP model | Goroutine fan-out benchmark | `../day-013-goroutines-gmp-model/` |
| 14 | Channels: buffered vs unbuffered | Blocking-behavior experiments | `../day-014-channels-buffered-vs-unbuffered/` |
| 15 | Channel idioms: close, range, nil | Pipeline stage helpers | `../day-015-channel-idioms-close-range-nil/` |
| 16 | select + timeouts | Timeout/cancel patterns | `../day-016-select-timeouts/` |
| 17 | Pipelines, fan-out/fan-in | Parallel pipeline | `../day-017-pipelines-fan-out-fan-in/` |
| 18 | Mutex/RWMutex, WaitGroup, Once | Concurrent counter variants | `../day-018-mutex-rwmutex-waitgroup-once/` |
| 19 | sync.Map, Pool, atomics | Object pool benchmark | `../day-019-sync-map-pool-atomics/` |
| 20 | context | Request-scoped timeout propagation | `../day-020-context/` |
| 21 | Memory model + race detector | Find and fix races with -race | `../day-021-memory-model-race-detector/` |
| 22 | Project: concurrent web crawler | Crawler with dedup + graceful shutdown | `../day-022-project-concurrent-web-crawler/` |

## Phase project
See the day marked **Project** above - it doesn't count as done until the code runs and is committed.
