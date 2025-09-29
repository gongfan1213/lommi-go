package tokens

import "sync"

// InmemAccumulator provides an in-memory implementation of the token accumulator
type InmemAccumulator struct {
	mu   sync.Mutex
	data map[string]int
}

// NewInmem creates a new in-memory token accumulator
func NewInmem() Accumulator {
	return &InmemAccumulator{data: map[string]int{}}
}

func (a *InmemAccumulator) Init(userID, sessionID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.data[userID+":"+sessionID] = 0
	return nil
}

func (a *InmemAccumulator) Summary(userID, sessionID string) (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	total := a.data[userID+":"+sessionID]
	// 简化成本计算（可按模型单价细化）
	cost := float64(total) * 0.000002
	return map[string]any{"total_tokens": total, "cost": cost}, nil
}

func (a *InmemAccumulator) Initialize(userID, sessionID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := userID + ":" + sessionID
	if _, ok := a.data[key]; !ok {
		a.data[key] = 0
	}
	return key, nil
}

// Add increments token count for a key
func (a *InmemAccumulator) Add(userID, sessionID string, n int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.data[userID+":"+sessionID] += n
}

// Cleanup clears stats for a session
func (a *InmemAccumulator) Cleanup(userID, sessionID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.data, userID+":"+sessionID)
	return nil
}
