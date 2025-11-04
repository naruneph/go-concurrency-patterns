package shutdown

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"orchestration_shutdown",
		"Graceful shutdown with phases: stop intake, soft drain, hard cancel.",
		ShutdownExample,
	)
}

type Service struct {
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	jobs    chan int
	closed  chan struct{}
	workers int
}

// NewService spins up N workers that respect ctx and read from jobs.
func NewService(workers int, queue int) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		ctx:     ctx,
		cancel:  cancel,
		jobs:    make(chan int, queue),
		closed:  make(chan struct{}),
		workers: workers,
	}
	s.start()
	return s
}

func (s *Service) start() {
	s.wg.Add(s.workers)
	for i := 0; i < s.workers; i++ {
		go func(id int) {
			defer s.wg.Done()
			for {
				select {
				case <-s.ctx.Done():
					fmt.Printf("worker %d: canceled\n", id)
					return
				case j, ok := <-s.jobs:
					if !ok {
						fmt.Printf("worker %d: intake closed, exit\n", id)
						return
					}
					// Simulate per-item work that checks ctx periodically.
					fmt.Printf("worker %d: processing %d\n", id, j)
					step := time.NewTimer(120 * time.Millisecond)
					select {
					case <-s.ctx.Done():
						step.Stop()
						return
					case <-step.C:
					}
				}
			}
		}(i + 1)
	}
}

// Submit puts a job in the queue (non-blocking just for demo).
func (s *Service) Submit(j int) bool {
	select {
	case <-s.closed:
		return false
	default:
	}
	select {
	case s.jobs <- j:
		return true
	default:
		return false
	}
}

// Shutdown does:
// 1) Stop intake (close jobs)
// 2) Soft drain: wait up to softTimeout for workers to finish
// 3) Hard cancel: cancel ctx, then wait up to hardTimeout
func (s *Service) Shutdown(softTimeout, hardTimeout time.Duration) {
	// Phase 1: stop intake so backlog can drain.
	select {
	case <-s.closed:
		// already closed
	default:
		close(s.closed)
		close(s.jobs)
		fmt.Println("shutdown: intake closed")
	}

	// Phase 2: soft drain (no new work, let current work finish)
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()

	select {
	case <-done:
		fmt.Println("shutdown: drained gracefully")
		return
	case <-time.After(softTimeout):
		fmt.Println("shutdown: soft timeout, escalating to hard cancel")
	}

	// Phase 3: hard cancel; tell workers to bail
	s.cancel()

	select {
	case <-done:
		fmt.Println("shutdown: canceled and joined workers")
	case <-time.After(hardTimeout):
		fmt.Println("shutdown: hard timeout; some workers may still be exiting")
	}
}

func ShutdownExample() {
	svc := NewService(3, 4)

	for i := 1; i <= 10; i++ {
		if ok := svc.Submit(i); !ok {
			fmt.Printf("submit %d: queue full or closed\n", i)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Begin graceful shutdown:
	// give 400ms to drain whatever is in-flight,
	// then cancel and give 300ms to force-exit.
	svc.Shutdown(400*time.Millisecond, 300*time.Millisecond)

	fmt.Println("shutdown: done")
}
