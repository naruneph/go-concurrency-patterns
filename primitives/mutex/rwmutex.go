package mutex

import (
	"fmt"
	"sync"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"primitives_rwmutex",
		"Demonstrates sync.RWMutex for concurrent read/write access to a map.",
		RWMutexExample,
	)
}

type safeMap[C comparable, T any] struct {
	mu sync.RWMutex
	m  map[C]T
}

func newSafeMap[C comparable, T any]() *safeMap[C, T] {
	return &safeMap[C, T]{
		m: make(map[C]T),
	}
}

func (sm *safeMap[C, T]) Get(key C) (T, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.m[key]
	return val, ok
}

func (sm *safeMap[C, T]) Set(key C, value T) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = value
}

func RWMutexExample() {
	sm := newSafeMap[string, int]()

	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 5 {
			sm.Set(fmt.Sprintf("key%d", i), i)
			fmt.Printf("Set %s to %d\n", fmt.Sprintf("key%d", i), i)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Readers
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 5 {
				if val, ok := sm.Get(fmt.Sprintf("key%d", j)); ok {
					fmt.Printf("Goroutine %d got %s: %d\n", id, fmt.Sprintf("key%d", j), val)
				} else {
					fmt.Printf("Goroutine %d: %s not found\n", id, fmt.Sprintf("key%d", j))
				}
				time.Sleep(150 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
}
