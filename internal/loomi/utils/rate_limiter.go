package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/redis/go-redis/v9"
)

// RateLimiter 速率限制器接口
type RateLimiter interface {
	// 检查是否允许请求
	Allow(ctx context.Context, key string) (bool, error)

	// 获取剩余请求数
	Remaining(ctx context.Context, key string) (int, error)

	// 重置限制
	Reset(ctx context.Context, key string) error

	// 获取限制信息
	GetLimitInfo(ctx context.Context, key string) (*LimitInfo, error)
}

// LimitInfo 限制信息
type LimitInfo struct {
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	ResetTime time.Time     `json:"reset_time"`
	Window    time.Duration `json:"window"`
}

// RedisRateLimiter Redis速率限制器
type RedisRateLimiter struct {
	client redis.UniversalClient
	logger *logx.Logger
	window time.Duration
	limit  int
	prefix string
}

// NewRedisRateLimiter 创建Redis速率限制器
func NewRedisRateLimiter(client redis.UniversalClient, logger *logx.Logger, window time.Duration, limit int) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		logger: logger,
		window: window,
		limit:  limit,
		prefix: "rate_limit:",
	}
}

// Allow 检查是否允许请求（滑动窗口算法）
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := rl.prefix + key
	now := time.Now()
	windowStart := now.Add(-rl.window)

	// 使用Lua脚本实现原子操作
	script := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local window_end = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		-- 删除过期的请求记录
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)
		
		-- 获取当前窗口内的请求数
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- 添加当前请求
			redis.call('ZADD', key, now, now)
			-- 设置过期时间
			redis.call('EXPIRE', key, window_end - window_start)
			return {1, limit - current - 1, window_end}
		else
			return {0, 0, window_end}
		end
	`

	result, err := rl.client.Eval(ctx, script, []string{redisKey},
		windowStart.Unix(), now.Unix(), rl.limit, now.Unix()).Result()
	if err != nil {
		rl.logger.Error(ctx, "速率限制检查失败",
			logx.KV("key", key),
			logx.KV("error", err))
		return false, err
	}

	// 解析结果
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 3 {
		return false, fmt.Errorf("无效的Lua脚本返回结果")
	}

	allowed := resultSlice[0].(int64) == 1
	remaining := resultSlice[1].(int64)
	resetTime := time.Unix(resultSlice[2].(int64), 0)

	if !allowed {
		rl.logger.Warn(ctx, "请求被速率限制",
			logx.KV("key", key),
			logx.KV("remaining", remaining),
			logx.KV("reset_time", resetTime))
	}

	return allowed, nil
}

// Remaining 获取剩余请求数
func (rl *RedisRateLimiter) Remaining(ctx context.Context, key string) (int, error) {
	redisKey := rl.prefix + key
	now := time.Now()
	windowStart := now.Add(-rl.window)

	// 删除过期的请求记录
	rl.client.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.Unix()))

	// 获取当前窗口内的请求数
	current, err := rl.client.ZCard(ctx, redisKey).Result()
	if err != nil {
		return 0, err
	}

	remaining := rl.limit - int(current)
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset 重置限制
func (rl *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := rl.prefix + key
	return rl.client.Del(ctx, redisKey).Err()
}

// GetLimitInfo 获取限制信息
func (rl *RedisRateLimiter) GetLimitInfo(ctx context.Context, key string) (*LimitInfo, error) {
	remaining, err := rl.Remaining(ctx, key)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	resetTime := now.Add(rl.window)

	return &LimitInfo{
		Limit:     rl.limit,
		Remaining: remaining,
		ResetTime: resetTime,
		Window:    rl.window,
	}, nil
}

// MemoryRateLimiter 内存速率限制器
type MemoryRateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	logger   *logx.Logger
	window   time.Duration
	limit    int
}

// NewMemoryRateLimiter 创建内存速率限制器
func NewMemoryRateLimiter(logger *logx.Logger, window time.Duration, limit int) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		requests: make(map[string][]time.Time),
		logger:   logger,
		window:   window,
		limit:    limit,
	}
}

// Allow 检查是否允许请求
func (ml *MemoryRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-ml.window)

	// 获取或创建请求记录
	requests, exists := ml.requests[key]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// 删除过期的请求记录
	validRequests := make([]time.Time, 0)
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// 检查是否超过限制
	if len(validRequests) >= ml.limit {
		ml.logger.Warn(ctx, "请求被速率限制",
			logx.KV("key", key),
			logx.KV("current", len(validRequests)),
			logx.KV("limit", ml.limit))
		return false, nil
	}

	// 添加当前请求
	validRequests = append(validRequests, now)
	ml.requests[key] = validRequests

	return true, nil
}

// Remaining 获取剩余请求数
func (ml *MemoryRateLimiter) Remaining(ctx context.Context, key string) (int, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-ml.window)

	requests, exists := ml.requests[key]
	if !exists {
		return ml.limit, nil
	}

	// 计算有效请求数
	validCount := 0
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validCount++
		}
	}

	remaining := ml.limit - validCount
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset 重置限制
func (ml *MemoryRateLimiter) Reset(ctx context.Context, key string) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	delete(ml.requests, key)
	return nil
}

// GetLimitInfo 获取限制信息
func (ml *MemoryRateLimiter) GetLimitInfo(ctx context.Context, key string) (*LimitInfo, error) {
	remaining, err := ml.Remaining(ctx, key)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	resetTime := now.Add(ml.window)

	return &LimitInfo{
		Limit:     ml.limit,
		Remaining: remaining,
		ResetTime: resetTime,
		Window:    ml.window,
	}, nil
}

// RateLimiterConfig 速率限制器配置
type RateLimiterConfig struct {
	Window          time.Duration `json:"window"`
	Limit           int           `json:"limit"`
	Type            string        `json:"type"` // "redis" or "memory"
	Prefix          string        `json:"prefix"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// RateLimiterManager 速率限制器管理器
