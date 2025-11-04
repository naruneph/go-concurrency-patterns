package heartbeat

import (
	"fmt"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"scale_heartbeat_time",
		"Demonstrates a simple heartbeat mechanism using time.Ticker.",
		HeartbeatTimeExample,
	)
}

func doWork(done <-chan struct{}, pulseInterval time.Duration) (<-chan struct{}, <-chan time.Time) {
	heartbeat := make(chan struct{})
	results := make(chan time.Time)

	go func() {
		defer close(heartbeat)
		defer close(results)

		pulse := time.Tick(pulseInterval)
		workGen := time.Tick(2 * pulseInterval)

		sendPulse := func() {
			select {
			case heartbeat <- struct{}{}:
			default:
			}
		}

		sendResult := func(r time.Time) {
			for {
				select {
				case results <- r:
					return
				case <-pulse:
					sendPulse()
				case <-done:
					return
				}
			}
		}

		for {
			select {
			case <-done:
				return
			case <-pulse:
				sendPulse()
			case r := <-workGen:
				sendResult(r)
			}
		}
	}()

	return heartbeat, results

}

func HeartbeatTimeExample() {
	done := make(chan struct{})
	time.AfterFunc(10*time.Second, func() {
		close(done)
	})
	heartbeat, results := doWork(done, 1*time.Second)

	for {
		select {
		case _, ok := <-heartbeat:
			if !ok {
				fmt.Printf("Done\n")
				return
			}
			fmt.Printf("Pulse\n")

		case r, ok := <-results:
			if !ok {
				fmt.Printf("Done\n")
				return
			}
			fmt.Printf("Result: %v\n", r.Second())

		case <-time.After(2 * time.Second):
			fmt.Printf("Worker seems to be dead\n")
			return
		}
	}
}
