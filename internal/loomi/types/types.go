package types

import (
	"context"

	"github.com/blueplan/loomi-go/internal/loomi/events"
)

// AgentRequest represents a request to an agent
type AgentRequest struct {
	Instruction string
	UserID      string
	SessionID   string
	UseFiles    bool
	FileIDs     []string
	AutoMode    bool
	Selections  []string
	// Extended fields to mirror Python request_data structure
	Background         map[string]any
	InteractionType    string
	InteractionContext map[string]any
	References         []string
	Nova3Selections    map[string]any
}

// Agent defines the interface for all agents
type Agent interface {
	ProcessRequest(ctx context.Context, req AgentRequest, emit func(ev events.StreamEvent) error) error
}
