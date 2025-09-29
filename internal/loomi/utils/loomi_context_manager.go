package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/pool"
)

// LoomiContextState Loomi上下文状态数据结构
type LoomiContextState struct {
	SessionID           string                   `json:"session_id"`
	UserID              string                   `json:"user_id"`
	ThreadID            string                   `json:"thread_id"`
	UserMessageQueue    []map[string]interface{} `json:"user_message_queue"`
	OrchestratorCalls   []map[string]interface{} `json:"orchestrator_calls"`
	CreatedNotes        []map[string]interface{} `json:"created_notes"`
	GlobalContext       map[string]interface{}   `json:"global_context"`
	AgentContexts       map[string]interface{}   `json:"agent_contexts"`
	SharedMemory        map[string]interface{}   `json:"shared_memory"`
	ConversationHistory []map[string]interface{} `json:"conversation_history"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
}

// LoomiContextManager Loomi多智能体上下文管理器
type LoomiContextManager struct {
	logger       log.Logger
	redisManager pool.Manager
	contexts     map[string]*LoomiContextState
}

// NewLoomiContextManager 创建Loomi上下文管理器
func NewLoomiContextManager(logger log.Logger, redisManager pool.Manager) *LoomiContextManager {
	return &LoomiContextManager{
		logger:       logger,
		redisManager: redisManager,
		contexts:     make(map[string]*LoomiContextState),
	}
}

// CreateContext 创建新的上下文状态
func (lcm *LoomiContextManager) CreateContext(ctx context.Context, userID, sessionID, threadID string) (*LoomiContextState, error) {
	lcm.logger.Info(ctx, "创建新的上下文状态",
		"user_id", userID,
		"session_id", sessionID,
		"thread_id", threadID)

	contextKey := lcm.getContextKey(userID, sessionID)

	// 检查是否已存在
	if existingContext, exists := lcm.contexts[contextKey]; exists {
		lcm.logger.Warn(ctx, "上下文已存在，返回现有上下文", "context_key", contextKey)
		return existingContext, nil
	}

	// 创建新的上下文状态
	contextState := &LoomiContextState{
		SessionID:           sessionID,
		UserID:              userID,
		ThreadID:            threadID,
		UserMessageQueue:    make([]map[string]interface{}, 0),
		OrchestratorCalls:   make([]map[string]interface{}, 0),
		CreatedNotes:        make([]map[string]interface{}, 0),
		GlobalContext:       make(map[string]interface{}),
		AgentContexts:       make(map[string]interface{}),
		SharedMemory:        make(map[string]interface{}),
		ConversationHistory: make([]map[string]interface{}, 0),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// 保存到内存
	lcm.contexts[contextKey] = contextState

	// 保存到Redis
	err := lcm.saveContextToRedis(ctx, contextState)
	if err != nil {
		lcm.logger.Error(ctx, "保存上下文到Redis失败", "error", err)
		// 不返回错误，继续使用内存版本
	}

	lcm.logger.Info(ctx, "上下文状态创建成功", "context_key", contextKey)
	return contextState, nil
}

// GetContext 获取上下文状态
func (lcm *LoomiContextManager) GetContext(ctx context.Context, userID, sessionID string) (*LoomiContextState, error) {
	contextKey := lcm.getContextKey(userID, sessionID)

	// 先从内存获取
	if contextState, exists := lcm.contexts[contextKey]; exists {
		return contextState, nil
	}

	// 从Redis获取
	contextState, err := lcm.loadContextFromRedis(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("从Redis加载上下文失败: %w", err)
	}

	if contextState != nil {
		// 保存到内存
		lcm.contexts[contextKey] = contextState
		return contextState, nil
	}

	return nil, fmt.Errorf("上下文不存在: userID=%s, sessionID=%s", userID, sessionID)
}

// UpdateContext 更新上下文状态
func (lcm *LoomiContextManager) UpdateContext(ctx context.Context, userID, sessionID string, updates map[string]interface{}) error {
	contextState, err := lcm.GetContext(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("获取上下文失败: %w", err)
	}

	// 更新上下文
	contextState.UpdatedAt = time.Now()

	// 应用更新
	for key, value := range updates {
		switch key {
		case "global_context":
			if globalContext, ok := value.(map[string]interface{}); ok {
				contextState.GlobalContext = globalContext
			}
		case "agent_contexts":
			if agentContexts, ok := value.(map[string]interface{}); ok {
				contextState.AgentContexts = agentContexts
			}
		case "shared_memory":
			if sharedMemory, ok := value.(map[string]interface{}); ok {
				contextState.SharedMemory = sharedMemory
			}
		case "conversation_history":
			if conversationHistory, ok := value.([]map[string]interface{}); ok {
				contextState.ConversationHistory = conversationHistory
			}
		}
	}

	// 保存到Redis
	err = lcm.saveContextToRedis(ctx, contextState)
	if err != nil {
		lcm.logger.Error(ctx, "保存更新后的上下文到Redis失败", "error", err)
		// 不返回错误，继续使用内存版本
	}

	lcm.logger.Info(ctx, "上下文状态更新成功",
		"user_id", userID,
		"session_id", sessionID,
		"updates_count", len(updates))

	return nil
}

// AddUserMessage 添加用户消息到队列
func (lcm *LoomiContextManager) AddUserMessage(ctx context.Context, userID, sessionID string, message map[string]interface{}) error {
	contextState, err := lcm.GetContext(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("获取上下文失败: %w", err)
	}

	// 添加消息到队列
	message["timestamp"] = time.Now()
	contextState.UserMessageQueue = append(contextState.UserMessageQueue, message)
	contextState.UpdatedAt = time.Now()

	// 保存到Redis
	err = lcm.saveContextToRedis(ctx, contextState)
	if err != nil {
		lcm.logger.Error(ctx, "保存用户消息到Redis失败", "error", err)
	}

	lcm.logger.Info(ctx, "用户消息添加到队列成功",
		"user_id", userID,
		"session_id", sessionID,
		"queue_length", len(contextState.UserMessageQueue))

	return nil
}

// AddOrchestratorCall 添加orchestrator调用记录
func (lcm *LoomiContextManager) AddOrchestratorCall(ctx context.Context, userID, sessionID string, call map[string]interface{}) error {
	contextState, err := lcm.GetContext(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("获取上下文失败: %w", err)
	}

	// 添加调用记录
	call["timestamp"] = time.Now()
	contextState.OrchestratorCalls = append(contextState.OrchestratorCalls, call)
	contextState.UpdatedAt = time.Now()

	// 保存到Redis
	err = lcm.saveContextToRedis(ctx, contextState)
	if err != nil {
		lcm.logger.Error(ctx, "保存orchestrator调用到Redis失败", "error", err)
	}

	lcm.logger.Info(ctx, "Orchestrator调用记录添加成功",
		"user_id", userID,
		"session_id", sessionID,
		"calls_count", len(contextState.OrchestratorCalls))

	return nil
}

// AddCreatedNote 添加创建的note记录
func (lcm *LoomiContextManager) AddCreatedNote(ctx context.Context, userID, sessionID string, note map[string]interface{}) error {
	contextState, err := lcm.GetContext(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("获取上下文失败: %w", err)
	}

	// 添加note记录
	note["timestamp"] = time.Now()
	contextState.CreatedNotes = append(contextState.CreatedNotes, note)
	contextState.UpdatedAt = time.Now()

	// 保存到Redis
	err = lcm.saveContextToRedis(ctx, contextState)
	if err != nil {
		lcm.logger.Error(ctx, "保存创建的note到Redis失败", "error", err)
	}

	lcm.logger.Info(ctx, "创建的note记录添加成功",
		"user_id", userID,
		"session_id", sessionID,
		"notes_count", len(contextState.CreatedNotes))

	return nil
}

// FormatContextForPrompt 格式化上下文用于提示词
func (lcm *LoomiContextManager) FormatContextForPrompt(ctx context.Context, userID, sessionID, agentName string, includeHistory, includeNotes, includeSelections bool, selections []string, includeSystem, includeDebug bool) (string, error) {
	contextState, err := lcm.GetContext(ctx, userID, sessionID)
	if err != nil {
		return "", fmt.Errorf("获取上下文失败: %w", err)
	}

	var promptParts []string

	// 添加系统信息
	if includeSystem {
		systemInfo := fmt.Sprintf("用户ID: %s\n会话ID: %s\n线程ID: %s\nAgent: %s",
			userID, sessionID, contextState.ThreadID, agentName)
		promptParts = append(promptParts, systemInfo)
	}

	// 添加对话历史
	if includeHistory && len(contextState.ConversationHistory) > 0 {
		historyText := "对话历史:\n"
		for i, msg := range contextState.ConversationHistory {
			historyText += fmt.Sprintf("%d. %v\n", i+1, msg)
		}
		promptParts = append(promptParts, historyText)
	}

	// 添加创建的notes
	if includeNotes && len(contextState.CreatedNotes) > 0 {
		notesText := "创建的Notes:\n"
		for i, note := range contextState.CreatedNotes {
			notesText += fmt.Sprintf("%d. %v\n", i+1, note)
		}
		promptParts = append(promptParts, notesText)
	}

	// 添加用户选择
	if includeSelections && len(selections) > 0 {
		selectionsText := "用户选择:\n"
		for i, selection := range selections {
			selectionsText += fmt.Sprintf("%d. %s\n", i+1, selection)
		}
		promptParts = append(promptParts, selectionsText)
	}

	// 添加全局上下文
	if len(contextState.GlobalContext) > 0 {
		globalText := "全局上下文:\n"
		for key, value := range contextState.GlobalContext {
			globalText += fmt.Sprintf("- %s: %v\n", key, value)
		}
		promptParts = append(promptParts, globalText)
	}

	// 添加Agent特定上下文
	if agentContext, exists := contextState.AgentContexts[agentName]; exists {
		agentText := fmt.Sprintf("Agent上下文 (%s):\n%v\n", agentName, agentContext)
		promptParts = append(promptParts, agentText)
	}

	// 添加共享内存
	if len(contextState.SharedMemory) > 0 {
		sharedText := "共享内存:\n"
		for key, value := range contextState.SharedMemory {
			sharedText += fmt.Sprintf("- %s: %v\n", key, value)
		}
		promptParts = append(promptParts, sharedText)
	}

	// 添加调试信息
	if includeDebug {
		debugText := fmt.Sprintf("调试信息:\n- 用户消息队列长度: %d\n- Orchestrator调用次数: %d\n- 创建的Notes数量: %d\n- 最后更新: %s",
			len(contextState.UserMessageQueue),
			len(contextState.OrchestratorCalls),
			len(contextState.CreatedNotes),
			contextState.UpdatedAt.Format(time.RFC3339))
		promptParts = append(promptParts, debugText)
	}

	// 组合所有部分
	formattedPrompt := ""
	for i, part := range promptParts {
		if i > 0 {
			formattedPrompt += "\n"
		}
		formattedPrompt += part
	}

	lcm.logger.Info(ctx, "上下文格式化完成",
		"user_id", userID,
		"session_id", sessionID,
		"agent_name", agentName,
		"prompt_length", len(formattedPrompt))

	return formattedPrompt, nil
}

// ClearContext 清理上下文状态
func (lcm *LoomiContextManager) ClearContext(ctx context.Context, userID, sessionID string) error {
	contextKey := lcm.getContextKey(userID, sessionID)

	// 从内存删除
	delete(lcm.contexts, contextKey)

	// 从Redis删除
	err := lcm.deleteContextFromRedis(ctx, userID, sessionID)
	if err != nil {
		lcm.logger.Error(ctx, "从Redis删除上下文失败", "error", err)
		// 不返回错误，内存版本已删除
	}

	lcm.logger.Info(ctx, "上下文状态清理完成", "context_key", contextKey)
	return nil
}

// GetContextStatistics 获取上下文统计信息
func (lcm *LoomiContextManager) GetContextStatistics(ctx context.Context, userID, sessionID string) (map[string]interface{}, error) {
	contextState, err := lcm.GetContext(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取上下文失败: %w", err)
	}

	stats := map[string]interface{}{
		"user_id":                     userID,
		"session_id":                  sessionID,
		"thread_id":                   contextState.ThreadID,
		"user_message_queue_length":   len(contextState.UserMessageQueue),
		"orchestrator_calls_count":    len(contextState.OrchestratorCalls),
		"created_notes_count":         len(contextState.CreatedNotes),
		"conversation_history_length": len(contextState.ConversationHistory),
		"global_context_keys":         len(contextState.GlobalContext),
		"agent_contexts_keys":         len(contextState.AgentContexts),
		"shared_memory_keys":          len(contextState.SharedMemory),
		"created_at":                  contextState.CreatedAt,
		"updated_at":                  contextState.UpdatedAt,
	}

	return stats, nil
}

// 辅助方法

// getContextKey 获取上下文键
func (lcm *LoomiContextManager) getContextKey(userID, sessionID string) string {
	return fmt.Sprintf("loomi_context:%s:%s", userID, sessionID)
}

// saveContextToRedis 保存上下文到Redis
func (lcm *LoomiContextManager) saveContextToRedis(ctx context.Context, contextState *LoomiContextState) error {
	if lcm.redisManager == nil {
		return nil
	}

	client, err := lcm.redisManager.GetClient("high_priority")
	if err != nil {
		return fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 序列化上下文状态
	data, err := json.Marshal(contextState)
	if err != nil {
		return fmt.Errorf("序列化上下文状态失败: %w", err)
	}

	// 构建Redis键
	key := lcm.getContextKey(contextState.UserID, contextState.SessionID)

	// 保存到Redis（这里需要根据实际的Redis客户端类型来实现）
	_ = client
	_ = key
	_ = data

	lcm.logger.Debug(ctx, "上下文状态保存到Redis", "key", key, "data_length", len(data))
	return nil
}

// loadContextFromRedis 从Redis加载上下文
func (lcm *LoomiContextManager) loadContextFromRedis(ctx context.Context, userID, sessionID string) (*LoomiContextState, error) {
	if lcm.redisManager == nil {
		return nil, nil
	}

	client, err := lcm.redisManager.GetClient("high_priority")
	if err != nil {
		return nil, fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 构建Redis键
	key := lcm.getContextKey(userID, sessionID)

	// 从Redis获取数据（这里需要根据实际的Redis客户端类型来实现）
	_ = client
	_ = key

	lcm.logger.Debug(ctx, "从Redis加载上下文", "key", key)
	return nil, nil
}

// deleteContextFromRedis 从Redis删除上下文
func (lcm *LoomiContextManager) deleteContextFromRedis(ctx context.Context, userID, sessionID string) error {
	if lcm.redisManager == nil {
		return nil
	}

	client, err := lcm.redisManager.GetClient("high_priority")
	if err != nil {
		return fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 构建Redis键
	key := lcm.getContextKey(userID, sessionID)

	// 从Redis删除数据（这里需要根据实际的Redis客户端类型来实现）
	_ = client
	_ = key

	lcm.logger.Debug(ctx, "从Redis删除上下文", "key", key)
	return nil
}
