package cond

import (
	"fmt"
	"sync"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"primitives_signal",
		"Demonstrates sync.Cond with Signal to notify one waiting goroutine.",
		SignalExample,
	)
}

type queue struct {
	items []int
	cond  *sync.Cond
}

func newQueue() *queue {
	return &queue{
		items: make([]int, 0),
		cond:  sync.NewCond(&sync.Mutex{}),
	}
}

func (q *queue) push(item int) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.items = append(q.items, item)
	q.cond.Signal() // Notify one waiting goroutine
}

func (q *queue) pop() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait() // Wait for an item to be available
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item
}

func SignalExample() {
	q := newQueue()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range 5 {
			q.push(i)
			fmt.Println("Pushed:", i)
		}
	}()

	go func() {
		defer wg.Done()
		for range 5 {
			item := q.pop()
			fmt.Println("Popped:", item)
		}
	}()

	wg.Wait()
}
