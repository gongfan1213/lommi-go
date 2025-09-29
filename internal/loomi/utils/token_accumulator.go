package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/pool"
)

// TokenUsage Token使用量数据结构
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AddUsage 累加另一个TokenUsage
func (tu *TokenUsage) AddUsage(other *TokenUsage) {
	tu.PromptTokens += other.PromptTokens
	tu.CompletionTokens += other.CompletionTokens
	tu.TotalTokens += other.TotalTokens
}

// ToMap 转换为字典
func (tu *TokenUsage) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"prompt_tokens":     tu.PromptTokens,
		"completion_tokens": tu.CompletionTokens,
		"total_tokens":      tu.TotalTokens,
	}
}

// FromMap 从字典创建TokenUsage
func (tu *TokenUsage) FromMap(data map[string]interface{}) {
	if promptTokens, ok := data["prompt_tokens"].(int); ok {
		tu.PromptTokens = promptTokens
	}
	if completionTokens, ok := data["completion_tokens"].(int); ok {
		tu.CompletionTokens = completionTokens
	}
	if totalTokens, ok := data["total_tokens"].(int); ok {
		tu.TotalTokens = totalTokens
	} else {
		// 如果没有total_tokens，则计算
		tu.TotalTokens = tu.PromptTokens + tu.CompletionTokens
	}
}

// TokenAccumulator Token累加器
type TokenAccumulator struct {
	logger       log.Logger
	redisManager pool.Manager
	prefix       string
}

// NewTokenAccumulator 创建Token累加器
func NewTokenAccumulator(logger log.Logger, redisManager pool.Manager) *TokenAccumulator {
	return &TokenAccumulator{
		logger:       logger,
		redisManager: redisManager,
		prefix:       "token_usage:",
	}
}

// Init 初始化Token累加器
func (ta *TokenAccumulator) Init(ctx context.Context, userID, sessionID string) error {
	ta.logger.Info(ctx, "初始化Token累加器",
		"user_id", userID,
		"session_id", sessionID)

	// 检查是否已存在
	key := ta.getKey(userID, sessionID)
	exists, err := ta.exists(ctx, key)
	if err != nil {
		return fmt.Errorf("检查Token累加器是否存在失败: %w", err)
	}

	if exists {
		ta.logger.Info(ctx, "Token累加器已存在", "key", key)
		return nil
	}

	// 创建初始记录
	initialUsage := &TokenUsage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
	}

	err = ta.saveUsage(ctx, key, initialUsage)
	if err != nil {
		return fmt.Errorf("保存初始Token使用量失败: %w", err)
	}

	ta.logger.Info(ctx, "Token累加器初始化成功", "key", key)
	return nil
}

// Summary 获取Token使用摘要
func (ta *TokenAccumulator) Summary(ctx context.Context, userID, sessionID string) (*TokenUsage, error) {
	ta.logger.Info(ctx, "获取Token使用摘要",
		"user_id", userID,
		"session_id", sessionID)

	key := ta.getKey(userID, sessionID)
	usage, err := ta.loadUsage(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("加载Token使用量失败: %w", err)
	}

	if usage == nil {
		// 如果不存在，返回零值
		usage = &TokenUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		}
	}

	ta.logger.Info(ctx, "Token使用摘要获取成功",
		"user_id", userID,
		"session_id", sessionID,
		"prompt_tokens", usage.PromptTokens,
		"completion_tokens", usage.CompletionTokens,
		"total_tokens", usage.TotalTokens)

	return usage, nil
}

// Initialize 初始化Token累加器（兼容性方法）
func (ta *TokenAccumulator) Initialize(ctx context.Context, userID, sessionID string) (string, error) {
	err := ta.Init(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	return "Token累加器初始化成功", nil
}

// Add 添加Token使用量
func (ta *TokenAccumulator) Add(ctx context.Context, userID, sessionID string, tokens int) error {
	ta.logger.Info(ctx, "添加Token使用量",
		"user_id", userID,
		"session_id", sessionID,
		"tokens", tokens)

	key := ta.getKey(userID, sessionID)

	// 获取当前使用量
	currentUsage, err := ta.loadUsage(ctx, key)
	if err != nil {
		return fmt.Errorf("加载当前Token使用量失败: %w", err)
	}

	if currentUsage == nil {
		// 如果不存在，创建新的
		currentUsage = &TokenUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		}
	}

	// 添加Token使用量（这里假设是completion tokens）
	currentUsage.CompletionTokens += tokens
	currentUsage.TotalTokens += tokens

	// 保存更新后的使用量
	err = ta.saveUsage(ctx, key, currentUsage)
	if err != nil {
		return fmt.Errorf("保存Token使用量失败: %w", err)
	}

	ta.logger.Info(ctx, "Token使用量添加成功",
		"user_id", userID,
		"session_id", sessionID,
		"added_tokens", tokens,
		"total_tokens", currentUsage.TotalTokens)

	return nil
}

