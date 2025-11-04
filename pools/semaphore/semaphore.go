package semaphore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"pools_semaphore",
		"A simple semaphore implementation to limit concurrent access.",
		SemaphoreExample,
	)
}

// Errors returned by the semaphore.
var (
	ErrClosed                = errors.New("semaphore: closed")
	ErrReleaseWithoutAcquire = errors.New("semaphore: release without acquire")
)

type Semaphore struct {
	tokens   chan struct{}
	capacity int

	mu     sync.Mutex
	closed bool
}

// NewSemaphore creates a semaphore with capacity n > 0.
func NewSemaphore(capacity int) *Semaphore {
	if capacity <= 0 {
		panic("semaphore capacity must be > 0")
	}
	return &Semaphore{
		tokens:   make(chan struct{}, capacity),
		capacity: capacity,
	}
}

// Acquire blocks until a token is available, context is done, or semaphore is closed.
func (s *Semaphore) Acquire(ctx context.Context) error {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return ErrClosed
	}

	select {
	case s.tokens <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire a token immediately.
// Returns true if acquired, false if none available. Returns error if closed.
func (s *Semaphore) TryAcquire() (bool, error) {
	s.mu.Lock()
	closed := s.closed
	defer s.mu.Unlock()
	if closed {
		return false, ErrClosed
	}
	select {
	case s.tokens <- struct{}{}:
		return true, nil
	default:
		return false, nil
	}
}



// Release returns one token to the semaphore.
// Returns ErrReleaseWithoutAcquire if Release would exceed capacity.
// Returns ErrClosed only if the semaphore was closed.
func (s *Semaphore) Release() error{
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return ErrClosed
	}

	select {
	case <-s.tokens:
		return nil
	default:
		return ErrReleaseWithoutAcquire
	}
}

// With runs fn after acquiring a token, and releases it afterward.
func (s *Semaphore) With(ctx context.Context, fn func(context.Context) error) (err error) {
	if err := s.Acquire(ctx); err != nil {
		return err
	}

	defer func() {
		relErr := s.Release()

		if r := recover(); r != nil {
			pErr := fmt.Errorf("panic in semaphore.With: %v", r)
			err = errors.Join(pErr, relErr)
			return
		}

		err = errors.Join(err, relErr)
	}()

	err = fn(ctx)
	return
}

// Close closes the semaphore, causing future Acquire and TryAcquire calls to fail.
// It does not forcefully release tokens currently held by callers; those callers
// should still call Release as usual.
func (s *Semaphore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
}


func SemaphoreExample() {
	rootCtx, cancelRootCtx := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancelRootCtx()

	sem := NewSemaphore(3)

	var inFlight int32
	task := func(context.Context) error {
		cur := atomic.AddInt32(&inFlight, 1)
		fmt.Printf("work: start (inFlight=%d)\n", cur)
		time.Sleep(150 * time.Millisecond)
		cur = atomic.AddInt32(&inFlight, -1)
		fmt.Printf("work: done  (inFlight=%d)\n", cur)
		return nil
	}

	// Nonblocking burst: accept if a token is free, otherwise reject fast.
	accepted, rejected := 0, 0
	for range 12 {
		ok, err := sem.TryAcquire()
		if err != nil{
			fmt.Println("try acquire error:", err)
			rejected++
			continue
		}
		if ok {
			accepted++
			go func() {
				if err := task(rootCtx); err != nil {
					fmt.Println("task error:", err)
				}
				if err := sem.Release(); err != nil {
					fmt.Println("release error:", err)
				}
			}()
		} else {
			rejected++
		}
	}
	fmt.Printf("submitted: %d, accepted immediately: %d, rejected fast: %d\n", 12, accepted, rejected)

	// Blocking submissions with a per-call deadline.
	waitAccepted := 0
	for range 6 {
		taskCtx, cancelTaskCtx := context.WithTimeout(rootCtx, 100*time.Millisecond)
		err := sem.With(taskCtx, task)
		cancelTaskCtx()
		if err == nil {
			waitAccepted++
		} else {
			fmt.Println("submit (wait):", err)
		}
	}

	<-rootCtx.Done()
	fmt.Printf("wait-accepted: %d\n", waitAccepted)
	sem.Close()
	fmt.Println("semaphore: done")
}
