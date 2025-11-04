package bridge

import (
	"context"
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"flows_bridge",
		"Flatten a channel-of-channels into a single stream with ctx-aware backpressure.",
		BridgeExample,
	)
}

// ChunkedSource emits nChunks inner streams; each inner stream produces chunkSize ints
// spaced by interval. Stops early if ctx is canceled.
func ChunkedSource(ctx context.Context, nChunks, chunkSize int, interval time.Duration) <-chan <-chan int {
	streams := make(chan (<-chan int))
	go func() {
		defer close(streams)
		for c := 0; c < nChunks; c++ {
			inner := make(chan int)
			base := c * 100
			// Producer for one chunk.
			go func() {
				defer close(inner)
				t := time.NewTicker(interval)
				defer t.Stop()
				for i := 0; i < chunkSize; i++ {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						select {
						case <-ctx.Done():
							return
						case inner <- base + i:
						}
					}
				}
			}()

			select {
			case <-ctx.Done():
				return
			case streams <- inner:
			}
		}
	}()
	return streams
}

// Bridge consumes a stream of streams and emits their values on a single output.
// It stops when ctx is canceled or when the outer stream closes. Output is closed on exit.
func Bridge[T any](ctx context.Context, streams <-chan <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for {
			var current <-chan T
			// Wait for the next inner stream or cancellation.
			select {
			case <-ctx.Done():
				return
			case ch, ok := <-streams:
				if !ok {
					return
				}
				current = ch
			}

			// Drain the current inner stream.
			for current != nil {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-current:
					if !ok {
						current = nil
						break
					}
					// Respect backpressure while forwarding.
					select {
					case <-ctx.Done():
						return
					case out <- v:
					}
				}
			}
		}
	}()
	return out
}

func BridgeExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()

	chs := ChunkedSource(ctx, 3, 5, 80*time.Millisecond)

	flat := Bridge(ctx, chs)

	for v := range flat {
		fmt.Printf("got: %d\n", v)
	}

	fmt.Println("bridge: done")
}
