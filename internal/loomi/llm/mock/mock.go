package mock

import (
	"context"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/llm"
)

type Mock struct{}

func New() *Mock { return &Mock{} }

func (m *Mock) SafeStreamCall(ctx context.Context, userID, sessionID string, messages []llm.Message, onChunk llm.StreamChunkHandler) error {
	// 简单回放：延迟并输出固定片段
	chunks := []string{"<think>思考...</think>", "<Action type=\"knowledge\">请输出3条知识</Action>"}
	for _, c := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_ = onChunk(ctx, c)
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
