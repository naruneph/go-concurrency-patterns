package ordone

import (
	"context"
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"flows_ordone",
		"Wrap a source channel so consumers can stop early without leaking upstream goroutines.",
		OrDoneExample,
	)
}

// OrDone forwards values from in until ctx is done or in closes.
// It closes the returned channel on exit.
func OrDone[T any](ctx context.Context, in <-chan T) <-chan T {
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

// Source emits 1..n every interval, or exits early if ctx is canceled.
func Source(ctx context.Context, n int, interval time.Duration) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		t := time.NewTicker(interval)
		defer t.Stop()

		for i := 1; i <= n; i++ {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				select {
				case <-ctx.Done():
					return
				case out <- i:
				}
			}
		}
	}()
	return out
}

func OrDoneExample() {
	// Parent governs the source lifetime.
	parent, stop := context.WithCancel(context.Background())
	defer stop()

	src := Source(parent, 10, 80*time.Millisecond)

	// Child context for the consumer; we will cancel after we’ve seen 3.
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	// Wrap the source with OrDone so the consumer can stop early safely.
	stream := OrDone(consumerCtx, src)

	seen := 0
	for v := range stream {
		fmt.Printf("got: %d\n", v)
		seen++
		if seen == 3 {
			// Consumer decides it's done; this cancels the wrapper,
			// which stops reading src and avoids leaking the producer.
			cancelConsumer()
		}
	}

	time.Sleep(50 * time.Millisecond)
	fmt.Println("ordone: consumer stopped early without leaks")
}