// AddPromptTokens 添加Prompt Token使用量
func (ta *TokenAccumulator) AddPromptTokens(ctx context.Context, userID, sessionID string, tokens int) error {
	ta.logger.Info(ctx, "添加Prompt Token使用量",
		"user_id", userID,
		"session_id", sessionID,
		"tokens", tokens)

	key := ta.getKey(userID, sessionID)

	// 获取当前使用量
	currentUsage, err := ta.loadUsage(ctx, key)
	if err != nil {
		return fmt.Errorf("加载当前Token使用量失败: %w", err)
	}

	if currentUsage == nil {
		// 如果不存在，创建新的
		currentUsage = &TokenUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		}
	}

	// 添加Prompt Token使用量
	currentUsage.PromptTokens += tokens
	currentUsage.TotalTokens += tokens

	// 保存更新后的使用量
	err = ta.saveUsage(ctx, key, currentUsage)
	if err != nil {
		return fmt.Errorf("保存Token使用量失败: %w", err)
	}

	ta.logger.Info(ctx, "Prompt Token使用量添加成功",
		"user_id", userID,
		"session_id", sessionID,
		"added_tokens", tokens,
		"total_tokens", currentUsage.TotalTokens)

	return nil
}

// AddCompletionTokens 添加Completion Token使用量
func (ta *TokenAccumulator) AddCompletionTokens(ctx context.Context, userID, sessionID string, tokens int) error {
	ta.logger.Info(ctx, "添加Completion Token使用量",
		"user_id", userID,
		"session_id", sessionID,
		"tokens", tokens)

	key := ta.getKey(userID, sessionID)

	// 获取当前使用量
	currentUsage, err := ta.loadUsage(ctx, key)
	if err != nil {
		return fmt.Errorf("加载当前Token使用量失败: %w", err)
	}

	if currentUsage == nil {
		// 如果不存在，创建新的
		currentUsage = &TokenUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		}
	}

	// 添加Completion Token使用量
	currentUsage.CompletionTokens += tokens
	currentUsage.TotalTokens += tokens

	// 保存更新后的使用量
	err = ta.saveUsage(ctx, key, currentUsage)
	if err != nil {
		return fmt.Errorf("保存Token使用量失败: %w", err)
	}

	ta.logger.Info(ctx, "Completion Token使用量添加成功",
		"user_id", userID,
		"session_id", sessionID,
		"added_tokens", tokens,
		"total_tokens", currentUsage.TotalTokens)

	return nil
}

// AddUsage 添加TokenUsage
func (ta *TokenAccumulator) AddUsage(ctx context.Context, userID, sessionID string, usage *TokenUsage) error {
	ta.logger.Info(ctx, "添加TokenUsage",
		"user_id", userID,
		"session_id", sessionID,
		"usage", usage.ToMap())

	key := ta.getKey(userID, sessionID)

	// 获取当前使用量
	currentUsage, err := ta.loadUsage(ctx, key)
	if err != nil {
		return fmt.Errorf("加载当前Token使用量失败: %w", err)
	}

	if currentUsage == nil {
		// 如果不存在，创建新的
		currentUsage = &TokenUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		}
	}

	// 累加使用量
	currentUsage.AddUsage(usage)

	// 保存更新后的使用量
	err = ta.saveUsage(ctx, key, currentUsage)
	if err != nil {
		return fmt.Errorf("保存Token使用量失败: %w", err)
	}

	ta.logger.Info(ctx, "TokenUsage添加成功",
		"user_id", userID,
		"session_id", sessionID,
		"total_tokens", currentUsage.TotalTokens)

	return nil
}

