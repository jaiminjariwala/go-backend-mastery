# My understanding of rate limiting

I use a token bucket to decide whether a request can enter the API. Load balancing chooses where work goes; rate limiting decides whether to admit it.

I open `/rate-limiting.html` on the same local server. I can change user count, request spacing, bucket capacity, and refill rate. Run experiment plays the decisions; Step inspects one arrival at a time. All requests share one bucket, not separate per-user buckets.

## My example

I start with capacity 2 and refill 2 tokens per second. Each request needs 1 token. Requests arrive every 100 ms.

| Time | Request | Tokens before | Decision | Tokens after |
| --- | --- | ---: | --- | ---: |
| 0 ms | 1 | 2 | Allowed | 1 |
| 100 ms | 2 | 1.2 | Allowed | 0.2 |
| 200 ms | 3 | 0.4 | Rejected, 429 | 0.4 |
| 300 ms | 4 | 0.6 | Rejected, 429 | 0.6 |
| 400 ms | 5 | 0.8 | Rejected, 429 | 0.8 |
| 500 ms | 6 | 1 | Allowed | 0 |

```text
People send requests
        |
        v
Token bucket: is at least one token available?
        |
    +---+---+
    |       |
   YES      NO
    |       |
Use token   Reject: 429 Too Many Requests
    |       No queue and no token consumed
    v
API server receives request
```

Capacity permits bursts. Refill controls the sustained admission rate. Two tokens per second does not mean a strict maximum of two in every one-second window when saved tokens are available.

The bucket refills continuously with simulated time, capped at capacity. Fractional tokens accumulate, but a request needs one whole token. Retry-after milliseconds describe when one token would be available if no competing request uses it first; it is not a reservation. Actual HTTP Retry-After headers use whole seconds or a date.

## What each file does

`simulation.go` calculates the decisions using integer thousandths of a token. `simulation_test.go` checks rejection, exact refill, capacity bounds, and invalid input. The shared server exposes `POST /api/rate-limit`. `rate-limiting.html` and `rate-limit.js` display the returned history using the existing website styles and assets.

This is a deterministic teaching simulation, not production middleware. The API returns a history as JSON; 200/429 are modeled outcomes inside it. No real Redis or protected service is involved. Rejected requests are not automatically retried. A production shared limit across multiple instances would need coordinated, atomic state, for example in Redis, and a deliberate per-user, per-IP, or API-wide policy.
