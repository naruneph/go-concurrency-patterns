# Go Concurrency Patterns

Small, runnable examples of practical Go concurrency patterns: pipelines, pools, rate limits, supervisors.

## Run

List all available examples and their docs:
```bash
go run ./cmd/gopatterns -list
```

Run a specific example:
```bash
go run ./cmd/gopatterns -example=<name>

# e.g.
go run ./cmd/gopatterns -example=flows_bridge
```

## Available examples

| Example key                   | Description                                                                                        |
| ----------------------------- | -------------------------------------------------------------------------------------------------- |
| `flows_bridge`                | Flatten a channel-of-channels into a single stream with ctx-aware backpressure.                    |
| `flows_fan_out_fan_in`        | Fan-out to N workers and fan-in results to a single channel, all ctx-cancellable.                  |
| `flows_generator`             | Cancellable number generator that emits an arithmetic sequence over a channel.                     |
| `flows_or`                    | Combine multiple Done channels into one that closes when any of them closes.                       |
| `flows_ordone`                | Wrap a source channel so consumers can stop early without leaking upstream goroutines.             |
| `flows_pipeline`              | Cancellable pipeline with map-like stages wired by channels.                                       |
| `flows_selectloop`            | Cancellable for-select worker loop that stops on context cancel or input close.                    |
| `flows_tee`                   | Duplicate a stream to two outputs with ctx-aware backpressure and clean shutdown.                  |
| `orchestration_contextio`     | Demonstrates a writer that respects context cancellation.                                          |
| `orchestration_contextio_net` | Minimal TCP echo with context-aware Read/Write via deadlines; shows success then a timed-out read. |
| `orchestration_errgroup`      | Errgroup usage to run a group of goroutines with error propagation and cancellation.               |
| `orchestration_shutdown`      | Graceful shutdown with phases: stop intake, soft drain, hard cancel.                               |
| `pools_semaphore`             | A simple semaphore implementation to limit concurrent access.                                      |
| `pools_semaphore_sync`        | Weighted semaphore from x/sync to limit concurrent access by weight.                               |
| `pools_workerpool`            | Fixed-size worker pool with a bounded queue.                                                       |
| `pools_workerpool_dynamic`    | Dynamic-size worker pool with a bounded queue.                                                     |
| `primitives_atomic`           | Demonstrates atomic operations using `sync/atomic`.                                                |
| `primitives_broadcast`        | `sync.Cond` with `Broadcast` to notify all waiting goroutines.                                     |
| `primitives_once`             | `sync.Once` for executing a function only once.                                                    |
| `primitives_pool`             | `sync.Pool` for managing expensive-to-allocate objects.                                            |
| `primitives_rwmutex`          | `sync.RWMutex` for concurrent read/write access to a map.                                          |
| `primitives_signal`           | `sync.Cond` with `Signal` to notify one waiting goroutine.                                         |
| `primitives_waitgroup`        | `sync.WaitGroup` for waiting on a set of goroutines.                                               |
| `scale_healing`               | Healing goroutines: context-driven supervisor that respawns a ward on missed heartbeats.           |
| `scale_heartbeat_time`        | Simple heartbeat mechanism using `time.Ticker`.                                                    |
| `scale_heartbeat_workunit`    | Heartbeat mechanism driven by work-unit progress.                                                  |
| `scale_ratelimit`             | Simple token-bucket rate limiter.                                                                  |

