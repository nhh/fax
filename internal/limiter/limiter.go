package limiter

import (
	"sync"
	"time"
)

func Start() *RateLimiter {
	rl := RateLimiter{counter: 0, mu: sync.Mutex{}}
	go resetRateLimit(&rl)

	return &rl
}

func resetRateLimit(rl *RateLimiter) {
	for {
		time.Sleep(60 * 5 * time.Second)
		rl.mu.Lock()
		rl.counter = 0
		rl.mu.Unlock()
	}
}

type RateLimiter struct {
	mu      sync.Mutex
	counter int
}

func (limiter *RateLimiter) IsLimited() bool {
	return limiter.counter >= 5
}

func (limiter *RateLimiter) Increment() {
	limiter.mu.Lock()
	limiter.counter++
	limiter.mu.Unlock()
}
