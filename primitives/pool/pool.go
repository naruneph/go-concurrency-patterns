package pool

import (
	"fmt"
	"sync"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"primitives_pool",
		"Demonstrates the use of sync.Pool for managing expensive-to-allocate objects.",
		PoolExample,
	)
}

type payload struct {
	id   int
	data []byte
}

func PoolExample() {
	var cnt int

	var pool = sync.Pool{
		New: func() any {
			cnt++
			fmt.Println("Allocating new Payload:", cnt)
			return &payload{id: cnt, data: make([]byte, 1024)}
		},
	}

	var wg sync.WaitGroup
	wg.Add(5)

	for i := range 5 {
		go func(workerID int) {
			defer wg.Done()
			obj := pool.Get().(*payload)
			fmt.Printf("Worker %d got Payload ID: %d\n", workerID, obj.id)
			// time.Sleep(100 * time.Millisecond)
			pool.Put(obj)
		}(i)
	}

	wg.Wait()
}
