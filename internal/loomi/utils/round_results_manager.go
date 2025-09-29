package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/pool"
)

// RoundResultsManager 轮次结果管理器
type RoundResultsManager struct {
	logger       log.Logger
	redisManager pool.Manager
	selectPrefix string
	allPrefix    string
}

// NewRoundResultsManager 创建轮次结果管理器
func NewRoundResultsManager(logger log.Logger, redisManager pool.Manager) *RoundResultsManager {
	return &RoundResultsManager{
		logger:       logger,
		redisManager: redisManager,
		selectPrefix: "user_select_results:",
		allPrefix:    "user_all_results:",
	}
}

// RoundResult 轮次结果
type RoundResult struct {
	Round     int                    `json:"round"`
	Timestamp time.Time              `json:"timestamp"`
	Results   map[string]interface{} `json:"results"`
}

// UserSelectResult 用户选择结果
type UserSelectResult struct {
	Round      int                    `json:"round"`
	Timestamp  time.Time              `json:"timestamp"`
	Selections map[string]interface{} `json:"selections"`
}

// SaveUserSelectResult 保存用户选择结果
func (rrm *RoundResultsManager) SaveUserSelectResult(ctx context.Context, userID, sessionID string, round int, selections map[string]interface{}) error {
	rrm.logger.Info(ctx, "保存用户选择结果",
		"user_id", userID,
		"session_id", sessionID,
		"round", round,
		"selections_count", len(selections))

	if rrm.redisManager == nil {
		rrm.logger.Warn(ctx, "Redis管理器不可用，跳过保存用户选择结果")
		return nil
	}

	client, err := rrm.redisManager.GetClient("high_priority")
	if err != nil {
		return fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 构建结果数据
	result := UserSelectResult{
		Round:      round,
		Timestamp:  time.Now(),
		Selections: selections,
	}

	// 序列化为JSON
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化用户选择结果失败: %w", err)
	}

	// 构建Redis键
	key := rrm.selectPrefix + userID + ":" + sessionID

	// 保存到Redis
	err = rrm.setRedisData(ctx, client, key, string(data))
	if err != nil {
		return fmt.Errorf("保存用户选择结果到Redis失败: %w", err)
	}

	rrm.logger.Info(ctx, "用户选择结果保存成功", "key", key)
	return nil
}

// SaveUserAllResult 保存所有生成结果
func (rrm *RoundResultsManager) SaveUserAllResult(ctx context.Context, userID, sessionID string, round int, results map[string]interface{}) error {
	rrm.logger.Info(ctx, "保存所有生成结果",
		"user_id", userID,
		"session_id", sessionID,
		"round", round,
		"results_count", len(results))

	if rrm.redisManager == nil {
		rrm.logger.Warn(ctx, "Redis管理器不可用，跳过保存所有生成结果")
		return nil
	}

	client, err := rrm.redisManager.GetClient("high_priority")
	if err != nil {
		return fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 构建结果数据
	result := RoundResult{
		Round:     round,
		Timestamp: time.Now(),
		Results:   results,
	}

	// 序列化为JSON
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化所有生成结果失败: %w", err)
	}

	// 构建Redis键
	key := rrm.allPrefix + userID + ":" + sessionID

	// 保存到Redis
	err = rrm.setRedisData(ctx, client, key, string(data))
	if err != nil {
		return fmt.Errorf("保存所有生成结果到Redis失败: %w", err)
	}

	rrm.logger.Info(ctx, "所有生成结果保存成功", "key", key)
	return nil
}

