package tokens

import (
	"context"
	"fmt"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/pool"
	"github.com/redis/go-redis/v9"
)

// RedisAccumulator stores token usage keyed by user:session per minute
type RedisAccumulator struct {
	r pool.Manager
}

func NewRedis(r pool.Manager) Accumulator { return &RedisAccumulator{r: r} }

func (a *RedisAccumulator) Init(userID, sessionID string) error {
	// ensure key exists; no-op
	return nil
}

func (a *RedisAccumulator) Initialize(userID, sessionID string) (string, error) {
	return fmt.Sprintf("tokens:%s:%s", userID, sessionID), nil
}

func (a *RedisAccumulator) Summary(userID, sessionID string) (any, error) {
	// Aggregate per-minute counters for this user/session
	ctx := context.Background()
	var total int64 = 0
	if a.r == nil {
		return map[string]any{"total_tokens": 0, "cost": 0.0}, nil
	}
	cli, err := a.r.GetClient("high_priority")
	if err != nil {
		return map[string]any{"total_tokens": 0, "cost": 0.0}, nil
	}
	c, ok := cli.(*redis.Client)
	if !ok {
		return map[string]any{"total_tokens": 0, "cost": 0.0}, nil
	}
	// keys pattern: loomi:tokens:{user_id}:{session_id}:YYYYMMDDHHMM
	pattern := fmt.Sprintf("loomi:tokens:%s:%s:*", userID, sessionID)
	var cursor uint64
	for {
		keys, cur, scanErr := c.Scan(ctx, cursor, pattern, 1000).Result()
		if scanErr != nil {
			break
		}
		cursor = cur
		if len(keys) > 0 {
			vals, mErr := c.MGet(ctx, keys...).Result()
			if mErr == nil {
				for _, v := range vals {
					if v == nil {
						continue
					}
					switch t := v.(type) {
					case string:
						if n, pErr := parseInt64(t); pErr == nil {
							total += n
						}
					case int64:
						total += t
					case int:
						total += int64(t)
					}
				}
			}
		}
		if cursor == 0 {
			break
		}
	}
	cost := float64(total) * 0.000002
	return map[string]any{"total_tokens": total, "cost": cost}, nil
}

func (a *RedisAccumulator) addTokens(ctx context.Context, userID, sessionID string, n int) {
	if a.r == nil {
		return
	}
	cli, err := a.r.GetClient("high_priority")
	if err != nil {
		return
	}
	c, ok := cli.(*redis.Client)
	if !ok {
		return
	}
	key := fmt.Sprintf("loomi:tokens:%s:%s:%s", userID, sessionID, time.Now().Format("200601021504"))
	_ = c.IncrBy(ctx, key, int64(n)).Err()
	_ = c.Expire(ctx, key, 24*time.Hour).Err()
}

// Add increments token counter for user/session at minute granularity
func (a *RedisAccumulator) Add(userID, sessionID string, n int) {
	if n <= 0 {
		return
	}
	a.addTokens(context.Background(), userID, sessionID, n)
}

// Cleanup removes all counters for a specific session
func (a *RedisAccumulator) Cleanup(userID, sessionID string) error {
	if a.r == nil {
		return nil
	}
	cli, err := a.r.GetClient("high_priority")
	if err != nil {
		return nil
	}
	c, ok := cli.(*redis.Client)
	if !ok {
		return nil
	}
	ctx := context.Background()
	pattern := fmt.Sprintf("loomi:tokens:%s:%s:*", userID, sessionID)
	var cursor uint64
	for {
		keys, cur, scanErr := c.Scan(ctx, cursor, pattern, 1000).Result()
		if scanErr != nil {
			break
		}
		cursor = cur
		if len(keys) > 0 {
			_ = c.Del(ctx, keys...).Err()
		}
		if cursor == 0 {
			break
		}
	}
	return nil
}

// parseInt64 tries to parse string to int64 without panicking
func parseInt64(s string) (int64, error) {
	var x int64
	var neg bool
	for i := 0; i < len(s); i++ {
		if i == 0 && s[i] == '-' {
			neg = true
			continue
		}
		d := s[i] - '0'
		if d > 9 {
			return 0, fmt.Errorf("invalid")
		}
		x = x*10 + int64(d)
	}
	if neg {
		x = -x
	}
	return x, nil
}
