package stopx

import (
	"context"
	"errors"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/pool"
	"github.com/redis/go-redis/v9"
)

type StoppedError struct{ Reason string }

func (e *StoppedError) Error() string { return e.Reason }

var ErrStopped = errors.New("stopped")

type Manager interface {
	Clear(userID, sessionID string) error
	Check(userID, sessionID string) error
	RequestStop(userID, sessionID string) error
	IsStopped(userID, sessionID string) (bool, error)
	ClearStopState(userID, sessionID string) error
}

// Inmem + Redis 结合的停止管理器，TTL=30s，与 Python 行为一致
type redisMgr struct{ r pool.Manager }

// NewRedis 返回带 Redis 能力的 Manager；保持原 inmem 独立
func NewRedis(r pool.Manager) Manager { return &redisMgr{r: r} }

func (m *redisMgr) Clear(userID, sessionID string) error {
	if m.r == nil {
		return nil
	}
	client, err := m.r.GetClient("high_priority")
	if err != nil {
		return nil
	}
	c, ok := client.(*redis.Client)
	if !ok {
		return nil
	}
	key := m.stopKey(userID, sessionID)
	_ = c.Del(context.Background(), key).Err()
	return nil
}

func (m *redisMgr) Check(userID, sessionID string) error {
	if m.r == nil {
		return nil
	}
	client, err := m.r.GetClient("high_priority")
	if err != nil {
		return nil
	}
	c, ok := client.(*redis.Client)
	if !ok {
		return nil
	}
	key := m.stopKey(userID, sessionID)
	v, err := c.Get(context.Background(), key).Result()
	if err == nil && v == "1" {
		return &StoppedError{Reason: "user requested stop"}
	}
	return nil
}

// RequestStop 供外部调用：设置 30s TTL 的停止标记
func (m *redisMgr) RequestStop(userID, sessionID string) error {
	if m.r == nil {
		return nil
	}
	client, err := m.r.GetClient("high_priority")
	if err != nil {
		return nil
	}
	c, ok := client.(*redis.Client)
	if !ok {
		return nil
	}
	key := m.stopKey(userID, sessionID)
	_ = c.Set(context.Background(), key, "1", 30*time.Second).Err()
	return nil
}

func (m *redisMgr) stopKey(userID, sessionID string) string {
	return "loomi:stop:" + userID + ":" + sessionID
}

// IsStopped checks if a session is stopped
func (m *redisMgr) IsStopped(userID, sessionID string) (bool, error) {
	if m.r == nil {
		return false, nil
	}
	client, err := m.r.GetClient("high_priority")
	if err != nil {
		return false, nil
	}
	c, ok := client.(*redis.Client)
	if !ok {
		return false, nil
	}
	key := m.stopKey(userID, sessionID)
	v, err := c.Get(context.Background(), key).Result()
	if err == nil && v == "1" {
		return true, nil
	}
	return false, nil
}

// ClearStopState clears the stop state for a session
func (m *redisMgr) ClearStopState(userID, sessionID string) error {
	return m.Clear(userID, sessionID)
}
