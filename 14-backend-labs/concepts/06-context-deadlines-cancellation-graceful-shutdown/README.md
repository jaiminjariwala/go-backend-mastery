# Lesson 06: Context, deadlines, cancellation, and graceful shutdown

## My understanding

I use context to carry a cancellation signal and an optional deadline to ongoing work. Context is not a database connection or a command that forcibly kills code. My operation must listen to it, or use an API that supports it.

I think of ordering food: my HTTP request is the order, the handler coordinates it like a waiter, and the slow operation is the kitchen preparing it. If I cancel, the waiter tells the kitchen the order is no longer needed. The kitchen must cooperate and stop.

## Cancellation: I no longer need the work

```text
I start a 10-second task
             │
             ▼
Browser sends a request; Go starts work
             │
       I click Cancel
             │
             ▼
Browser aborts the request
             │
             ▼
The request context signals cancellation
             │
             ▼
My waiting operation notices and stops early
Go terminal records that work stopped
```

The browser knows it canceled, but cannot receive confirmation over the aborted request. I check the Go terminal for evidence that the server stopped too. This example is a real interruptible timer, not a database write. A plain long Sleep would not notice cancellation early.

## Timeout and deadline: I only wait this long

A timeout is a duration, such as two seconds. A deadline is the cutoff time calculated from it. I use WithTimeout to create that cutoff. An earlier parent deadline or cancellation still wins. I release context resources with cancel even when work succeeds.

```text
Task needs 5 seconds; I allow only 2 seconds
                     │
                     ▼
At 2 seconds, the deadline signal arrives
                     │
                     ▼
The operation stops instead of waiting all 5 seconds
                     │
                     ▼
The browser receives a timeout response
```

This demo chooses status 408 for its timeout message; context itself does not select an HTTP status. In real APIs I choose a status for the situation: 408 conventionally concerns request timeout, and an upstream gateway timeout may use 504.

## Graceful shutdown: closing the restaurant

Cancellation concerns one order. Shutdown concerns the whole restaurant. I stop letting new customers in, then give existing customers time to finish. I do not cancel every active request immediately just because I am closing.

```text
I press Control+C (or the process receives SIGTERM)
                       │
                       ▼
Go stops accepting new connections
                       │
                       ▼
Existing requests can finish within 12 seconds
                       │
           ┌───────────┴────────────┐
           ▼                        ▼
All requests finish          12-second budget expires
           │                        │
           ▼                        ▼
Clean shutdown               Close remaining connections
           │                        │
           └───────────┬────────────┘
                       ▼
Main returns; database pool is closed
```

I use a fresh shutdown context because the context listening for the interrupt is already canceled. Shutdown waits for ordinary active HTTP requests; it does not automatically drain hijacked connections such as WebSockets. Closing connections signals their request contexts, but cannot forcibly stop arbitrary non-cooperative goroutines. A second interrupt can terminate immediately.

## How I run and verify it

From my repository root:

```sh
go run ./14-backend-labs/cmd/server
```

I open http://127.0.0.1:8080/concepts/context. I keep DATABASE_URL in the terminal if I also want the notes API, but lesson 6 itself needs no database or new installation.

1. I click **Finish in 1 second** and expect 200.
2. I click **5-second task, 2-second limit** and expect a timeout after roughly two seconds.
3. I start the 10-second task and cancel early. I check the terminal for “client canceled; task stopped early”.
4. To test graceful shutdown, I start the 10-second task again, leave it running, and press Control+C in the Go terminal. I keep the browser open. The existing task should finish, then the terminal reports shutdown complete. New requests stop working because the server has closed its entrance.
5. I restart Go to use the app again. If my terminal's go run launcher exits immediately on interrupt, I can build a standalone executable with `go build -o /tmp/go-backend-lab-server ./14-backend-labs/cmd/server`, run `/tmp/go-backend-lab-server`, and repeat the shutdown check.

## Files and tests

| File | What it does |
| --- | --- |
| handler.go | Waits cooperatively and handles completion, timeout, and cancellation |
| demo.go | Registers fixed-duration tasks and the page |
| index.html | Starts requests and cancels with AbortController |
| shutdown.go | Waits for active requests with a bounded shutdown budget |
| handler_test.go | Checks completion, deadline, cancellation, and no response body after cancellation |
| shutdown_test.go | Uses a local HTTP connection to check that shutdown waits for a running request |

```sh
go test -v ./14-backend-labs/concepts/06-context-deadlines-cancellation-graceful-shutdown
```

The handler tests avoid long waits. The shutdown test uses a real local listener, not Neon. Browser behavior and keyboard signals are separate manual checks. All four lesson topics are implemented; this is a small learning example, not a complete production lifecycle framework.
