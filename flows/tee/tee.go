package tee

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"flows_tee",
		"Duplicate a stream to two outputs with ctx-aware backpressure and clean shutdown.",
		TeeExample,
	)
}

// Tee duplicates every value from in onto two output channels.
func Tee[T any](ctx context.Context, in <-chan T) (<-chan T, <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}

				// Send v to both outs. We keep trying until both succeed or ctx cancels.
				o1, o2 := out1, out2
				for o1 != nil || o2 != nil {
					select {
					case <-ctx.Done():
						return
					case o1 <- v:
						o1 = nil
					case o2 <- v:
						o2 = nil
					}
				}
			}
		}
	}()

	return out1, out2
}

// Source emits 1..n at interval or stops early if ctx cancels.
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

func TeeExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 650*time.Millisecond)
	defer cancel()

	in := Source(ctx, 10, 80*time.Millisecond)

	a, b := Tee(ctx, in)

	var wg sync.WaitGroup
	wg.Add(2)

	// Consumer A
	go func() {
		defer wg.Done()
		for v := range a {
			fmt.Printf("A: %d\n", v)
		}
		fmt.Println("A: closed")
	}()

	// Consumer B
	go func() {
		defer wg.Done()
		for v := range b {
			time.Sleep(120 * time.Millisecond)
			fmt.Printf("B: %d\n", v)
		}
		fmt.Println("B: closed")
	}()

	wg.Wait()
	fmt.Println("tee: done")
}
