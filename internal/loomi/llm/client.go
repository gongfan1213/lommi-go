package llm

import (
	"context"
	"fmt"
)

type Message struct {
	Role    string
	Content string
}

type StreamChunkHandler func(ctx context.Context, chunk string) error

type Client interface {
	SafeStreamCall(ctx context.Context, userID, sessionID string, messages []Message, onChunk StreamChunkHandler) error
}

// UsageProvider 可选能力：提供最近一次调用的 token 用量（prompt/completion）
type UsageProvider interface {
	LastUsage(userID, sessionID string) (prompt int, completion int, ok bool)
}

// NewClient creates a new LLM client
func NewClient(provider string, config interface{}) (Client, error) {
	// TODO: Implement actual client creation based on provider
	return &mockClient{}, nil
}

// mockClient is a mock implementation for testing
type mockClient struct{}

func (m *mockClient) SafeStreamCall(ctx context.Context, userID, sessionID string, messages []Message, onChunk StreamChunkHandler) error {
	// Mock implementation
	fmt.Println("Mock LLM call for user:", userID, "session:", sessionID)
	return nil
}

// Ensure mockClient also implements UsageProvider with zero usage
func (m *mockClient) LastUsage(userID, sessionID string) (int, int, bool) { return 0, 0, true }
