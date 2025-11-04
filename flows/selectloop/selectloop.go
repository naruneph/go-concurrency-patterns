package selectloop

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)



func init() {
	registry.Register(
		"flows_selectloop",
		"Cancellable for-select worker loop that stops on context cancel or input close.",
		SelectLoopExample,
	)
}

func SelectLoopExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	in := make(chan int)
	var wg sync.WaitGroup

	// Worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				fmt.Println("worker: ctx canceled")
				return
			case v, ok := <-in:
				if !ok {
					fmt.Println("worker: input closed")
					return
				}
				fmt.Printf("worker: got %d\n", v)
			}
		}
	}()

	// Producer
	go func() {
		defer close(in)
		for i := 1; i <= 10; i++ {
			in <- i
			time.Sleep(100 * time.Millisecond)
		}
	}()

	wg.Wait()
	fmt.Println("selectloop: done")
}