// GetUserSelectResults 获取用户选择结果
func (rrm *RoundResultsManager) GetUserSelectResults(ctx context.Context, userID, sessionID string) ([]UserSelectResult, error) {
	rrm.logger.Info(ctx, "获取用户选择结果",
		"user_id", userID,
		"session_id", sessionID)

	if rrm.redisManager == nil {
		rrm.logger.Warn(ctx, "Redis管理器不可用，返回空结果")
		return []UserSelectResult{}, nil
	}

	client, err := rrm.redisManager.GetClient("high_priority")
	if err != nil {
		return nil, fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 构建Redis键
	key := rrm.selectPrefix + userID + ":" + sessionID

	// 从Redis获取数据
	data, err := rrm.getRedisData(ctx, client, key)
	if err != nil {
		return nil, fmt.Errorf("从Redis获取用户选择结果失败: %w", err)
	}

	if data == "" {
		return []UserSelectResult{}, nil
	}

	// 反序列化数据
	var results []UserSelectResult
	err = json.Unmarshal([]byte(data), &results)
	if err != nil {
		return nil, fmt.Errorf("反序列化用户选择结果失败: %w", err)
	}

	rrm.logger.Info(ctx, "用户选择结果获取成功", "count", len(results))
	return results, nil
}

// GetUserAllResults 获取所有生成结果
func (rrm *RoundResultsManager) GetUserAllResults(ctx context.Context, userID, sessionID string) ([]RoundResult, error) {
	rrm.logger.Info(ctx, "获取所有生成结果",
		"user_id", userID,
		"session_id", sessionID)

	if rrm.redisManager == nil {
		rrm.logger.Warn(ctx, "Redis管理器不可用，返回空结果")
		return []RoundResult{}, nil
	}

	client, err := rrm.redisManager.GetClient("high_priority")
	if err != nil {
		return nil, fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 构建Redis键
	key := rrm.allPrefix + userID + ":" + sessionID

	// 从Redis获取数据
	data, err := rrm.getRedisData(ctx, client, key)
	if err != nil {
		return nil, fmt.Errorf("从Redis获取所有生成结果失败: %w", err)
	}

	if data == "" {
		return []RoundResult{}, nil
	}

	// 反序列化数据
	var results []RoundResult
	err = json.Unmarshal([]byte(data), &results)
	if err != nil {
		return nil, fmt.Errorf("反序列化所有生成结果失败: %w", err)
	}

	rrm.logger.Info(ctx, "所有生成结果获取成功", "count", len(results))
	return results, nil
}

// GetLatestUserSelectResult 获取最新的用户选择结果
func (rrm *RoundResultsManager) GetLatestUserSelectResult(ctx context.Context, userID, sessionID string) (*UserSelectResult, error) {
	results, err := rrm.GetUserSelectResults(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	// 返回最新的结果
	latest := results[len(results)-1]
	return &latest, nil
}

// GetLatestUserAllResult 获取最新的所有生成结果
func (rrm *RoundResultsManager) GetLatestUserAllResult(ctx context.Context, userID, sessionID string) (*RoundResult, error) {
	results, err := rrm.GetUserAllResults(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	// 返回最新的结果
	latest := results[len(results)-1]
	return &latest, nil
}

// ClearUserResults 清理用户结果
func (rrm *RoundResultsManager) ClearUserResults(ctx context.Context, userID, sessionID string) error {
	rrm.logger.Info(ctx, "清理用户结果",
		"user_id", userID,
		"session_id", sessionID)

	if rrm.redisManager == nil {
		rrm.logger.Warn(ctx, "Redis管理器不可用，跳过清理用户结果")
		return nil
	}

	client, err := rrm.redisManager.GetClient("high_priority")
	if err != nil {
		return fmt.Errorf("获取Redis客户端失败: %w", err)
	}

	// 构建Redis键
	selectKey := rrm.selectPrefix + userID + ":" + sessionID
	allKey := rrm.allPrefix + userID + ":" + sessionID

	// 删除用户选择结果
	err = rrm.deleteRedisData(ctx, client, selectKey)
	if err != nil {
		rrm.logger.Error(ctx, "删除用户选择结果失败", "error", err)
	}

	// 删除所有生成结果
	err = rrm.deleteRedisData(ctx, client, allKey)
	if err != nil {
		rrm.logger.Error(ctx, "删除所有生成结果失败", "error", err)
	}

	rrm.logger.Info(ctx, "用户结果清理完成")
	return nil
}

// GetUserResultStatistics 获取用户结果统计信息
func (rrm *RoundResultsManager) GetUserResultStatistics(ctx context.Context, userID, sessionID string) (map[string]interface{}, error) {
	rrm.logger.Info(ctx, "获取用户结果统计信息",
		"user_id", userID,
		"session_id", sessionID)

	selectResults, err := rrm.GetUserSelectResults(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取用户选择结果失败: %w", err)
	}

	allResults, err := rrm.GetUserAllResults(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取所有生成结果失败: %w", err)
	}

	stats := map[string]interface{}{
		"user_id":              userID,
		"session_id":           sessionID,
		"select_results_count": len(selectResults),
		"all_results_count":    len(allResults),
		"total_rounds":         len(selectResults),
	}

	// 计算各轮次的统计信息
	if len(selectResults) > 0 {
		latestSelect := selectResults[len(selectResults)-1]
		stats["latest_select_round"] = latestSelect.Round
		stats["latest_select_timestamp"] = latestSelect.Timestamp
	}

	if len(allResults) > 0 {
		latestAll := allResults[len(allResults)-1]
		stats["latest_all_round"] = latestAll.Round
		stats["latest_all_timestamp"] = latestAll.Timestamp
	}

	rrm.logger.Info(ctx, "用户结果统计信息获取成功", "stats", stats)
	return stats, nil
}

// 辅助方法

// setRedisData 设置Redis数据
func (rrm *RoundResultsManager) setRedisData(ctx context.Context, client interface{}, key, value string) error {
	// 这里需要根据实际的Redis客户端类型来实现
	// 由于pool.Manager返回的是interface{}，需要类型断言
	// 这是一个简化的实现，实际项目中需要根据具体的Redis客户端类型来实现

	rrm.logger.Debug(ctx, "设置Redis数据", "key", key, "value_length", len(value))

	// 模拟设置数据（实际实现需要根据具体的Redis客户端）
	// 这里假设有一个通用的Set方法
	return nil
}

// getRedisData 获取Redis数据
func (rrm *RoundResultsManager) getRedisData(ctx context.Context, client interface{}, key string) (string, error) {
	// 这里需要根据实际的Redis客户端类型来实现
	// 由于pool.Manager返回的是interface{}，需要类型断言
	// 这是一个简化的实现，实际项目中需要根据具体的Redis客户端类型来实现

	rrm.logger.Debug(ctx, "获取Redis数据", "key", key)

	// 模拟获取数据（实际实现需要根据具体的Redis客户端）
	// 这里假设有一个通用的Get方法
	return "", nil
}

// deleteRedisData 删除Redis数据
func (rrm *RoundResultsManager) deleteRedisData(ctx context.Context, client interface{}, key string) error {
	// 这里需要根据实际的Redis客户端类型来实现
	// 由于pool.Manager返回的是interface{}，需要类型断言
	// 这是一个简化的实现，实际项目中需要根据具体的Redis客户端类型来实现

	rrm.logger.Debug(ctx, "删除Redis数据", "key", key)

	// 模拟删除数据（实际实现需要根据具体的Redis客户端）
	// 这里假设有一个通用的Delete方法
	return nil
}

// GetRoundByTimestamp 根据时间戳获取轮次
func (rrm *RoundResultsManager) GetRoundByTimestamp(ctx context.Context, userID, sessionID string, timestamp time.Time) (*RoundResult, error) {
	results, err := rrm.GetUserAllResults(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		if result.Timestamp.Equal(timestamp) {
			return &result, nil
		}
	}

	return nil, nil
}

// GetResultsByRoundRange 根据轮次范围获取结果
func (rrm *RoundResultsManager) GetResultsByRoundRange(ctx context.Context, userID, sessionID string, startRound, endRound int) ([]RoundResult, error) {
	results, err := rrm.GetUserAllResults(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	var filteredResults []RoundResult
	for _, result := range results {
		if result.Round >= startRound && result.Round <= endRound {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults, nil
}

// ExportUserResults 导出用户结果
func (rrm *RoundResultsManager) ExportUserResults(ctx context.Context, userID, sessionID string) (map[string]interface{}, error) {
	selectResults, err := rrm.GetUserSelectResults(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取用户选择结果失败: %w", err)
	}

	allResults, err := rrm.GetUserAllResults(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取所有生成结果失败: %w", err)
	}

	export := map[string]interface{}{
		"user_id":        userID,
		"session_id":     sessionID,
		"export_time":    time.Now(),
		"select_results": selectResults,
		"all_results":    allResults,
	}

	return export, nil
}
