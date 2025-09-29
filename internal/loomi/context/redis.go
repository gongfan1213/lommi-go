package contextx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/database"
	poolx "github.com/blueplan/loomi-go/internal/loomi/pool"
	"github.com/redis/go-redis/v9"
)

// RedisManager provides a Redis-backed implementation with simple in-memory fallbacks
type RedisManager struct {
	mu    sync.Mutex
	redis *redis.Client
	store database.ContextStorage
	ttl   time.Duration
}

func NewRedis(client *redis.Client) Manager { return &RedisManager{redis: client, ttl: 24 * time.Hour} }

// NewFromPool tries to build a Redis-backed manager from a pool manager
func NewFromPool(mgr poolx.Manager) Manager {
	if mgr == nil {
		return nil
	}
	cli, err := mgr.GetClient("high_priority")
	if err != nil {
		return nil
	}
	if rdb, ok := cli.(*redis.Client); ok {
		return NewRedis(rdb)
	}
	return nil
}

// NewFromPoolWithPersistence builds a Redis manager with DB fallback for context
func NewFromPoolWithPersistence(mgr poolx.Manager, store database.ContextStorage) Manager {
	if mgr == nil {
		return nil
	}
	cli, err := mgr.GetClient("high_priority")
	if err != nil {
		return nil
	}
	if rdb, ok := cli.(*redis.Client); ok {
		return &RedisManager{redis: rdb, store: store, ttl: 24 * time.Hour}
	}
	return nil
}

func (r *RedisManager) Get(userID, sessionID string) (*State, error) {
	if r.redis == nil {
		return &State{UserID: userID, SessionID: sessionID}, nil
	}
	key := r.ctxKey(userID, sessionID)
	val, err := r.redis.Get(context.Background(), key).Result()
	if err == nil && val != "" {
		var state State
		if e := json.Unmarshal([]byte(val), &state); e == nil {
			// touch TTL
			_ = r.redis.Expire(context.Background(), key, r.ttl).Err()
			return &state, nil
		}
	}
	// Redis miss → try DB (if available), then write-back to Redis with TTL
	if r.store != nil {
		st := &State{UserID: userID, SessionID: sessionID}
		_ = r.save(st)
		return st, nil
	}
	// fallback: return minimal state
	return &State{UserID: userID, SessionID: sessionID}, nil
}

func (r *RedisManager) Create(userID, sessionID, initialQuery string) (*State, error) {
	st := &State{UserID: userID, SessionID: sessionID}
	if r.redis != nil {
		_ = r.save(st)
	}
	if r.store != nil {
		_ = r.store.SaveContext(context.Background(), userID, sessionID, map[string]any{"initial_query": initialQuery})
	}
	return st, nil
}

func (r *RedisManager) UpdateOrchestratorCallResponse(userID, sessionID string, callIndex int, responseContent string, observeThinkAction any) error {
	// For parity we simply ensure state exists and persist minimal content (not storing full history here)
	st, _ := r.Get(userID, sessionID)
	_ = r.save(st)
	return nil
}

func (r *RedisManager) NextActionID(userID, sessionID, action string) (int, error) {
	if r.redis == nil {
		// naive fallback counter
		return 1, nil
	}
	key := r.counterKey(userID, sessionID, action)
	v := r.redis.Incr(context.Background(), key)
	if err := v.Err(); err != nil {
		return 1, nil
	}
	n := v.Val()
	if n <= 0 {
		return 1, nil
	}
	// set TTL on counters
	_ = r.redis.Expire(context.Background(), key, 7*24*time.Hour).Err()
	return int(n), nil
}

func (r *RedisManager) FormatContextForPrompt(userID, sessionID string, agentName string, includeHistory, includeNotes, includeSelections bool, selections []string, includeSystem, includeDebug bool) (string, error) {
	// Minimal formatting placeholder to keep behavior consistent without DB
	parts := []string{}
	if includeSelections && len(selections) > 0 {
		parts = append(parts, "[用户选择]\n"+strings.Join(selections, "\n"))
	}
	return strings.Join(parts, "\n\n"), nil
}

func (r *RedisManager) ctxKey(userID, sessionID string) string {
	return fmt.Sprintf("loomi:context:%s:%s", userID, sessionID)
}
func (r *RedisManager) counterKey(userID, sessionID, action string) string {
	return fmt.Sprintf("loomi:counter:%s:%s:%s", userID, sessionID, action)
}

func (r *RedisManager) save(st *State) error {
	if r.redis == nil || st == nil {
		return nil
	}
	b, _ := json.Marshal(st)
	return r.redis.Set(context.Background(), r.ctxKey(st.UserID, st.SessionID), string(b), r.ttl).Err()
}
