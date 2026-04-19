package main

import (
	"fmt"
	"sync"
	"time"
)

type nodeCache struct {
	mu        sync.RWMutex
	nodes     []OpenClashNode
	lastFetch time.Time
	cacheKey  string // 缓存所属的 cfg 键（address:port），不匹配则视为缓存失效
	ttl       time.Duration
}

var globalCache = &nodeCache{
	ttl: 10 * time.Second, // 缓存 10 秒
}

// cacheKeyOf 根据 cfg 生成缓存键，避免数据源变更后返回旧缓存
func cacheKeyOf(cfg Config) string {
	return fmt.Sprintf("%s:%s", cfg.APIAddress, cfg.APISourcePort)
}

// FetchNodesCached 带缓存的节点数据获取，避免高并发时对上游产生大量重复请求
func FetchNodesCached(cfg Config) ([]OpenClashNode, error) {
	key := cacheKeyOf(cfg)

	globalCache.mu.RLock()
	if globalCache.cacheKey == key && time.Since(globalCache.lastFetch) < globalCache.ttl {
		nodes := globalCache.nodes
		globalCache.mu.RUnlock()
		return nodes, nil
	}
	globalCache.mu.RUnlock()

	// 缓存过期或数据源变更，重新获取
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	// 双检锁：可能已被其他 goroutine 刷新
	if globalCache.cacheKey == key && time.Since(globalCache.lastFetch) < globalCache.ttl {
		return globalCache.nodes, nil
	}

	nodes, err := FetchNodes(cfg)
	if err != nil {
		// 失败时不缓存，下次请求立即重试
		return nodes, err
	}
	globalCache.nodes = nodes
	globalCache.lastFetch = time.Now()
	globalCache.cacheKey = key
	return nodes, nil
}

// InvalidateCache 使缓存失效（配置变更、手动刷新后调用）
func InvalidateCache() {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.lastFetch = time.Time{}
	globalCache.cacheKey = ""
}
