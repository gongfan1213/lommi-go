package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/llm"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// ComfortMessageTool 安慰消息生成工具
type ComfortMessageTool struct {
	config       *config.Config
	logger       log.Logger
	llmClient    llm.Client
	comfortModel string
	temperature  float64
	maxTokens    int
	topP         float64
}

// NewComfortMessageTool 创建安慰消息工具
func NewComfortMessageTool(cfg *config.Config, logger log.Logger) (*ComfortMessageTool, error) {
	// 读取comfort消息专用配置
	comfortModel := getEnv("OPENAI_MODEL_COMFORT_MESSAGE", "qwen-turbo")
	temperature := getEnvFloat("OPENAI_TEMPERATURE_COMFORT_MESSAGE", 0.1)
	maxTokens := getEnvInt("OPENAI_MAX_TOKENS_COMFORT_MESSAGE", 1024)
	topP := getEnvFloat("OPENAI_TOP_P_COMFORT_MESSAGE", 0.1)

	// 根据LLM_DEFAULT_PROVIDER设置选择正确的LLM客户端
	defaultProvider := getEnv("LLM_DEFAULT_PROVIDER", "openai")

	// 创建LLM客户端
	llmClient, err := llm.NewClient(defaultProvider, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	tool := &ComfortMessageTool{
		config:       cfg,
		logger:       logger,
		llmClient:    llmClient,
		comfortModel: comfortModel,
		temperature:  temperature,
		maxTokens:    maxTokens,
		topP:         topP,
	}

	logger.Info(context.Background(), "Comfort消息工具初始化完成",
		"model", comfortModel,
		"temperature", temperature,
		"max_tokens", maxTokens,
		"top_p", topP,
		"provider", defaultProvider)

	return tool, nil
}

// GenerateFirstComfort 生成首次输入的安慰消息
func (c *ComfortMessageTool) GenerateFirstComfort(ctx context.Context, userInput string) (string, error) {
	c.logger.Info(ctx, "生成首次安慰消息", "user_input", userInput[:min(100, len(userInput))])

	// 使用首次安慰消息模板
	prompt := fmt.Sprintf(`你是一个专业的AI助手，擅长为用户提供温暖、贴心的回复。

用户刚刚输入了以下内容：
"%s"

请生成一条简短、温暖、鼓励性的安慰消息，让用户感受到被理解和关怀。消息应该：
1. 简洁明了，不超过50字
2. 语气温和、鼓励性
3. 表达对用户需求的理解
4. 给予积极的支持

请直接回复安慰消息，不要包含其他内容：`, userInput)

	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	// 调用LLM生成安慰消息
	var response string
	err := c.llmClient.SafeStreamCall(ctx, "comfort", "first", messages, func(ctx context.Context, chunk string) error {
		response += chunk
		return nil
	})

	if err != nil {
		c.logger.Error(ctx, "生成首次安慰消息失败", "error", err)
		return "", fmt.Errorf("生成首次安慰消息失败: %w", err)
	}

	c.logger.Info(ctx, "首次安慰消息生成成功", "response_length", len(response))
	return response, nil
}

// GenerateComfort 生成后续安慰消息
func (c *ComfortMessageTool) GenerateComfort(ctx context.Context, userInput string, contextInfo string) (string, error) {
	c.logger.Info(ctx, "生成后续安慰消息",
		"user_input", userInput[:min(100, len(userInput))],
		"context_info", contextInfo[:min(100, len(contextInfo))])

	// 使用后续安慰消息模板
	prompt := fmt.Sprintf(`你是一个专业的AI助手，擅长为用户提供温暖、贴心的回复。

用户刚刚输入了以下内容：
"%s"

当前上下文信息：
"%s"

请生成一条简短、温暖、鼓励性的安慰消息，让用户感受到被理解和关怀。消息应该：
1. 简洁明了，不超过50字
2. 语气温和、鼓励性
3. 结合上下文信息，表达对用户需求的理解
4. 给予积极的支持和鼓励

请直接回复安慰消息，不要包含其他内容：`, userInput, contextInfo)

	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	// 调用LLM生成安慰消息
	var response string
	err := c.llmClient.SafeStreamCall(ctx, "comfort", "followup", messages, func(ctx context.Context, chunk string) error {
		response += chunk
		return nil
	})

	if err != nil {
		c.logger.Error(ctx, "生成后续安慰消息失败", "error", err)
		return "", fmt.Errorf("生成后续安慰消息失败: %w", err)
	}

	c.logger.Info(ctx, "后续安慰消息生成成功", "response_length", len(response))
	return response, nil
}

// GenerateErrorComfort 生成错误安慰消息
func (c *ComfortMessageTool) GenerateErrorComfort(ctx context.Context, errorMsg string, userInput string) (string, error) {
	c.logger.Info(ctx, "生成错误安慰消息",
		"error_msg", errorMsg,
		"user_input", userInput[:min(100, len(userInput))])

	// 使用错误安慰消息模板
	prompt := fmt.Sprintf(`你是一个专业的AI助手，擅长为用户提供温暖、贴心的回复。

系统遇到了以下错误：
"%s"

用户刚刚输入了以下内容：
"%s"

请生成一条简短、温暖、鼓励性的安慰消息，让用户感受到被理解和支持。消息应该：
1. 简洁明了，不超过50字
2. 语气温和、鼓励性
3. 表达对用户的理解和支持
4. 给予积极的鼓励，让用户不要担心

请直接回复安慰消息，不要包含其他内容：`, errorMsg, userInput)

	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	// 调用LLM生成安慰消息
	var response string
	err := c.llmClient.SafeStreamCall(ctx, "comfort", "error", messages, func(ctx context.Context, chunk string) error {
		response += chunk
		return nil
	})

	if err != nil {
		c.logger.Error(ctx, "生成错误安慰消息失败", "error", err)
		return "", fmt.Errorf("生成错误安慰消息失败: %w", err)
	}

	c.logger.Info(ctx, "错误安慰消息生成成功", "response_length", len(response))
	return response, nil
}

// GenerateProgressComfort 生成进度安慰消息
func (c *ComfortMessageTool) GenerateProgressComfort(ctx context.Context, progress string, userInput string) (string, error) {
	c.logger.Info(ctx, "生成进度安慰消息",
		"progress", progress,
		"user_input", userInput[:min(100, len(userInput))])

	// 使用进度安慰消息模板
	prompt := fmt.Sprintf(`你是一个专业的AI助手，擅长为用户提供温暖、贴心的回复。

当前处理进度：
"%s"

用户刚刚输入了以下内容：
"%s"

请生成一条简短、温暖、鼓励性的安慰消息，让用户感受到被理解和支持。消息应该：
1. 简洁明了，不超过50字
2. 语气温和、鼓励性
3. 表达对进度的理解和鼓励
4. 给予积极的支持

请直接回复安慰消息，不要包含其他内容：`, progress, userInput)

	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	// 调用LLM生成安慰消息
	var response string
	err := c.llmClient.SafeStreamCall(ctx, "comfort", "progress", messages, func(ctx context.Context, chunk string) error {
		response += chunk
		return nil
	})

	if err != nil {
		c.logger.Error(ctx, "生成进度安慰消息失败", "error", err)
		return "", fmt.Errorf("生成进度安慰消息失败: %w", err)
	}

	c.logger.Info(ctx, "进度安慰消息生成成功", "response_length", len(response))
	return response, nil
}

// GetComfortConfig 获取安慰消息配置
func (c *ComfortMessageTool) GetComfortConfig() map[string]interface{} {
	return map[string]interface{}{
		"model":       c.comfortModel,
		"temperature": c.temperature,
		"max_tokens":  c.maxTokens,
		"top_p":       c.topP,
	}
}

// SetComfortConfig 设置安慰消息配置
func (c *ComfortMessageTool) SetComfortConfig(model string, temperature float64, maxTokens int, topP float64) {
	c.comfortModel = model
	c.temperature = temperature
	c.maxTokens = maxTokens
	c.topP = topP

	c.logger.Info(context.Background(), "安慰消息配置已更新",
		"model", model,
		"temperature", temperature,
		"max_tokens", maxTokens,
		"top_p", topP)
}

// 辅助函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
