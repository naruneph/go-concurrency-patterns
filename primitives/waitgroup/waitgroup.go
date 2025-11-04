package waitgroup

import (
	"fmt"
	"sync"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"primitives_waitgroup",
		"Demonstrates sync.WaitGroup for waiting for a collection of goroutines to finish.",
		WaitGroupExample,
	)
}

func WaitGroupExample() {
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Println("Goroutine", id, "finished")
		}(i)
	}

	wg.Wait()
}
