package main

import (
	"sync"
	"time"
)

type nodeCache struct {
	mu        sync.RWMutex
	nodes     []OpenClashNode
	lastFetch time.Time
	ttl       time.Duration
}

var globalCache = &nodeCache{
	ttl: 10 * time.Second, // 缓存 10 秒
}

// FetchNodesCached 带缓存的节点数据获取，避免高并发时对上游产生大量重复请求
func FetchNodesCached(cfg Config) ([]OpenClashNode, error) {
	globalCache.mu.RLock()
	if time.Since(globalCache.lastFetch) < globalCache.ttl {
		nodes := globalCache.nodes
		globalCache.mu.RUnlock()
		return nodes, nil
	}
	globalCache.mu.RUnlock()

	// 缓存过期，重新获取
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	// 双检锁：可能已被其他 goroutine 刷新
	if time.Since(globalCache.lastFetch) < globalCache.ttl {
		return globalCache.nodes, nil
	}

	nodes, err := FetchNodes(cfg)
	if err != nil {
		// 失败时不缓存，下次请求立即重试
		return nodes, err
	}
	globalCache.nodes = nodes
	globalCache.lastFetch = time.Now()
	return nodes, nil
}

// InvalidateCache 使缓存失效（配置变更、手动刷新后调用）
func InvalidateCache() {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.lastFetch = time.Time{}
}
