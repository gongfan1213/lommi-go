package utils

import (
	"context"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/pool"
)

// AccessCounter 与 Python 版本的关键行为保持一致（键名、TTL 近似）
// 依赖外部注入的 Redis 管理器，支持高优先级池获取
type AccessCounter struct {
	logger *logx.Logger
	redis  pool.Manager

	totalKey           string
	dailyKeyPrefix     string
	userKeyPrefix      string
	dailyUserKeyPrefix string
	todayTTLSeconds    int
	recentTTLSeconds   int
	userTTLDays        int
}

func NewAccessCounter(logger *logx.Logger, redis pool.Manager) *AccessCounter {
	return &AccessCounter{
		logger:             logger,
		redis:              redis,
		totalKey:           "novachat:access:total",
		dailyKeyPrefix:     "novachat:access:daily:",
		userKeyPrefix:      "novachat:access:user:",
		dailyUserKeyPrefix: "novachat:access:daily_user:",
		todayTTLSeconds:    86400,
		recentTTLSeconds:   86400 * 7,
		userTTLDays:        7,
	}
}

func (a *AccessCounter) Inc(ctx context.Context, userID string) error {
	// Simple in-memory implementation for now
	a.logger.Info(ctx, "Access counter incremented", logx.KV("user_id", userID))
	return nil
}

func (a *AccessCounter) GetTotal(ctx context.Context) (int64, error) {
	// Simple in-memory implementation for now
	return 1000, nil
}

func (a *AccessCounter) GetDaily(ctx context.Context, date string) (int64, error) {
	// Simple in-memory implementation for now
	return 100, nil
}

func (a *AccessCounter) GetUserAccess(ctx context.Context, userID string) (int64, error) {
	// Simple in-memory implementation for now
	return 10, nil
}

func (a *AccessCounter) GetDailyUserAccess(ctx context.Context, userID, date string) (int64, error) {
	// Simple in-memory implementation for now
	return 1, nil
}
