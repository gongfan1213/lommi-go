package stopx

// InmemManager provides an in-memory implementation of the stop manager
type InmemManager struct {
	// Simple in-memory implementation
}

// NewInmem creates a new in-memory stop manager
func NewInmem() Manager {
	return &InmemManager{}
}

func (m *InmemManager) Clear(userID, sessionID string) error {
	// No-op for in-memory implementation
	return nil
}

func (m *InmemManager) Check(userID, sessionID string) error {
	// No-op for in-memory implementation
	return nil
}

func (m *InmemManager) RequestStop(userID, sessionID string) error {
	// No-op for in-memory implementation
	return nil
}

func (m *InmemManager) IsStopped(userID, sessionID string) (bool, error) {
	// No-op for in-memory implementation
	return false, nil
}

func (m *InmemManager) ClearStopState(userID, sessionID string) error {
	// No-op for in-memory implementation
	return nil
}
