package workerpooldyn

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
		"pools_workerpool_dynamic",
		"Dynamic-size worker pool with a bounded queue.",
		WorkerPoolDynExample,
	)
}

// Job is the unit of work run by workers.
type Job func(ctx context.Context) error

// ErrClosed is returned when submitting to a closed pool.
var ErrClosed = errors.New("pool: closed")

// DynPool runs Jobs on a fixed set of worker goroutines reading from a bounded queue.
type DynPool struct {
	queue       chan Job
	minWorkers  int
	maxWorkers  int
	idleTimeout time.Duration

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once

	closedCh chan struct{}
	mu       sync.Mutex

	workers int32
}

// NewDynPool creates a dynamic worker pool.
// - minWorkers: minimum number of workers kept alive
// - maxWorkers: hard cap on workers (>= minWorkers)
// - queueSize:  bounded queue capacity (>= 0)
// - idle:       worker exits after being idle for this long, as long as workers > minWorkers
func NewDynPool(minWorkers, maxWorkers, queueSize int, idle time.Duration) *DynPool {
	if minWorkers <= 0 {
		panic("minWorkers must be > 0")
	}
	if maxWorkers < minWorkers {
		panic("maxWorkers must be >= minWorkers")
	}
	if queueSize < 0 {
		panic("queueSize must be >= 0")
	}
	if idle <= 0 {
		panic("idleTimeout must be > 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &DynPool{
		queue:       make(chan Job, queueSize),
		minWorkers:  minWorkers,
		maxWorkers:  maxWorkers,
		idleTimeout: idle,
		ctx:         ctx,
		cancel:      cancel,
		closedCh:    make(chan struct{}),
	}

	// Start min workers.
	for i := 0; i < minWorkers; i++ {
		p.spawnWorker()
	}
	return p
}

// spawnWorker increments the worker count and starts a worker goroutine.
func (p *DynPool) spawnWorker() {
	cur := atomic.AddInt32(&p.workers, 1)
	p.wg.Add(1)
	go p.worker(int(cur))
}

// worker runs jobs until the pool context is canceled, the queue is closed and drained,
// or the worker has been idle for idleTimeout and worker count > minWorkers.
func (p *DynPool) worker(id int) {
	defer p.wg.Done()

	idleTimer := time.NewTimer(p.idleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case <-p.ctx.Done():
			p.decWorker()
			return
		case job, ok := <-p.queue:
			if !ok {
				// Queue closed and drained.
				p.decWorker()
				return
			}
			// Got work: we are no longer idle; reset the timer after running the job.
			// Shield the pool from job panics.
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("worker %02d: job panic: %v\n", id, r)
					}
				}()
				err := job(p.ctx)
				if err != nil {
					fmt.Printf("worker %02d: job error: %v\n", id, err)
				}
			}()
			// Reset idle timer after doing work
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(p.idleTimeout)

		case <-idleTimer.C:
			// Timer fired: it has been idle long enough.
			for {
				cur := atomic.LoadInt32(&p.workers)
				if int(cur) <= p.minWorkers {
					// Keep waiting. Reset timer.
					idleTimer.Reset(p.idleTimeout)
					break
				}
				// Try to drop ourselves.
				if atomic.CompareAndSwapInt32(&p.workers, cur, cur-1) {
					return
				}
				// Re-check.
			}
		}
	}
}

func (p *DynPool) decWorker() {
	atomic.AddInt32(&p.workers, -1)
}

// TrySubmit attempts to enqueue a job immediately without blocking.
// Returns (accepted, error). Error is ErrClosed if the pool is closed.
func (p *DynPool) TrySubmit(job Job) (bool, error) {
	select {
	case <-p.closedCh:
		return false, ErrClosed
	default:
	}

	// Serialize send vs close to avoid "send on closed channel".
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.closedCh:
		return false, ErrClosed
	default:
	}
	select {
	case p.queue <- job:
		p.maybeScaleUp()
		return true, nil
	default:
		return false, nil
	}
}

