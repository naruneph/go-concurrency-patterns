package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"scale_ratelimit",
		"Demonstrates a simple token-bucket rate limiter.",
		RateLimitExample,
	)
}

// Limit is tokens per second. A zero Limit allows no events.
type Limit float64

// Per(10, 1s) == 10 tokens/sec; Per(1, 3s) == 0.333... tokens/sec.
func Per(eventCount int, duration time.Duration) Limit {
	return Limit(float64(eventCount) / duration.Seconds())
}

// Limiter is a token-bucket limiter.
type Limiter struct {
	mu     sync.Mutex
	rate   Limit      // tokens per second
	burst  int        // bucket capacity (max tokens)
	tokens float64    // current tokens
	last   time.Time  // last refill timestamp
}

// NewLimiter allows bursts up to b. If r == 0, no events are ever allowed.
func NewLimiter(r Limit, b int) *Limiter {
	now := time.Now()
	toks := float64(b)
	if r <= 0 {
		r = 0
		b = 0
		toks = 0 // start empty
	}
	return &Limiter{
		rate:   r,
		burst:  b,
		tokens: toks,
		last:   now,
	}
}

// advance refills tokens based on elapsed time.
func (l *Limiter) advance(now time.Time) {
	if l.last.IsZero() {
		l.last = now
		return
	}
	elapsed := now.Sub(l.last).Seconds()
	if elapsed <= 0 {
		return
	}
	l.tokens += float64(l.rate) * elapsed
	max := float64(l.burst)
	if l.tokens > max {
		l.tokens = max
	}
	l.last = now
}

// WaitN blocks until n events are permitted, or ctx is done, or the
// predicted wait exceeds ctx.Deadline. Returns an error in those cases.
func (l *Limiter) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}
	if n > l.burst {
		return fmt.Errorf("n exceeds burst size")
	}
	if l.rate <= 0 {
		return fmt.Errorf("rate is zero; no events permitted")
	}

	l.mu.Lock()
	for {
		now := time.Now()
		l.advance(now)

		// Enough tokens now?
		if l.tokens >= float64(n) {
			l.tokens -= float64(n)
			l.mu.Unlock()
			return nil
		}

		// Compute wait for the deficit.
		need := float64(n) - l.tokens
		wait := time.Duration(need/float64(l.rate)*float64(time.Second) + 0.5) // round sensibly

		// If deadline exists and we can't possibly make it, fail fast.
		if dl, ok := ctx.Deadline(); ok && now.Add(wait).After(dl) {
			l.mu.Unlock()
			return context.DeadlineExceeded
		}

		// Sleep without holding the lock; be cancel-aware.
		l.mu.Unlock()
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
			// try again; tokens will be refilled on next loop
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		}
		l.mu.Lock()
	}
}

// Wait is shortcut for WaitN(ctx, 1).
func (l *Limiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}



func RateLimitExample() {
	// 1 event every 3s, burst 3: first 3 fire immediately, then ~3s each.
	limiter := NewLimiter(Per(1, time.Second*3), 3)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	for i := range 10 {
		if err := limiter.Wait(ctx); err != nil {
			fmt.Println("blocked:", err)
			continue
		}
		fmt.Printf("%s event %d allowed\n", time.Now().Format("15:04:05.000"), i)
	}
}