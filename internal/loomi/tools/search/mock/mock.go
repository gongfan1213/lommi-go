package mock

import (
	"context"
)

type Mock struct{}

func New() *Mock { return &Mock{} }

func (m *Mock) SearchWeb(ctx context.Context, query string, count int) (map[string]any, error) {
	// 返回一个与 Agent 预期兼容的结构
	results := []map[string]any{
		{"title": "示例新闻A", "content": "内容A...", "publish_date": "2025-01-01", "link": "https://a"},
		{"title": "示例新闻B", "content": "内容B...", "publish_date": "2025-01-02", "link": "https://b"},
	}
	return map[string]any{
		"success": true,
		"data": results,
	}, nil
}
