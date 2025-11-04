package atomicx

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"primitives_atomic",
		"Demonstrates atomic operations using sync/atomic package.",
		AtomicExample,
	)
}

func AtomicExample() {
	var counter int64
	var wg sync.WaitGroup

	wg.Add(5)
	for range 5 {
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				atomic.AddInt64(&counter, 1)
			}
		}()
	}

	wg.Wait()
	fmt.Printf("Final Counter Value: %d\n", counter)
}
