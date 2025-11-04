package heartbeat

import (
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"scale_heartbeat_workunit",
		"Demonstrates a heartbeat mechanism using work units.",
		HeartbeatWorkunitExample,
	)
}

func worker(done <-chan struct{}) (<-chan struct{}, <-chan int) {
	heartbeat := make(chan struct{}, 1)
	results := make(chan int)

	go func() {
		defer close(heartbeat)
		defer close(results)

		for i := range 10 {

			select {
			case <-done:
				return
			case heartbeat <- struct{}{}:
			default:
			}

			// Simulate a unit of work
			time.Sleep(500 * time.Millisecond)

			select {
			case <-done:
				return
			case results <- i:
			}

		}
	}()

	return heartbeat, results
}

func HeartbeatWorkunitExample() {
	done := make(chan struct{})
	defer close(done)

	heartbeat, results := worker(done)

	for {
		select {
		case _, ok := <-heartbeat:
			if !ok {
				return
			}
			println("pulse")
		case r, ok := <-results:
			if !ok {
				return
			}
			println("result:", r)
		}
	}
}
