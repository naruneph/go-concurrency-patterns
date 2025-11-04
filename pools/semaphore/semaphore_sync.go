package semaphore

import (
	"context"
	"fmt"
	"golang.org/x/sync/semaphore"
	"sync"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"pools_semaphore_sync",
		"Demonstrates a weighted semaphore from x/sync lib to limit concurrent access based on weight.",
		SemaphoreSyncExample,
	)
}


func weightedWorker(ctx context.Context, id int, weight int64, sem *semaphore.Weighted, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := sem.Acquire(ctx, weight); err != nil {
		fmt.Printf("worker %d: failed to acquire weight %d: %v\n", id, weight, err)
		return
	}
	defer sem.Release(weight)

	fmt.Printf("worker %d: acquired weight %d\n", id, weight)
	time.Sleep(600 * time.Millisecond)
	fmt.Printf("worker %d: releasing weight %d\n", id, weight)
}

func SemaphoreSyncExample() {
	sem := semaphore.NewWeighted(5)
	var wg sync.WaitGroup

	ctx := context.Background()
	tasks := []int64{1, 2, 3, 2, 1, 1}

	wg.Add(len(tasks))
	for i, w := range tasks {
		go weightedWorker(ctx, i, w, sem, &wg)
	}

	wg.Wait()
	fmt.Println("all weighted tasks done")
}
