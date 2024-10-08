package limiter

import (
	"sync"
	"time"
)

func Init() Limiter {
	rl := rateLimiter{counter: 0, mu: sync.Mutex{}}
	go resetRateLimit(&rl)

	return &rl
}

func resetRateLimit(rl *rateLimiter) {
	for {
		time.Sleep(60 * 5 * time.Second)
		rl.mu.Lock()
		rl.counter = 0
		rl.mu.Unlock()
	}
}

type Limiter interface {
	IsLimited() bool
	Increment()
}

type rateLimiter struct {
	mu      sync.Mutex
	counter int
}

func (limiter *rateLimiter) IsLimited() bool {
	return limiter.counter >= 5
}

func (limiter *rateLimiter) Increment() {
	limiter.mu.Lock()
	limiter.counter++
	limiter.mu.Unlock()
}
