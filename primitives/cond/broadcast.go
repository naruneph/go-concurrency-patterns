package cond

import (
	"fmt"
	"sync"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"primitives_broadcast",
		"Demonstrates sync.Cond with Broadcast to notify all waiting goroutines.",
		BroadcastExample,
	)
}

func BroadcastExample() {
	button := sync.NewCond(&sync.Mutex{})
	var wg sync.WaitGroup
	wg.Add(3)

	subscribe := func(b *sync.Cond, fn func()) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			wg.Done()
			b.L.Lock()
			defer b.L.Unlock()
			b.Wait()
			fn()
		}()
		wg.Wait()
	}

	subscribe(button, func() {
		println("Listener 1: Button clicked!")
		wg.Done()
	})
	subscribe(button, func() {
		println("Listener 2: Button clicked!")
		wg.Done()
	})
	subscribe(button, func() {
		println("Listener 3: Button clicked!")
		wg.Done()
	})

	fmt.Println("Clicking the button...")
	button.Broadcast()

	wg.Wait()

	fmt.Println("All listeners have been notified.")
}
