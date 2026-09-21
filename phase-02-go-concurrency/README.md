# Phase 2 - Go concurrency deep dive

Days 11-18. Full topic checklist lives in the [main README](../README.md).

| Day | Focus | Build / Do | Folder |
|---|---|---|---|
| 11 | Goroutines + GMP model | Goroutine fan-out benchmark | `../day-011-goroutines-gmp-model/` |
| 12 | Channels: buffered/unbuffered + idioms | Blocking-behavior experiments | `../day-012-channels-buffered-unbuffered-idioms/` |
| 13 | select + timeouts | Timeout/cancel patterns | `../day-013-select-timeouts/` |
| 14 | Pipelines, fan-out/fan-in | Parallel pipeline | `../day-014-pipelines-fan-out-fan-in/` |
| 15 | Mutex/RWMutex, WaitGroup, Once | Concurrent counter variants | `../day-015-mutex-rwmutex-waitgroup-once/` |
| 16 | context | Request-scoped timeout propagation | `../day-016-context/` |
| 17 | Memory model + race detector | Find & fix races with `-race` | `../day-017-memory-model-race-detector/` |
| 18 | **Project:** concurrent web crawler | Crawler with dedup + graceful shutdown | `../day-018-project-concurrent-web-crawler/` |

## Phase project
See the day marked **Project** above - it doesn't count as done until the code runs and is committed.
