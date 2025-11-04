package once

import (
	"fmt"
	"sync"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"primitives_once",
		"Demonstrates sync.Once for executing a function only once.",
		OnceExample,
	)
}

func OnceExample() {
	var once sync.Once

	var wg sync.WaitGroup
	wg.Add(3)

	for i := range 3 {
		go func(id int) {
			defer wg.Done()
			once.Do(func() {
				fmt.Println("Function executed by goroutine", id)
			})
			fmt.Println("Goroutine", id, "finished")
		}(i)
	}

	wg.Wait()
}
