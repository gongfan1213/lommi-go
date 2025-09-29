package contextx

import "context"

type State struct {
	UserID       string
	SessionID    string
	CreatedNotes []string
}

type Manager interface {
	Get(userID, sessionID string) (*State, error)
	Create(userID, sessionID, initialQuery string) (*State, error)
	UpdateOrchestratorCallResponse(userID, sessionID string, callIndex int, responseContent string, observeThinkAction any) error
	NextActionID(userID, sessionID, action string) (int, error)
	// Build formatted context for prompts (history + notes + selections)
	FormatContextForPrompt(userID, sessionID, agentName string, includeHistory, includeNotes, includeSelections bool, selections []string, includeSystem, includeDebug bool) (string, error)
}

// 与 Python core/context.py 对齐：require_id 上下文键与便捷方法
// 便于日志模块注入并读取
type RequireIDKey struct{}

func WithRequireID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequireIDKey{}, id)
}

func GetRequireID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(RequireIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

// UserIDKey 与 Python 中 user_id 上下文语义对齐
type UserIDKey struct{}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey{}, id)
}

func GetUserID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(UserIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}
