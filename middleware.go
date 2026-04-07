package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	ips      map[string]*rate.Limiter
	lastSeen map[string]time.Time
	mu       sync.RWMutex
	r        rate.Limit
	burst    int
	maxIdle  time.Duration // 超过此时间未访问的 entry 会被清理
	stop     chan struct{}
	wg       sync.WaitGroup
}

func newIPLimiter(r rate.Limit, burst int) *ipLimiter {
	limiter := &ipLimiter{
		ips:      make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		r:        r,
		burst:    burst,
		maxIdle:  240 * time.Minute,
		stop:     make(chan struct{}),
	}
	limiter.wg.Add(1)
	go limiter.cleanupLoop()
	return limiter
}

func (i *ipLimiter) cleanupLoop() {
	defer i.wg.Done()
	ticker := time.NewTicker(60 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			i.cleanupStale()
		case <-i.stop:
			return
		}
	}
}

func (i *ipLimiter) cleanupStale() {
	i.mu.Lock()
	defer i.mu.Unlock()
	now := time.Now()
	for ip, lastSeen := range i.lastSeen {
		if now.Sub(lastSeen) > i.maxIdle {
			delete(i.ips, ip)
			delete(i.lastSeen, ip)
		}
	}
}

// Stop 关闭限流器的后台清理 goroutine
func (i *ipLimiter) Stop() {
	close(i.stop)
	i.wg.Wait()
}

func (i *ipLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()
	if existingLim, exists := i.ips[ip]; exists {
		i.lastSeen[ip] = time.Now()
		return existingLim
	}
	newLim := rate.NewLimiter(i.r, i.burst)
	i.ips[ip] = newLim
	i.lastSeen[ip] = time.Now()
	return newLim
}

func rateLimitMiddleware(limiter *ipLimiter) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if !limiter.getLimiter(ip).Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
