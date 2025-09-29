package pool

import "context"

// InmemManager provides an in-memory implementation of the pool manager
type InmemManager struct {
	// Simple in-memory implementation
}

// NewInmem creates a new in-memory pool manager
func NewInmem() Manager {
	return &InmemManager{}
}

func (m *InmemManager) Prewarm(ctx context.Context, pools []string) (map[string]any, error) {
	result := make(map[string]any)
	for _, pool := range pools {
		result[pool] = map[string]any{
			"success": true,
			"message": "Pool " + pool + " prewarmed successfully",
		}
	}
	return result, nil
}

func (m *InmemManager) Stats() (map[string]int, error) {
	return map[string]int{
		"redis_requests":   0,
		"redis_failures":   0,
		"pool_exhaustions": 0,
	}, nil
}

func (m *InmemManager) GetClient(poolType string) (interface{}, error) {
	// Return a mock client
	return "mock_client", nil
}
