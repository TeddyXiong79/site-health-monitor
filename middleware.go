package main

import (
	"net"
	"net/http"
	"strings"
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
	// 先用读锁快速检查
	i.mu.RLock()
	if existingLim, exists := i.ips[ip]; exists {
		i.mu.RUnlock()
		// 更新 lastSeen 需要写锁
		i.mu.Lock()
		i.lastSeen[ip] = time.Now()
		i.mu.Unlock()
		return existingLim
	}
	i.mu.RUnlock()

	// 升级为写锁创建新 limiter
	i.mu.Lock()
	defer i.mu.Unlock()
	// 双检锁：可能已被其他 goroutine 创建
	if existingLim, exists := i.ips[ip]; exists {
		i.lastSeen[ip] = time.Now()
		return existingLim
	}
	newLim := rate.NewLimiter(i.r, i.burst)
	i.ips[ip] = newLim
	i.lastSeen[ip] = time.Now()
	return newLim
}

// extractIP 从请求中提取客户端真实 IP（支持反向代理）
func extractIP(r *http.Request) string {
	// 优先检查 X-Real-IP
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// 再检查 X-Forwarded-For（取第一个）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	// 最后用 RemoteAddr，去掉端口号
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func rateLimitMiddleware(limiter *ipLimiter) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			if !limiter.getLimiter(ip).Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
