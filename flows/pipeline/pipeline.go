package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"flows_pipeline",
		"Cancellable pipeline with map-like stages wired by channels.",
		PipelineExample,
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

func Map[T any, U any](ctx context.Context, in <-chan T, fn func(T) U) <-chan U {
	out := make(chan U)
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
				u := fn(v)
				select {
				case <-ctx.Done():
					return
				case out <- u:
				}
			}
		}
	}()
	return out
}

func PipelineExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 650*time.Millisecond)
	defer cancel()

	src := Source(ctx, 10, 100*time.Millisecond)

	squared := Map(ctx, src, func(v int) int { return v * v })

	doubled := Map(ctx, squared, func(v int) int { return 2 * v })

	for v := range doubled {
		fmt.Printf("out: %d\n", v)
	}

	fmt.Println("pipeline: done")
}
