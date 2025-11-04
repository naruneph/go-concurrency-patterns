package generator

import (
	"context"
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"flows_generator",
		"Cancellable number generator that emits an arithmetic sequence over a channel.",
		GeneratorExample,
	)
}

func numbers(ctx context.Context, start, step int, interval time.Duration) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := start; ; v += step {
			select {
			case <-ctx.Done():
				return
			case out <- v:
				if interval > 0 {
					timer := time.NewTimer(interval)
					select {
					case <-ctx.Done():
						timer.Stop()
						return
					case <-timer.C:
					}
				}
			}
		}
	}()
	return out
}

func GeneratorExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
	defer cancel()

	seq := numbers(ctx, 1, 1, 100*time.Millisecond)

	for v := range seq {
		fmt.Printf("got %d\n", v)
	}

	fmt.Println("generator: done")
}
