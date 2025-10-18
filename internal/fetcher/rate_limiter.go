package fetcher

import (
	"context"
	"sync"
	"time"
)

type RateLimiter struct {
	maxConcurrent  int
	rpm            int
	hostSemaphores map[string]*hostLimiter
	mu             sync.RWMutex
}

type hostLimiter struct {
	sem      chan struct{} // Semaphore for concurrency
	lastTime time.Time
	requests int
	mu       sync.Mutex
}

func NewRateLimiter(maxConcurrent, rpm int) *RateLimiter {
	return &RateLimiter{
		maxConcurrent:  maxConcurrent,
		rpm:            rpm,
		hostSemaphores: make(map[string]*hostLimiter),
	}
}

func (rl *RateLimiter) Wait(ctx context.Context, host string) error {
	rl.mu.Lock()
	limiter, exists := rl.hostSemaphores[host]
	if !exists {
		limiter = &hostLimiter{
			sem: make(chan struct{}, rl.maxConcurrent),
		}
		rl.hostSemaphores[host] = limiter
	}
	rl.mu.Unlock()

	// Acquire semaphore (concurrency control)
	select {
	case limiter.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	defer func() { <-limiter.sem }()

	// Apply RPM throttle
	limiter.mu.Lock()
	now := time.Now()

	// Reset counters if minute has passed
	if now.Sub(limiter.lastTime) > time.Minute {
		limiter.requests = 0
		limiter.lastTime = now
	}

	// Check if we've exceeded RPM
	if limiter.requests >= rl.rpm {
		waitTime := time.Minute - now.Sub(limiter.lastTime)
		limiter.mu.Unlock()

		select {
		case <-time.After(waitTime):
			limiter.mu.Lock()
			limiter.requests = 0
			limiter.lastTime = time.Now()
			limiter.requests++
			limiter.mu.Unlock()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	limiter.requests++
	limiter.mu.Unlock()

	return nil
}
