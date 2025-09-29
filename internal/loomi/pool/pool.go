package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/redis/go-redis/v9"
)

// Manager 连接池管理器接口
type Manager interface {
	// 获取Redis客户端
	GetRedisClient(ctx context.Context, poolType string) (*redis.Client, error)

	// 获取连接池统计信息
	GetPoolStats(ctx context.Context) (map[string]interface{}, error)

	// 健康检查
	HealthCheck(ctx context.Context) (map[string]interface{}, error)

	// 预预热连接池
	PrewarmPools(ctx context.Context, poolTypes []string) error

	// 关闭连接池
	Close() error
}

// PoolManager 连接池管理器实现
type PoolManager struct {
	redisPools map[string]*redis.Client
	config     *config.MemoryConfig
	logger     *logx.Logger
	mu         sync.RWMutex
	stats      PoolStats
}

// PoolStats 连接池统计信息
type PoolStats struct {
	RedisRequests   int64     `json:"redis_requests"`
	RedisFailures   int64     `json:"redis_failures"`
	PoolExhaustions int64     `json:"pool_exhaustions"`
	LastReset       time.Time `json:"last_reset"`
}

// NewPoolManager 创建新的连接池管理器
func NewPoolManager(cfg *config.MemoryConfig, logger *logx.Logger) *PoolManager {
	pm := &PoolManager{
		redisPools: make(map[string]*redis.Client),
		config:     cfg,
		logger:     logger,
		stats: PoolStats{
			LastReset: time.Now(),
		},
	}

	pm.logger.Info(context.Background(), "连接池管理器初始化完成")
	return pm
}

// GetRedisClient 获取Redis客户端
func (pm *PoolManager) GetRedisClient(ctx context.Context, poolType string) (*redis.Client, error) {
	pm.mu.RLock()
	client, exists := pm.redisPools[poolType]
	pm.mu.RUnlock()

	if exists {
		pm.stats.RedisRequests++
		return client, nil
	}

	// 创建新的连接池
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 双重检查
	if client, exists := pm.redisPools[poolType]; exists {
		pm.stats.RedisRequests++
		return client, nil
	}

	client, err := pm.createRedisPool(poolType)
	if err != nil {
		pm.stats.RedisFailures++
		return nil, fmt.Errorf("创建Redis连接池失败 (pool_type=%s): %w", poolType, err)
	}

	pm.redisPools[poolType] = client
	pm.stats.RedisRequests++

	pm.logger.Info(ctx, "创建新的Redis连接池", logx.KV("pool_type", poolType))
	return client, nil
}

// createRedisPool 创建Redis连接池
func (pm *PoolManager) createRedisPool(poolType string) (*redis.Client, error) {
	// 根据池类型设置不同的连接数
	maxConnections := pm.getMaxConnectionsForPoolType(poolType)

	// 构建Redis URL
	var redisURL string
	if pm.config.RedisPassword != "" {
		redisURL = fmt.Sprintf("redis://:%s@%s:%d/%d",
			pm.config.RedisPassword,
			pm.config.RedisHost,
			pm.config.RedisPort,
			pm.config.RedisDB)
	} else {
		redisURL = fmt.Sprintf("redis://%s:%d/%d",
			pm.config.RedisHost,
			pm.config.RedisPort,
			pm.config.RedisDB)
	}

	// 创建Redis客户端
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("解析Redis URL失败: %w", err)
	}

	opt.PoolSize = maxConnections
	opt.MinIdleConns = 1
	opt.MaxRetries = 3
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	opt.PoolTimeout = 4 * time.Second

	client := redis.NewClient(opt)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("Redis连接测试失败: %w", err)
	}

	return client, nil
}

// getMaxConnectionsForPoolType 根据池类型获取最大连接数
func (pm *PoolManager) getMaxConnectionsForPoolType(poolType string) int {
	switch poolType {
	case "high_priority":
		return 200
	case "normal":
		return 100
	case "background":
		return 50
	default:
		return 100
	}
}

