package utils

import (
	"context"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/pool"
	"github.com/redis/go-redis/v9"
)

// LayeredQueue 分层队列实现，与 Python 版本保持一致
type LayeredQueue struct {
	logger *logx.Logger
	redis  pool.Manager

	queueKeyPrefix   string
	processingPrefix string
	timeoutSeconds   int
}

func NewLayeredQueue(logger *logx.Logger, redis pool.Manager) *LayeredQueue {
	return &LayeredQueue{
		logger:           logger,
		redis:            redis,
		queueKeyPrefix:   "loomi:queue:",
		processingPrefix: "loomi:processing:",
		timeoutSeconds:   300, // 5 minutes
	}
}

func (q *LayeredQueue) Push(ctx context.Context, queueName string, item string) error {
	if q.redis == nil {
		q.logger.Info(ctx, "Item pushed to queue", logx.KV("queue", queueName), logx.KV("item", item))
		return nil
	}
	cli, err := q.redis.GetClient("high_priority")
	if err != nil {
		return nil
	}
	if c, ok := cli.(*redis.Client); ok {
		key := q.queueKeyPrefix + queueName
		_ = c.LPush(ctx, key, item).Err()
	}
	return nil
}

func (q *LayeredQueue) Pop(ctx context.Context, queueName string) (string, error) {
	if q.redis == nil {
		return "", nil
	}
	cli, err := q.redis.GetClient("high_priority")
	if err != nil {
		return "", nil
	}
	if c, ok := cli.(*redis.Client); ok {
		key := q.queueKeyPrefix + queueName
		v, err := c.BRPop(ctx, time.Duration(q.timeoutSeconds)*time.Second, key).Result()
		if err == nil && len(v) == 2 {
			return v[1], nil
		}
	}
	return "", nil
}

func (q *LayeredQueue) MarkProcessing(ctx context.Context, queueName, item string) error {
	if q.redis == nil {
		q.logger.Info(ctx, "Item marked as processing", logx.KV("queue", queueName), logx.KV("item", item))
		return nil
	}
	cli, err := q.redis.GetClient("high_priority")
	if err != nil {
		return nil
	}
	if c, ok := cli.(*redis.Client); ok {
		key := q.processingPrefix + queueName
		_ = c.Set(ctx, key+":"+item, time.Now().UTC().Format(time.RFC3339), time.Duration(q.timeoutSeconds)*time.Second).Err()
	}
	return nil
}

func (q *LayeredQueue) MarkCompleted(ctx context.Context, queueName, item string) error {
	if q.redis == nil {
		q.logger.Info(ctx, "Item marked as completed", logx.KV("queue", queueName), logx.KV("item", item))
		return nil
	}
	cli, err := q.redis.GetClient("high_priority")
	if err != nil {
		return nil
	}
	if c, ok := cli.(*redis.Client); ok {
		key := q.processingPrefix + queueName
		_ = c.Del(ctx, key+":"+item).Err()
	}
	return nil
}
