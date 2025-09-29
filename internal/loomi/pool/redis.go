package pool

import (
	"context"
	"sync"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/redis/go-redis/v9"
)

// RedisManager Redis连接池管理器
type RedisManager struct {
	clients map[string]*redis.Client
	mu      sync.RWMutex
	logger  *logx.Logger
}

// NewRedisManager 创建Redis连接池管理器
func NewRedisManager(logger *logx.Logger) *RedisManager {
	return &RedisManager{
		clients: make(map[string]*redis.Client),
		logger:  logger,
	}
}

// GetClient 获取Redis客户端
func (rm *RedisManager) GetClient(poolType string) (*redis.Client, error) {
	rm.mu.RLock()
	client, exists := rm.clients[poolType]
	rm.mu.RUnlock()

	if exists {
		return client, nil
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 双重检查
	if client, exists := rm.clients[poolType]; exists {
		return client, nil
	}

	// 创建内存Redis客户端
	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	rm.clients[poolType] = client
	rm.logger.Info(context.Background(), "创建Redis客户端", logx.KV("pool_type", poolType))

	return client, nil
}

// Close 关闭所有Redis连接
func (rm *RedisManager) Close() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var lastErr error
	for poolType, client := range rm.clients {
		if err := client.Close(); err != nil {
			rm.logger.Error(context.Background(), "关闭Redis客户端失败", logx.KV("pool_type", poolType), logx.KV("error", err))
			lastErr = err
		}
	}

	rm.clients = make(map[string]*redis.Client)
	return lastErr
}

// Ping 测试连接
func (rm *RedisManager) Ping(ctx context.Context, poolType string) error {
	client, err := rm.GetClient(poolType)
	if err != nil {
		return err
	}

	return client.Ping(ctx).Err()
}

// Stats 获取统计信息
func (rm *RedisManager) Stats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := map[string]interface{}{
		"active_clients": len(rm.clients),
		"client_types":   make([]string, 0, len(rm.clients)),
	}

	for poolType := range rm.clients {
		stats["client_types"] = append(stats["client_types"].([]string), poolType)
	}

	return stats
}
