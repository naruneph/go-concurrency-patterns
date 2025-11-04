package fanoutin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"flows_fan_out_fan_in",
		"Fan-out to N workers and fan-in results to a single channel, all ctx-cancellable.",
		FanOutFanInExample,
	)
}

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

func FanOut[T any, U any](ctx context.Context, workerCount int, in <-chan T, fn func(T) U) <-chan U {
	out := make(chan U)

	var wg sync.WaitGroup
	wg.Add(workerCount)

	worker := func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				u := fn(v)

				select {
				case <-ctx.Done():
					return
				case out <- u:
				}
			}
		}
	}

	for range workerCount {
		go worker()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func FanOutFanInExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 650*time.Millisecond)
	defer cancel()

	in := Source(ctx, 20, 60*time.Millisecond)

	out := FanOut(ctx, 4, in, func(v int) int {
		time.Sleep(100 * time.Millisecond)
		return v * v
	})

	// FanIn
	count := 0
	for v := range out {
		fmt.Printf("result: %d\n", v)
		count++
	}

	fmt.Printf("processed: %d\n", count)
	fmt.Println("fanout-fanin: done")
}
