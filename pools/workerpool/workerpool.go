package workerpool

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
		"pools_workerpool",
		"Fixed-size worker pool with a bounded queue.",
		WorkerPoolExample,
	)
}


// Job is the unit of work run by workers.
type Job func(ctx context.Context) error

// ErrClosed is returned when submitting to a closed pool.
var ErrClosed = errors.New("pool: closed")

// Pool runs Jobs on a fixed set of worker goroutines reading from a bounded queue.
type Pool struct {
	queue     chan Job        // bounded buffer of jobs
	ctx       context.Context // pool lifetime context
	cancel    context.CancelFunc
	wg        sync.WaitGroup  // waits for workers to exit
	closeOnce sync.Once
	closedCh  chan struct{} // closed in Close(); submitters wake immediately
	mu        sync.Mutex    // serializes queue send vs close to avoid send-after-close panics
}

// NewPool creates a pool with `workers` workers and a bounded queue of size `queueSize`.
// If queueSize == 0, Submit will block until a worker is ready (no buffering).
func NewPool(workersNum, queueSize int) *Pool {
	if workersNum <= 0 {
		panic("workersNum must be > 0")
	}
	if queueSize < 0 {
		panic("queueSize must be >= 0")
	}

	queue := make(chan Job, queueSize)
	ctx, cancel := context.WithCancel(context.Background())

	p := &Pool{
		queue:    queue,
		ctx:      ctx,
		cancel:   cancel,
		closedCh: make(chan struct{}),
	}

	// Start fixed workers.
	p.wg.Add(workersNum)
	for i := 0; i < workersNum; i++ {
		go p.worker(i)
	}
	return p
}

// worker is the loop run by each worker goroutine.
func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.queue:
			if !ok {
				// queue closed and drained
				return
			}
			// Shield the pool from job panics.
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("worker %d: job panic: %v\n", id, r)
					}
				}()
				err := job(p.ctx)
				if err != nil {
					fmt.Printf("worker %d: job error: %v\n", id, err)
				}
			}()
		}
	}
}

// TrySubmit attempts to enqueue a job immediately without blocking.
// Returns (accepted, error). Error is ErrClosed if the pool is closed.
func (p *Pool) TrySubmit(job Job) (bool, error) {
	select {
	case <-p.closedCh:
		return false, ErrClosed
	default:
	}

	// Serialize send vs close.
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.closedCh:
		return false, ErrClosed
	default:
	}

	select {
	case p.queue <- job:
		return true, nil
	default:
		return false, nil
	}
}

// SubmitWithContext enqueues the job, blocking until there is queue space,
// the provided ctx is done, or the pool is closed.
func (p *Pool) SubmitWithContext(ctx context.Context, job Job) error {
	for {
		// Fast-fail checks without holding the lock.
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
			return nil
		default:
			// No space right now; release the lock and wait briefly before retry.
			p.mu.Unlock()
			time.Sleep(100 * time.Microsecond)
		}
	}
}

// Submit blocks until the job is enqueued or the pool is closed.
func (p *Pool) Submit(job Job) error {
	return p.SubmitWithContext(context.Background(), job)
}

// Close stops accepting new jobs, cancels the pool context (so jobs can observe it),
// closes the queue to signal workers to drain, and is idempotent.
func (p *Pool) Close() {
	p.closeOnce.Do(func() {
		close(p.closedCh) // wake SubmitWithContext/TrySubmit waiters first
		p.cancel()        // tell workers/jobs to bail if they honor ctx
		p.mu.Lock()
		close(p.queue)    // safe: no concurrent sends now
		p.mu.Unlock()
	})
}

// Wait blocks until all workers have exited. Call after Close() to wait for shutdown.
func (p *Pool) Wait() {
	p.wg.Wait()
}

func ts() string { 
    return time.Now().Format("15:04:05.000") 
}

func WorkerPoolExample() {
	rootCtx, cancelRootCtx := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancelRootCtx()

	p := NewPool(3, 5)
	var inFlight int32
	task := func(id int, dur time.Duration) Job {
		return func(ctx context.Context) error {
			cur := atomic.AddInt32(&inFlight, 1)
			fmt.Printf("%s task %02d start (inFlight=%d)\n", ts(), id, cur)
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

	// Burst: nonblocking try-submits.
	accepted, rejected := 0, 0
	for i := 0; i < 12; i++ {
		ok, err := p.TrySubmit(task(i, 400*time.Millisecond))
		if err != nil {
			fmt.Println(ts(), "try submit error:", err)
			continue
		}
		if ok {
			accepted++
		} else {
			rejected++
			fmt.Printf("%s task %02d rejected (queue full)\n", ts(), i)
		}
	}
	fmt.Printf("submitted: 12, accepted immediately: %d, rejected fast: %d\n", accepted, rejected)

	// Blocking submissions with a per-submit deadline.
	waitAccepted := 0
	for i := 12; i < 18; i++ {
		ctx, cancel := context.WithTimeout(rootCtx, 250*time.Millisecond)
		err := p.SubmitWithContext(ctx, task(i, 300*time.Millisecond))
		cancel()
		if err == nil {
			waitAccepted++
			fmt.Printf("%s submit %02d accepted\n", ts(), i)
		} else {
			fmt.Printf("%s submit %02d failed: %v\n", ts(), i, err)
		}
	}

	// Let things run a bit, then shut down cleanly.
	<-rootCtx.Done()
	p.Close()
	p.Wait()
	fmt.Println("pool: shutdown complete")
}