// GetPoolStats 获取连接池统计信息
func (pm *PoolManager) GetPoolStats(ctx context.Context) (map[string]interface{}, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := map[string]interface{}{
		"redis_requests":   pm.stats.RedisRequests,
		"redis_failures":   pm.stats.RedisFailures,
		"pool_exhaustions": pm.stats.PoolExhaustions,
		"last_reset":       pm.stats.LastReset,
		"active_pools":     len(pm.redisPools),
		"pool_types":       make([]string, 0, len(pm.redisPools)),
	}

	// 添加池类型信息
	for poolType := range pm.redisPools {
		stats["pool_types"] = append(stats["pool_types"].([]string), poolType)
	}

	return stats, nil
}

// HealthCheck 健康检查
func (pm *PoolManager) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	health := map[string]interface{}{
		"overall_status": "healthy",
		"pools":          make(map[string]interface{}),
	}

	allHealthy := true

	for poolType, client := range pm.redisPools {
		poolHealth := map[string]interface{}{
			"status": "unknown",
		}

		// 测试连接
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := client.Ping(pingCtx).Err(); err != nil {
			poolHealth["status"] = "unhealthy"
			poolHealth["error"] = err.Error()
			allHealthy = false
		} else {
			poolHealth["status"] = "healthy"
			// 获取连接池统计
			poolStats := client.PoolStats()
			poolHealth["stats"] = map[string]interface{}{
				"total_conns": poolStats.TotalConns,
				"idle_conns":  poolStats.IdleConns,
				"stale_conns": poolStats.StaleConns,
				"hits":        poolStats.Hits,
				"misses":      poolStats.Misses,
			}
		}
		cancel()

		health["pools"].(map[string]interface{})[poolType] = poolHealth
	}

	if !allHealthy {
		health["overall_status"] = "degraded"
	}

	return health, nil
}

// PrewarmPools 预预热连接池
func (pm *PoolManager) PrewarmPools(ctx context.Context, poolTypes []string) error {
	if len(poolTypes) == 0 {
		poolTypes = []string{"high_priority", "normal", "background"}
	}

	pm.logger.Info(ctx, "开始预热连接池", logx.KV("pool_types", poolTypes))

	for _, poolType := range poolTypes {
		client, err := pm.GetRedisClient(ctx, poolType)
		if err != nil {
			pm.logger.Error(ctx, "预热连接池失败", logx.KV("pool_type", poolType), logx.KV("error", err))
			continue
		}

		// 执行一些预热操作
		for i := 0; i < 3; i++ {
			if err := client.Ping(ctx).Err(); err != nil {
				pm.logger.Warn(ctx, "预热操作失败", logx.KV("pool_type", poolType), logx.KV("attempt", i+1), logx.KV("error", err))
			}
		}

		pm.logger.Info(ctx, "连接池预热完成", logx.KV("pool_type", poolType))
	}

	return nil
}

// Close 关闭连接池
func (pm *PoolManager) Close() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var lastErr error
	for poolType, client := range pm.redisPools {
		if err := client.Close(); err != nil {
			pm.logger.Error(context.Background(), "关闭连接池失败", logx.KV("pool_type", poolType), logx.KV("error", err))
			lastErr = err
		}
	}

	pm.redisPools = make(map[string]*redis.Client)
	pm.logger.Info(context.Background(), "所有连接池已关闭")

	return lastErr
}

// ResetStats 重置统计信息
func (pm *PoolManager) ResetStats() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.stats = PoolStats{
		LastReset: time.Now(),
	}
}

// GetConnectionStats 获取连接统计信息
func (pm *PoolManager) GetConnectionStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := map[string]interface{}{
		"redis_requests":   pm.stats.RedisRequests,
		"redis_failures":   pm.stats.RedisFailures,
		"pool_exhaustions": pm.stats.PoolExhaustions,
		"active_pools":     len(pm.redisPools),
	}

	return stats
}

// 全局连接池管理器实例
var (
	globalPoolManager Manager
	globalPoolOnce    sync.Once
)

// GetPoolManager 获取全局连接池管理器
func GetPoolManager() Manager {
	return globalPoolManager
}

// InitializePoolManager 初始化全局连接池管理器
func InitializePoolManager(cfg *config.MemoryConfig, logger *logx.Logger) (Manager, error) {
	var err error
	globalPoolOnce.Do(func() {
		globalPoolManager = NewPoolManager(cfg, logger)

		// 预热连接池
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err = globalPoolManager.PrewarmPools(ctx, []string{"high_priority"}); err != nil {
			logger.Error(ctx, "预热连接池失败", logx.KV("error", err))
		}
	})

	return globalPoolManager, err
}
