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

// TikTokScriptAgent 抖音口播稿创作智能体
type TikTokScriptAgent struct {
	*base.BaseLoomiAgent
	xmlParser *XMLParser
}

// NewTikTokScriptAgent 创建新的抖音口播稿创作智能体
func NewTikTokScriptAgent(logger log.Logger) *TikTokScriptAgent {
	agent := &TikTokScriptAgent{
		BaseLoomiAgent: base.NewBaseLoomiAgent("loomi_tiktok_script_agent", logger),
		xmlParser:      NewXMLParser(),
	}

	// 设置系统提示词
	agent.SetSystemPrompt(ACTION_PROMPTS["tiktok_script"])

	logger.Info(context.Background(), "TikTokScriptAgent初始化完成")
	return agent
}

// ProcessRequest 处理抖音口播稿创作请求
func (a *TikTokScriptAgent) ProcessRequest(ctx context.Context, request *types.AgentRequest) (*types.AgentResponse, error) {
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

	a.Logger.Info(ctx, "TikTokScriptAgent处理请求",
		"instruction", instruction[:min(100, len(instruction))],
		"user_id", userID,
		"session_id", sessionID,
		"use_files", useFiles)

	// 设置当前会话信息
	a.SetCurrentSession(userID, sessionID)
	defer a.ClearCurrentSession()

	// 构建用户提示词
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, userID, sessionID, instruction, "tiktok_script", autoMode, userSelections)
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
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<tiktok_script\d+>`})

	// 解析抖音脚本结果
	scripts, err := a.ParseTikTokScriptsWithUniqueIDs(ctx, llmResponse, request, userID, sessionID)
	if err != nil {
		a.Logger.Error(ctx, "解析抖音脚本失败", "error", err)
		// 返回原始响应
		return &types.AgentResponse{
			Content: llmResponse,
			Type:    "tiktok_script",
		}, nil
	}

	a.Logger.Info(ctx, "抖音脚本解析完成",
		"user_id", userID,
		"session_id", sessionID,
		"scripts_count", len(scripts))

	// 构建响应
	response := &types.AgentResponse{
		Type:    "tiktok_script",
		Content: scripts,
	}

	// 构建metadata
	metadata := map[string]interface{}{
		"instruction": instruction,
	}
	if otherContent != "" {
		metadata["agent_other_message"] = otherContent
	}
	response.Metadata = metadata

	// 创建notes
	if len(scripts) > 0 {
		a.Logger.Info(ctx, "开始批量并发创建notes",
			"user_id", userID,
			"session_id", sessionID,
			"count", len(scripts))

		startTime := time.Now()

		// 并发创建notes
		for _, script := range scripts {
			// 只存储正文内容，让系统格式化处理标题和封面文案
			err := a.CreateNote(ctx, userID, sessionID, "tiktok_script",
				script["id"].(string), script["content"].(string),
				a.getStringValue(script, "title"), 0)
			if err != nil {
				a.Logger.Error(ctx, "创建note失败",
					"script_id", script["id"],
					"error", err)
			}
		}

		duration := time.Since(startTime)
		a.Logger.Info(ctx, "批量并发创建notes完成",
			"user_id", userID,
			"session_id", sessionID,
			"count", len(scripts),
			"duration", duration)
	}

	return response, nil
}

// ParseTikTokScriptsWithUniqueIDs 解析LLM响应中的抖音口播稿结果，每个脚本单独获取唯一ID
func (a *TikTokScriptAgent) ParseTikTokScriptsWithUniqueIDs(ctx context.Context, response string, request *types.AgentRequest, userID, sessionID string) ([]map[string]interface{}, error) {
	// 使用XML解析器解析内容
	config := UNIFIED_CONFIGS["tiktok_script"]
	parseResults, err := a.xmlParser.Parse(response, config, 1)
	if err != nil {
		return nil, fmt.Errorf("XML解析失败: %w", err)
	}

	// 为每个脚本单独获取会话内唯一ID
	var scripts []map[string]interface{}
	for i, result := range parseResults {
		uniqueID, err := a.GetNextActionID(ctx, userID, sessionID, "tiktok_script")
		if err != nil {
			a.Logger.Error(ctx, "获取唯一ID失败", "error", err)
			uniqueID = time.Now().UnixNano() // 使用时间戳作为备用ID
		}

		// 确保markdown格式兼容性
		title := a.EnsureMarkdownCompatibility(result.Title)
		content := a.EnsureMarkdownCompatibility(result.Content)
		coverText := a.EnsureMarkdownCompatibility(result.CoverText)
		hook := a.EnsureMarkdownCompatibility(result.Hook)

		// 记录markdown分析日志
		a.LogMarkdownAnalysis(content, fmt.Sprintf("抖音脚本%d内容", i+1))

		scripts = append(scripts, map[string]interface{}{
			"id":          fmt.Sprintf("tiktok_script%d", uniqueID),
			"title":       title,
			"content":     content,
			"cover_text":  coverText,
			"hook":        hook,
			"raw_content": result.RawContent,
			"type":        result.Type,
		})
	}

	a.Logger.Info(ctx, "为抖音脚本分配了唯一ID",
		"count", len(scripts),
		"ids", a.extractIDs(scripts))

	return scripts, nil
}

// ParseTikTokScripts 解析LLM响应中的抖音口播稿结果（保留原方法兼容性）
func (a *TikTokScriptAgent) ParseTikTokScripts(response string, request *types.AgentRequest, nextID int) []map[string]interface{} {
	// 使用统一格式XML解析器
	config := UNIFIED_CONFIGS["tiktok_script"]
	parseResults, err := a.xmlParser.Parse(response, config, nextID)
	if err != nil {
		a.Logger.Error(context.Background(), "XML解析失败", "error", err)
		return nil
	}

	// 转换为新格式并应用markdown处理
	var scripts []map[string]interface{}
	for i, result := range parseResults {
		// 确保markdown格式兼容性
		title := a.EnsureMarkdownCompatibility(result.Title)
		content := a.EnsureMarkdownCompatibility(result.Content)
		coverText := a.EnsureMarkdownCompatibility(result.CoverText)
		hook := a.EnsureMarkdownCompatibility(result.Hook)

		// 记录markdown分析日志
		a.LogMarkdownAnalysis(content, fmt.Sprintf("抖音脚本%d内容", i+1))

		scripts = append(scripts, map[string]interface{}{
			"id":          result.ID,
			"title":       title,
			"content":     content,
			"cover_text":  coverText,
			"hook":        hook,
			"raw_content": result.RawContent,
			"type":        result.Type,
		})
	}

	a.Logger.Info(context.Background(), "解析出抖音口播稿", "count", len(scripts))
	return scripts
}

// CleanTikTokTags 清理文本中的抖音标签，只保留思考过程
func (a *TikTokScriptAgent) CleanTikTokTags(text string) string {
	// 使用通用XML解析器的清理方法，支持带数字和不带数字的标签
	cleanText := a.xmlParser.CleanXMLTags(text, "tiktok_script")

	// 移除其他可能的XML标签片段
	cleanText = regexp.MustCompile(`</?title[^>]*>`).ReplaceAllString(cleanText, "")
	cleanText = regexp.MustCompile(`</?content[^>]*>`).ReplaceAllString(cleanText, "")
	cleanText = regexp.MustCompile(`</?cover_text[^>]*>`).ReplaceAllString(cleanText, "")
	cleanText = regexp.MustCompile(`</?hook[^>]*>`).ReplaceAllString(cleanText, "")
	cleanText = regexp.MustCompile(`</?text[^>]*>`).ReplaceAllString(cleanText, "")
	cleanText = regexp.MustCompile(`</?标题[^>]*>`).ReplaceAllString(cleanText, "")
	cleanText = regexp.MustCompile(`</?正文[^>]*>`).ReplaceAllString(cleanText, "")

	return cleanText
}

// ExtractOtherContent 提取标签之外的其他内容
func (a *TikTokScriptAgent) ExtractOtherContent(response string, tags []string) string {
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
func (a *TikTokScriptAgent) EnsureMarkdownCompatibility(text string) string {
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

// LogMarkdownAnalysis 记录markdown分析日志
func (a *TikTokScriptAgent) LogMarkdownAnalysis(content, label string) {
	// 这里可以添加markdown分析逻辑
	a.Logger.Debug(context.Background(), "Markdown分析", "label", label, "content_length", len(content))
}

// getStringValue 安全地获取字符串值
func (a *TikTokScriptAgent) getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// extractIDs 提取scripts中的ID列表
func (a *TikTokScriptAgent) extractIDs(scripts []map[string]interface{}) []string {
	var ids []string
	for _, script := range scripts {
		if id, ok := script["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids
}