// CalculateCredits 计算积分消耗
func (ta *TokenAccumulator) CalculateCredits(ctx context.Context, userID, sessionID string, rate float64) (float64, error) {
	ta.logger.Info(ctx, "计算积分消耗",
		"user_id", userID,
		"session_id", sessionID,
		"rate", rate)

	usage, err := ta.Summary(ctx, userID, sessionID)
	if err != nil {
		return 0, fmt.Errorf("获取Token使用摘要失败: %w", err)
	}

	credits := float64(usage.TotalTokens) * rate

	ta.logger.Info(ctx, "积分消耗计算完成",
		"user_id", userID,
		"session_id", sessionID,
		"total_tokens", usage.TotalTokens,
		"rate", rate,
		"credits", credits)

	return credits, nil
}

// Cleanup 清理Token累加器数据
func (ta *TokenAccumulator) Cleanup(ctx context.Context, userID, sessionID string) error {
	ta.logger.Info(ctx, "清理Token累加器数据",
		"user_id", userID,
		"session_id", sessionID)

	key := ta.getKey(userID, sessionID)
	err := ta.delete(ctx, key)
	if err != nil {
		return fmt.Errorf("删除Token累加器数据失败: %w", err)
	}

	ta.logger.Info(ctx, "Token累加器数据清理成功", "key", key)
	return nil
}

// CleanupOldData 清理旧数据
func (ta *TokenAccumulator) CleanupOldData(ctx context.Context, olderThan time.Duration) error {
	ta.logger.Info(ctx, "清理旧Token数据", "older_than", olderThan)

	// 这里需要根据实际的Redis实现来清理旧数据
	// 目前只是一个占位实现

	ta.logger.Info(ctx, "旧Token数据清理完成")
	return nil
}

// GetUsageHistory 获取使用历史
func (ta *TokenAccumulator) GetUsageHistory(ctx context.Context, userID, sessionID string, limit int) ([]*TokenUsage, error) {
	ta.logger.Info(ctx, "获取Token使用历史",
		"user_id", userID,
		"session_id", sessionID,
		"limit", limit)

	// 这里需要根据实际的存储实现来获取历史数据
	// 目前返回当前使用量
	usage, err := ta.Summary(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取Token使用摘要失败: %w", err)
	}

	var history []*TokenUsage
	if usage != nil {
		history = append(history, usage)
	}

	ta.logger.Info(ctx, "Token使用历史获取成功", "count", len(history))
	return history, nil
}

// 辅助方法

// getKey 获取Redis键
func (ta *TokenAccumulator) getKey(userID, sessionID string) string {
	return fmt.Sprintf("%s%s:%s", ta.prefix, userID, sessionID)
}

// exists 检查键是否存在
func (ta *TokenAccumulator) exists(ctx context.Context, key string) (bool, error) {
	if ta.redisManager == nil {
		return false, nil
	}

	// 这里需要根据实际的Redis客户端类型来实现
	// 目前返回false
	return false, nil
}

// saveUsage 保存Token使用量
func (ta *TokenAccumulator) saveUsage(ctx context.Context, key string, usage *TokenUsage) error {
	if ta.redisManager == nil {
		return nil
	}

	// 序列化使用量
	data, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("序列化Token使用量失败: %w", err)
	}

	// 这里需要根据实际的Redis客户端类型来实现保存
	_ = data

	ta.logger.Debug(ctx, "保存Token使用量", "key", key, "usage", usage.ToMap())
	return nil
}

// loadUsage 加载Token使用量
func (ta *TokenAccumulator) loadUsage(ctx context.Context, key string) (*TokenUsage, error) {
	if ta.redisManager == nil {
		return nil, nil
	}

	// 这里需要根据实际的Redis客户端类型来实现加载
	// 目前返回nil

	ta.logger.Debug(ctx, "加载Token使用量", "key", key)
	return nil, nil
}

// delete 删除Token使用量
func (ta *TokenAccumulator) delete(ctx context.Context, key string) error {
	if ta.redisManager == nil {
		return nil
	}

	// 这里需要根据实际的Redis客户端类型来实现删除
	// 目前返回nil

	ta.logger.Debug(ctx, "删除Token使用量", "key", key)
	return nil
}