// SubmitWithContext enqueues the job, blocking until there is queue space,
// the provided ctx is done, or the pool is closed.
func (p *DynPool) SubmitWithContext(ctx context.Context, job Job) error {
	for {
		// Quick exits
		select {
		case <-p.closedCh:
			return ErrClosed
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to enqueue under the lock to avoid racing Close().
		p.mu.Lock()
		select {
		case <-p.closedCh:
			p.mu.Unlock()
			return ErrClosed
		default:
		}
		select {
		case p.queue <- job:
			p.mu.Unlock()
			p.maybeScaleUp()
			return nil
		default:
			p.mu.Unlock()
			// No slot right now; wait on either ctx or a short tick to retry.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-p.closedCh:
				return ErrClosed
			case <-time.After(time.Millisecond):
			}
		}
	}
}

// Submit blocks until the job is enqueued or the pool is closed.
func (p *DynPool) Submit(job Job) error {
	return p.SubmitWithContext(context.Background(), job)
}

// maybeScaleUp spawns at most one worker if backlog exists and we are under max.
func (p *DynPool) maybeScaleUp() {
	cur := atomic.LoadInt32(&p.workers)
	if int(cur) >= p.maxWorkers {
		return
	}
	// Heuristic: if there's backlog, consider adding a worker.
	if len(p.queue) > 0 {
		// Best-effort; if another goroutine spawns concurrently, we may overshoot by 1
		// in theory, but CompareAndSwap in worker reap keeps bounds sane over time.
		if atomic.AddInt32(&p.workers, 1) <= int32(p.maxWorkers) {
			p.wg.Add(1)
			go p.worker(int(cur) + 1)
			return
		}
		// If we incremented past max, decrement back.
		atomic.AddInt32(&p.workers, -1)
	}
}

// Close stops accepting new jobs, cancels the pool context (so jobs can observe it),
// closes the queue to signal workers to drain, and is idempotent.
func (p *DynPool) Close() {
	p.closeOnce.Do(func() {
		close(p.closedCh) // wake submitters first
		p.cancel()        // tell workers/jobs to bail
		p.mu.Lock()
		close(p.queue) // safe: no concurrent send can pass this lock
		p.mu.Unlock()
	})
}

// Wait blocks until all workers have exited.
func (p *DynPool) Wait() {
	p.wg.Wait()
}

func ts() string { return time.Now().Format("15:04:05.000") }

func WorkerPoolDynExample() {
	p := NewDynPool(2, 6, 8, 500*time.Millisecond)

	var inFlight int32
	task := func(id int, dur time.Duration) Job {
		return func(ctx context.Context) error {
			cur := atomic.AddInt32(&inFlight, 1)
			fmt.Printf("%s task %02d start (inFlight=%d, workers=%d, qlen=%d)\n",
				ts(), id, cur, atomic.LoadInt32(&p.workers), len(p.queue))
			select {
			case <-time.After(dur):
			case <-ctx.Done():
				fmt.Printf("%s task %02d canceled by pool ctx\n", ts(), id)
			}
			cur = atomic.AddInt32(&inFlight, -1)
			fmt.Printf("%s task %02d done  (inFlight=%d)\n", ts(), id, cur)
			return nil
		}
	}

	// Burst 1: small, should be handled by min workers, maybe +1 due to backlog.
	for i := 0; i < 4; i++ {
		_, _ = p.TrySubmit(task(i, 300*time.Millisecond))
	}

	time.Sleep(250 * time.Millisecond)

	// Burst 2: big spike → queue grows → pool scales up toward max.
	for i := 4; i < 20; i++ {
		_, _ = p.TrySubmit(task(i, 450*time.Millisecond))
	}

	// A few blocking submissions with timeout.
	accepted := 0
	for i := 20; i < 26; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		if err := p.SubmitWithContext(ctx, task(i, 300*time.Millisecond)); err == nil {
			accepted++
		} else {
			fmt.Printf("%s submit %02d failed: %v\n", ts(), i, err)
		}
		cancel()
	}
	fmt.Printf("blocking accepted: %d\n", accepted)

	// Let things run so we can see idle reaping.
	time.Sleep(2 * time.Second)
	p.Close()
	p.Wait()
	fmt.Println("dynpool: shutdown complete")
}