type RateLimiterManager struct {
	limiters map[string]RateLimiter
	configs  map[string]RateLimiterConfig
	logger   *logx.Logger
	mu       sync.RWMutex
}

// NewRateLimiterManager 创建速率限制器管理器
func NewRateLimiterManager(logger *logx.Logger) *RateLimiterManager {
	return &RateLimiterManager{
		limiters: make(map[string]RateLimiter),
		configs:  make(map[string]RateLimiterConfig),
		logger:   logger,
	}
}

// AddRateLimiter 添加速率限制器
func (rlm *RateLimiterManager) AddRateLimiter(name string, config RateLimiterConfig, client redis.UniversalClient) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	var limiter RateLimiter

	switch config.Type {
	case "redis":
		if client == nil {
			rlm.logger.Error(context.Background(), "Redis客户端未提供", logx.KV("limiter_name", name))
			return
		}
		limiter = NewRedisRateLimiter(client, rlm.logger, config.Window, config.Limit)
		if config.Prefix != "" {
			limiter.(*RedisRateLimiter).prefix = config.Prefix
		}
	case "memory":
		limiter = NewMemoryRateLimiter(rlm.logger, config.Window, config.Limit)
	default:
		rlm.logger.Error(context.Background(), "不支持的速率限制器类型",
			logx.KV("limiter_name", name),
			logx.KV("type", config.Type))
		return
	}

	rlm.limiters[name] = limiter
	rlm.configs[name] = config

	rlm.logger.Info(context.Background(), "添加速率限制器",
		logx.KV("name", name),
		logx.KV("type", config.Type),
		logx.KV("window", config.Window),
		logx.KV("limit", config.Limit))
}

// GetRateLimiter 获取速率限制器
func (rlm *RateLimiterManager) GetRateLimiter(name string) (RateLimiter, bool) {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()

	limiter, exists := rlm.limiters[name]
	return limiter, exists
}

// Allow 检查是否允许请求
func (rlm *RateLimiterManager) Allow(ctx context.Context, limiterName, key string) (bool, error) {
	limiter, exists := rlm.GetRateLimiter(limiterName)
	if !exists {
		return false, fmt.Errorf("速率限制器不存在: %s", limiterName)
	}

	return limiter.Allow(ctx, key)
}

// GetLimitInfo 获取限制信息
func (rlm *RateLimiterManager) GetLimitInfo(ctx context.Context, limiterName, key string) (*LimitInfo, error) {
	limiter, exists := rlm.GetRateLimiter(limiterName)
	if !exists {
		return nil, fmt.Errorf("速率限制器不存在: %s", limiterName)
	}

	return limiter.GetLimitInfo(ctx, key)
}

// ListLimiters 列出所有速率限制器
func (rlm *RateLimiterManager) ListLimiters() map[string]RateLimiterConfig {
	rlm.mu.RLock()
	defer rlm.mu.RUnlock()

	result := make(map[string]RateLimiterConfig)
	for name, config := range rlm.configs {
		result[name] = config
	}

	return result
}

// RemoveRateLimiter 移除速率限制器
func (rlm *RateLimiterManager) RemoveRateLimiter(name string) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()

	delete(rlm.limiters, name)
	delete(rlm.configs, name)

	rlm.logger.Info(context.Background(), "移除速率限制器", logx.KV("name", name))
}

// 全局速率限制器管理器实例
var (
	globalRateLimiterManager *RateLimiterManager
	globalRateLimiterOnce    sync.Once
)

// GetRateLimiterManager 获取全局速率限制器管理器
func GetRateLimiterManager() *RateLimiterManager {
	return globalRateLimiterManager
}

// InitializeRateLimiterManager 初始化全局速率限制器管理器
func InitializeRateLimiterManager(logger *logx.Logger) *RateLimiterManager {
	globalRateLimiterOnce.Do(func() {
		globalRateLimiterManager = NewRateLimiterManager(logger)
	})

	return globalRateLimiterManager
}
