package errgroupx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
	"golang.org/x/sync/errgroup"
)

func init() {
	registry.Register(
		"orchestration_errgroup",
		"Errgroup usage to run a group of goroutines with error propagation and cancellation.",
		ExampleErrGroup,
	)
}

func pretendWork(ctx context.Context, name string, d time.Duration, fail bool) error {
	select {
	case <-time.After(d):
		if fail {
			return errors.New(name + ": failed")
		}
		fmt.Println(name, "ok")
		return nil
	case <-ctx.Done():
		// someone else failed or overall deadline hit
		fmt.Println(name, "canceled:", ctx.Err())
		return ctx.Err()
	}
}

func ExampleErrGroup() {
	root, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	g, ctx := errgroup.WithContext(root)
	g.SetLimit(2)

	// A finishes fine
	g.Go(func() error { return pretendWork(ctx, "A", 400*time.Millisecond, false) })
	// B fails
	g.Go(func() error { return pretendWork(ctx, "B", 600*time.Millisecond, true) })
	// C would take long, but should get canceled when B fails
	g.Go(func() error { return pretendWork(ctx, "C", 1200*time.Millisecond, false) })

	// 3) Wait for completion; get first non-nil error
	if err := g.Wait(); err != nil {
		fmt.Println("group error:", err)
	} else {
		fmt.Println("group: all ok")
	}
}
