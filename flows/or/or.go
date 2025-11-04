package or

import (
	"context"
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"flows_or",
		"Combine multiple Done channels into one that closes when any of them closes.",
		OrExample,
	)
}

func Or(channels ...<-chan struct{}) <-chan struct{} {
	switch len(channels) {
	case 0:
		return nil
	case 1:
		return channels[0]
	}

	orDone := make(chan struct{})
	go func() {
		defer close(orDone)

		switch len(channels) {
		case 2:
			select {
			case <-channels[0]:
			case <-channels[1]:
			}
		default:
			select {
			case <-channels[0]:
			case <-channels[1]:
			case <-channels[2]:
			case <-Or(append(channels[3:], orDone)...):
			}
		}
	}()
	return orDone
}

// OrContext is Or plus a context. It closes when ctx is canceled or any channel closes.
func OrContext(ctx context.Context, channels ...<-chan struct{}) <-chan struct{} {
	all := append(channels, ctx.Done())
	return Or(all...)
}

func After(d time.Duration) <-chan struct{} {
	c := make(chan struct{})
	time.AfterFunc(d, func() { close(c) })
	return c
}

func OrExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()

	a := After(900 * time.Millisecond)
	b := After(400 * time.Millisecond)
	c := After(1000 * time.Millisecond)

	done := OrContext(ctx, a, b, c)

	start := time.Now()
	<-done
	elapsed := time.Since(start).Round(10 * time.Millisecond)

	fmt.Printf("or-channel fired after %v\n", elapsed)
	fmt.Println("or: done")
}
