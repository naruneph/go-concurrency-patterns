package healing

import (
	"context"
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"scale_healing",
		"Healing goroutines: a context-driven supervisor that monitors heartbeats and respawns the ward on timeout.",
		HealingExample,
	)
}

// orForward wraps an input stream and stops forwarding when ctx is canceled.
func orForward[T any](ctx context.Context, in <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- v:
				}
			}
		}
	}()
	return out
}

// bridge flattens a channel-of-channels into one stream, honoring ctx cancellation.
func bridge(ctx context.Context, streams <-chan <-chan any) <-chan any {
	out := make(chan any)
	go func() {
		defer close(out)
		for {
			var cur <-chan any
			select {
			case <-ctx.Done():
				return
			case s, ok := <-streams:
				if !ok {
					return
				}
				cur = s
			}
			for v := range orForward(ctx, cur) {
				select {
				case <-ctx.Done():
					return
				case out <- v:
				}
			}
		}
	}()
	return out
}

// take forwards up to n items from valueStream or stops early if ctx is canceled.
func take(ctx context.Context, valueStream <-chan any, n int) <-chan any {
	out := make(chan any)
	go func() {
		defer close(out)
		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-valueStream:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- v:
				}
			}
		}
	}()
	return out
}

// startGoroutineFn starts a ward under the provided ctx, asks it to pulse at pulseInterval,
// and returns a heartbeat channel (can be nil to simulate a broken ward).
type startGoroutineFn func(ctx context.Context, pulseInterval time.Duration) (heartbeat <-chan struct{})

// newSteward returns a monitor that restarts a ward when it misses heartbeats for `timeout`.
func newSteward(timeout time.Duration, startWard startGoroutineFn) startGoroutineFn {
	return func(ctx context.Context, pulseInterval time.Duration) <-chan struct{} {
		heartbeat := make(chan struct{}, 1) // buffered so our pulse is non-blocking
		go func() {
			defer close(heartbeat)

			pulse := time.NewTicker(pulseInterval)
			defer pulse.Stop()

			for {
				// Start (or restart) a ward with a fresh child context.
				wardCtx, cancelWard := context.WithCancel(ctx)
				wardHB := startWard(wardCtx, timeout/2)

				// Each cycle has its own timeout window.
				timeoutTimer := time.NewTimer(timeout)

			monitor:
				for {
					select {
					case <-ctx.Done():
						// Global stop: cancel current ward and exit.
						if !timeoutTimer.Stop() {
							<-timeoutTimer.C
						}
						cancelWard()
						return

					case <-pulse.C:
						// Steward liveness pulse; drop if nobody listening.
						select {
						case heartbeat <- struct{}{}:
						default:
						}

					case _, ok := <-wardHB:
						if !ok {
							// Ward has exited; restart it.
							fmt.Println("steward: ward has exited; restarting")
							if !timeoutTimer.Stop() { <-timeoutTimer.C }
							cancelWard()
							break monitor
						}
						// Ward is alive; reset timeout and keep monitoring.
						if !timeoutTimer.Stop() {
							<-timeoutTimer.C
						}
						timeoutTimer.Reset(timeout)

					case <-timeoutTimer.C:
						// No heartbeat in time; restart.
						fmt.Println("steward: ward unhealthy; restarting")
						cancelWard()
						break monitor
					}
				}
			}
		}()
		return heartbeat
	}
}

// irresponsibleWard: never heartbeats; just waits for ctx cancel.
// Steward will repeatedly time out and restart it.
func irresponsibleWard(ctx context.Context, _ time.Duration) <-chan struct{} {
	fmt.Println("ward: Hello, I'm irresponsible!")
	go func() {
		<-ctx.Done()
		fmt.Println("ward: I am halting.")
	}()
	return nil // no heartbeat channel
}

// doWorkFn returns a ward factory AND a flattened stream of ints produced by each ward instance.
// The ward heartbeats on a ticker and emits the provided ints in order.
// If it sees a negative, it "fails" by returning, which triggers a restart.
func doWorkFn(parent context.Context, ints ...int) (startGoroutineFn, <-chan any) {
	chStreams := make(chan (<-chan any))
	out := bridge(parent, chStreams)

	ward := func(ctx context.Context, pulseInterval time.Duration) <-chan struct{} {
		data := make(chan any)
		hb := make(chan struct{}, 1)

		go func() {
			defer close(data)
			defer close(hb)

			// Publish this ward's output into the bridge.
			select {
			case chStreams <- data:
			case <-ctx.Done():
				return
			}

			t := time.NewTicker(pulseInterval)
			defer t.Stop()

			for {
				// Restarted wards replay `ints` from the start.
				for _, v := range ints {
					if v < 0 {
						fmt.Printf("ward: saw negative %v → simulating failure\n", v)
						return
					}
					for {
						select {
						case <-ctx.Done():
							return
						case <-t.C:
							// Non-blocking heartbeat.
							select {
							case hb <- struct{}{}:
							default:
							}
						case data <- v:
							// emitted one value; move on
							goto next
						}
					}
				next:
				}
			}
		}()

		return hb
	}

	return ward, out
}

func HealingExample() {

	// Demo 1: steward heals an irresponsible ward repeatedly, then we cancel everything.
	{
		fmt.Println("=== Demo 1: healing an irresponsible ward ===")

		steward := newSteward(4*time.Second, irresponsibleWard)

		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(9*time.Second, func() {
			fmt.Println("main: canceling steward and ward")
			cancel()
		})

		// Consume steward heartbeats until canceled (keeps example alive).
		for range steward(ctx, 4*time.Second) {
		}
		fmt.Println("Demo 1 complete")
	}

	// Demo 2: value-producing ward fails, steward restarts it, we read a few values.
	{
		fmt.Println("=== Demo 2: healing a value-producing ward ===")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		wardFactory, intStream := doWorkFn(ctx, 1, 2, -1, 3, 4, 5)
		steward := newSteward(1*time.Millisecond, wardFactory)

		// Start steward; we ignore its own heartbeat here.
		steward(ctx, 500*time.Millisecond)

		for v := range take(ctx, intStream, 6) {
			fmt.Printf("Received: %v\n", v)
		}
		fmt.Println("Demo 2 complete")
	}

}
