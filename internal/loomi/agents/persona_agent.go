package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/base"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/types"
)

// PersonaAgent 受众画像智能体
type PersonaAgent struct {
	*base.BaseLoomiAgent
	xmlParser *XMLParser
}

// NewPersonaAgent 创建新的受众画像智能体
func NewPersonaAgent(logger log.Logger) *PersonaAgent {
	agent := &PersonaAgent{
		BaseLoomiAgent: base.NewBaseLoomiAgent("loomi_persona_agent", logger),
		xmlParser:      NewXMLParser(),
	}

	// 设置系统提示词
	agent.SetSystemPrompt(ACTION_PROMPTS["persona"])

	logger.Info(context.Background(), "PersonaAgent初始化完成")
	return agent
}

// ProcessRequest 处理受众画像请求
func (a *PersonaAgent) ProcessRequest(ctx context.Context, request *types.AgentRequest) (*types.AgentResponse, error) {
	query := request.Query
	instruction := request.Instruction
	if instruction == "" {
		instruction = query
	}
	userID := request.UserID
	sessionID := request.SessionID
	useFiles := request.UseFiles
	autoMode := request.AutoMode
	userSelections := request.UserSelections

	a.Logger.Info(ctx, "PersonaAgent处理请求",
		"instruction", instruction[:min(100, len(instruction))],
		"user_id", userID,
		"session_id", sessionID,
		"use_files", useFiles)

	// 设置当前会话信息
	a.SetCurrentSession(userID, sessionID)
	defer a.ClearCurrentSession()

	// 构建用户提示词
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, userID, sessionID, instruction, "persona", autoMode, userSelections)
	if err != nil {
		return nil, fmt.Errorf("构建用户提示词失败: %w", err)
	}

	// 构建消息
	messages := []types.Message{
		{Role: "system", Content: a.GetSystemPrompt()},
		{Role: "user", Content: userPrompt},
	}

	// 检查停止状态
	if err := a.CheckStopState(ctx, userID, sessionID); err != nil {
		return nil, err
	}

	// 流式处理LLM响应
	a.Logger.Info(ctx, "开始收集LLM响应",
		"user_id", userID,
		"session_id", sessionID)

	llmResponse, err := a.SafeStreamCall(ctx, userID, sessionID, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	a.Logger.Info(ctx, "LLM响应收集完成",
		"user_id", userID,
		"session_id", sessionID,
		"response_length", len(llmResponse))

	// 再次检查停止状态
	if err := a.CheckStopState(ctx, userID, sessionID); err != nil {
		return nil, err
	}

	// 检查是否有标签之外的其他内容
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<persona\d+>`})

	// 解析受众画像结果
	personas, err := a.ParsePersonasWithUniqueIDs(ctx, llmResponse, request, userID, sessionID)
	if err != nil {
		a.Logger.Error(ctx, "解析受众画像失败", "error", err)
		// 返回原始响应
		return &types.AgentResponse{
			Content: llmResponse,
			Type:    "persona",
		}, nil
	}

	a.Logger.Info(ctx, "受众画像解析完成",
		"user_id", userID,
		"session_id", sessionID,
		"personas_count", len(personas))

	// 构建响应
	response := &types.AgentResponse{
		Type:    "persona",
		Content: personas,
	}

	// 处理instruction，只保存|后面的部分
	displayInstruction := instruction
	if strings.Contains(instruction, "||") {
		parts := strings.Split(instruction, "||")
		displayInstruction = strings.TrimSpace(parts[len(parts)-1])
	}

	// 构建metadata
	metadata := map[string]interface{}{
		"instruction": displayInstruction,
	}
	if otherContent != "" {
		metadata["agent_other_message"] = otherContent
	}
	response.Metadata = metadata

	// 创建notes
	if len(personas) > 0 {
		a.Logger.Info(ctx, "开始批量并发创建notes",
			"user_id", userID,
			"session_id", sessionID,
			"count", len(personas))

		startTime := time.Now()

		// 并发创建notes
		for _, persona := range personas {
			err := a.CreateNote(ctx, userID, sessionID, "persona", persona["id"].(string),
				persona["content"].(string), persona["title"].(string), 0)
			if err != nil {
				a.Logger.Error(ctx, "创建note失败",
					"persona_id", persona["id"],
					"error", err)
			}
		}

		duration := time.Since(startTime)
		a.Logger.Info(ctx, "批量并发创建notes完成",
			"user_id", userID,
			"session_id", sessionID,
			"count", len(personas),
			"duration", duration)
	}

	return response, nil
}

// ParsePersonasWithUniqueIDs 解析LLM响应中的受众画像结果，每个画像单独获取唯一ID
func (a *PersonaAgent) ParsePersonasWithUniqueIDs(ctx context.Context, response string, request *types.AgentRequest, userID, sessionID string) ([]map[string]interface{}, error) {
	// 使用XML解析器解析内容
	config := UNIFIED_CONFIGS["persona"]
	parseResults, err := a.xmlParser.Parse(response, config, 1)
	if err != nil {
		return nil, fmt.Errorf("XML解析失败: %w", err)
	}

	// 为每个画像单独获取会话内唯一ID
	var personas []map[string]interface{}
	for _, result := range parseResults {
		uniqueID, err := a.GetNextActionID(ctx, userID, sessionID, "persona")
		if err != nil {
			a.Logger.Error(ctx, "获取唯一ID失败", "error", err)
			uniqueID = time.Now().UnixNano() // 使用时间戳作为备用ID
		}

		// 确保markdown格式兼容性
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("受众画像 %d", len(personas)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		personas = append(personas, map[string]interface{}{
			"id":      fmt.Sprintf("persona%d", uniqueID),
			"title":   title,
			"content": content,
			"type":    result.Type,
		})
	}

	a.Logger.Info(ctx, "为受众画像分配了唯一ID",
		"count", len(personas),
		"ids", a.extractIDs(personas))

	return personas, nil
}

// ParsePersonas 解析LLM响应中的受众画像结果（保留原方法兼容性）
func (a *PersonaAgent) ParsePersonas(response string, request *types.AgentRequest, nextID int) []map[string]interface{} {
	// 使用统一格式XML解析器
	config := UNIFIED_CONFIGS["persona"]
	parseResults, err := a.xmlParser.Parse(response, config, nextID)
	if err != nil {
		a.Logger.Error(context.Background(), "XML解析失败", "error", err)
		return nil
	}

	// 转换为包含标题的格式
	var personas []map[string]interface{}
	for _, result := range parseResults {
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("受众画像 %d", len(personas)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		personas = append(personas, map[string]interface{}{
			"id":      result.ID,
			"title":   title,
			"content": content,
			"type":    result.Type,
		})
	}

	return personas
}

// CleanPersonaTags 清理文本中的persona标签，只保留思考过程
func (a *PersonaAgent) CleanPersonaTags(text string) string {
	// 使用通用XML解析器的清理方法
	return a.xmlParser.CleanXMLTags(text, "persona")
}

// ExtractOtherContent 提取标签之外的其他内容
func (a *PersonaAgent) ExtractOtherContent(response string, tags []string) string {
	// 创建正则表达式模式
	var patterns []string
	for _, tag := range tags {
		patterns = append(patterns, fmt.Sprintf("<%s[^>]*>.*?</%s>", tag, tag))
	}

	// 合并所有模式
	pattern := strings.Join(patterns, "|")
	re := regexp.MustCompile(`(?s)` + pattern)

	// 移除所有匹配的标签内容
	cleaned := re.ReplaceAllString(response, "")

	// 清理多余的空白字符
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(cleaned, "\n")

	return cleaned
}

// EnsureMarkdownCompatibility 确保markdown格式兼容性
func (a *PersonaAgent) EnsureMarkdownCompatibility(text string) string {
	if text == "" {
		return ""
	}

	// 基本的markdown兼容性处理
	// 转义特殊字符
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "\"", "\\\"")
	text = strings.ReplaceAll(text, "\n", "\\n")

	return text
}

// extractIDs 提取personas中的ID列表
func (a *PersonaAgent) extractIDs(personas []map[string]interface{}) []string {
	var ids []string
	for _, persona := range personas {
		if id, ok := persona["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
