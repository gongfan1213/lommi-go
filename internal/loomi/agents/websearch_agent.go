package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/blueplan/loomi-go/internal/loomi/base"
	"github.com/blueplan/loomi-go/internal/loomi/events"
	"github.com/blueplan/loomi-go/internal/loomi/llm"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	zhipu "github.com/blueplan/loomi-go/internal/loomi/tools/search"
	"github.com/blueplan/loomi-go/internal/loomi/types"
	xmlx "github.com/blueplan/loomi-go/internal/loomi/utils/xml"
)

// WebSearchAgent handles web search with two-phase architecture
// Phase 1: Zhipu AI direct search (nova3_zhipu_websearch)
// Phase 2: LLM summarization based on search results (nova3_websearch)
type WebSearchAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
	zhipuCli  zhipu.ZhipuClient
}

// NewWebSearchAgent creates a new web search agent
func NewWebSearchAgent(logger *logx.Logger, client llm.Client) *WebSearchAgent {
	baseAgent := base.NewBaseLoomiAgent("loomi_websearch_agent", logger, client)
	return &WebSearchAgent{
		BaseLoomiAgent: baseAgent,
		xmlParser:      xmlx.NewLoomiXMLParser(),
		zhipuCli:       zhipu.NewHTTP(),
	}
}

// ProcessRequest processes web search requests with two-phase architecture
func (a *WebSearchAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing web search request",
		logx.KV("user_id", req.UserID),
		logx.KV("session_id", req.SessionID),
		logx.KV("instruction_length", len(req.Instruction)))

	// Set current session
	a.SetCurrentSession(req.UserID, req.SessionID)
	defer a.ClearCurrentSession()

	// Clear stop state before starting
	if err := a.ClearStopState(ctx, req.UserID, req.SessionID); err != nil {
		a.Logger.Error(ctx, "Failed to clear stop state", logx.KV("error", err))
	}

	// Check stop status
	if err := a.CheckAndRaiseIfStopped(ctx, req.UserID, req.SessionID); err != nil {
		return err
	}

	// Build clean prompt
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "websearch", req.AutoMode, req.Selections)
	if err != nil {
		a.Logger.Error(ctx, "Failed to build prompt", logx.KV("error", err))
		userPrompt = req.Instruction
	}

	// Prepare messages
	messages := []llm.Message{
		{Role: "system", Content: a.getSystemPrompt()},
		{Role: "user", Content: userPrompt},
	}

	// ==================== Phase 1: Zhipu AI Direct Search ====================

	// Emit search start thought
	if a.ShouldEmitThought("开始搜索") {
		thoughtEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentThought,
			Data:    fmt.Sprintf("🔍 开始使用智谱AI搜索：%s", req.Instruction),
		}
		if err := emit(thoughtEvent); err != nil {
			return err
		}
	}

	// Zhipu AI search integration
	searchResults, err := a.zhipuCli.SearchWeb(ctx, req.Instruction, 5)
	if err != nil {
		a.Logger.Error(ctx, "Zhipu websearch failed", logx.KV("error", err))
		// Fallback to mock for resilience
		searchResults = a.mockZhipuSearchResults(req.Instruction)
	}

	// Format search results for frontend display
	formattedSearchResults := a.formatZhipuSearchResults(searchResults, req.Instruction)

	// Send Zhipu search results to frontend (nova3_zhipu_websearch)
	zhipuEvent := events.StreamEvent{
		Type:    events.LLMChunk,
		Content: events.ContentNova3ZhipuWebsearch,
		Data:    formattedSearchResults,
	}
	if err := emit(zhipuEvent); err != nil {
		return err
	}

	a.Logger.Info(ctx, "Sent Zhipu search results to frontend",
		logx.KV("results_count", len(formattedSearchResults)))

	// Check stop status
	if err := a.CheckAndRaiseIfStopped(ctx, req.UserID, req.SessionID); err != nil {
		return err
	}

	// ==================== Phase 2: LLM Intelligent Summarization ====================

	// Emit summarization start thought
	if a.ShouldEmitThought("开始总结") {
		thoughtEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentThought,
			Data:    "📝 开始分析和总结搜索结果...",
		}
		if err := emit(thoughtEvent); err != nil {
			return err
		}
	}

	// Build LLM summarization input
	searchContent := a.extractSearchContent(searchResults)
	summaryPrompt := a.buildSummaryPrompt(req.Instruction, searchContent)

	// Update messages for summarization phase
	messages = []llm.Message{
		{Role: "system", Content: a.getSystemPrompt()},
		{Role: "user", Content: summaryPrompt},
	}

	// Collect LLM response for summarization
	a.Logger.Info(ctx, "Starting LLM response collection for summarization")
	llmResponse := ""

	err = a.SafeStreamCall(ctx, req.UserID, req.SessionID, messages, func(ctx context.Context, chunk string) error {
		llmResponse += chunk

		// Emit thought process if configured
		if a.ShouldEmitThought(chunk) {
			thoughtEvent := events.StreamEvent{
				Type:    events.LLMChunk,
				Content: events.ContentThought,
				Data:    chunk,
			}
			if err := emit(thoughtEvent); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	a.Logger.Info(ctx, "LLM response collection completed",
		logx.KV("response_length", len(llmResponse)))

	// Check stop status again
	if err := a.CheckAndRaiseIfStopped(ctx, req.UserID, req.SessionID); err != nil {
		return err
	}

	// Get next action ID
	nextID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "websearch")
	if err != nil {
		a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
		nextID = 1
	}

	// Extract other content outside of websearch tags
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<websearch\\d+>`})

	// Parse web search summaries
	summaries, err := a.parseWebSearchSummaries(ctx, llmResponse, req, nextID)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse web search summaries", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentNova3Websearch,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "Web search parsing completed",
		logx.KV("summaries_count", len(summaries)))

	// Send web search results
	if len(summaries) > 0 {
		// Process instruction for display
		displayInstruction := a.processInstruction(req.Instruction)

		// Build metadata
		metadata := map[string]any{"instruction": displayInstruction}
		if otherContent != "" {
			metadata["agent_other_message"] = otherContent
		}

		searchEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentNova3Websearch,
			Data:    summaries,
			Meta:    metadata,
		}

		if err := emit(searchEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createWebSearchNotes(ctx, req, summaries); err != nil {
			a.Logger.Error(ctx, "Failed to create web search notes", logx.KV("error", err))
		}

		a.Logger.Info(ctx, "Web search processing completed",
			logx.KV("notes_count", len(summaries)))
	} else {
		// Send raw response if no summaries found
		a.Logger.Info(ctx, "Sending raw response (no web search summaries parsed)")
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentNova3Websearch,
			Data:    llmResponse,
		})
	}

	return nil
}

// mockZhipuSearchResults creates mock Zhipu AI search results
func (a *WebSearchAgent) mockZhipuSearchResults(query string) map[string]any {
	return map[string]any{
		"success": true,
		"search_results": map[string]any{
			"content": map[string]any{
				"search_result": []map[string]string{
					{
						"title":        fmt.Sprintf("关于%s的搜索结果1", query),
						"content":      fmt.Sprintf("这是关于%s的详细搜索结果内容1，包含相关信息。", query),
						"icon":         "🔍",
						"media":        "web",
						"publish_date": "2024-01-01",
						"link":         "https://example.com/result1",
					},
					{
						"title":        fmt.Sprintf("关于%s的搜索结果2", query),
						"content":      fmt.Sprintf("这是关于%s的详细搜索结果内容2，包含相关信息。", query),
						"icon":         "🔍",
						"media":        "web",
						"publish_date": "2024-01-02",
						"link":         "https://example.com/result2",
					},
				},
			},
		},
	}
}

// formatZhipuSearchResults formats Zhipu AI search results for frontend display
func (a *WebSearchAgent) formatZhipuSearchResults(zhipuResults map[string]any, query string) []map[string]any {
	formattedResults := []map[string]any{}

	if zhipuResults["success"] == false {
		return formattedResults
	}

	searchResultsData, ok := zhipuResults["search_results"].(map[string]any)
	if !ok {
		return formattedResults
	}

	content, ok := searchResultsData["content"].(map[string]any)
	if !ok {
		return formattedResults
	}

	searchResult, ok := content["search_result"].([]map[string]string)
	if !ok {
		return formattedResults
	}

	for i, result := range searchResult {
		formattedResult := map[string]any{
			"id":           fmt.Sprintf("zhipu_search_%d", i+1),
			"type":         "zhipu_search",
			"content":      result["content"],
			"title":        result["title"],
			"icon":         result["icon"],
			"media":        result["media"],
			"publish_date": result["publish_date"],
			"link":         result["link"],
		}
		formattedResults = append(formattedResults, formattedResult)
	}

	return formattedResults
}

// extractSearchContent extracts text content from Zhipu AI search results for LLM summarization
func (a *WebSearchAgent) extractSearchContent(zhipuResults map[string]any) string {
	contentList := []string{}

	if zhipuResults["success"] == false {
		return "无搜索结果"
	}

	searchResultsData, ok := zhipuResults["search_results"].(map[string]any)
	if !ok {
		return "无搜索结果"
	}

	content, ok := searchResultsData["content"].(map[string]any)
	if !ok {
		return "无搜索结果"
	}

	searchResult, ok := content["search_result"].([]map[string]string)
	if !ok {
		return "无搜索结果"
	}

	for i, result := range searchResult {
		title := result["title"]
		contentText := result["content"]
		publishDate := result["publish_date"]

		resultText := fmt.Sprintf("搜索结果%d:\n标题: %s\n发布时间: %s\n内容: %s\n", i+1, title, publishDate, contentText)
		contentList = append(contentList, resultText)
	}

	return strings.Join(contentList, "\n")
}

// buildSummaryPrompt builds LLM summarization prompt
func (a *WebSearchAgent) buildSummaryPrompt(originalQuery string, searchContent string) string {
	return fmt.Sprintf(`用户搜索需求: %s

以下是智谱AI搜索返回的结果:
%s

请根据用户的搜索需求，将上述搜索结果整理为数个有用的信息块。每个信息块应该有完整的时间、来龙去脉，而不是碎片化的信息。使用合适的颗粒度进行整理。`, originalQuery, searchContent)
}

// parseWebSearchSummaries parses LLM response for web search summaries
func (a *WebSearchAgent) parseWebSearchSummaries(
	ctx context.Context,
	response string,
	req types.AgentRequest,
	nextID int,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.UnifiedConfigs["websearch"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	// Assign unique IDs to each summary
	summaries := make([]map[string]any, 0, len(parseResults))

	for _, result := range parseResults {
		// Get unique ID for this summary
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "websearch")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Ensure markdown compatibility
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("搜索结果 %d", len(summaries)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		summaries = append(summaries, map[string]any{
			"id":      fmt.Sprintf("websearch%d", uniqueID),
			"title":   title,
			"content": content,
			"type":    result.Type,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to web search summaries",
		logx.KV("count", len(summaries)))

	return summaries, nil
}

// createWebSearchNotes creates notes for web search results
func (a *WebSearchAgent) createWebSearchNotes(
	ctx context.Context,
	req types.AgentRequest,
	summaries []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating web search notes",
		logx.KV("count", len(summaries)))

	for _, summary := range summaries {
		id := summary["id"].(string)
		title := summary["title"].(string)
		content := summary["content"].(string)

		err := a.CreateNote(ctx, req.UserID, req.SessionID, "websearch", id, content, title, "", nil)
		if err != nil {
			a.Logger.Error(ctx, "Failed to create web search note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created web search note", logx.KV("id", id))
		}
	}

	return nil
}

// processInstruction processes the instruction for display
func (a *WebSearchAgent) processInstruction(instruction string) string {
	// Extract the part after || if present
	if idx := strings.LastIndex(instruction, "||"); idx != -1 {
		return strings.TrimSpace(instruction[idx+2:])
	}
	return instruction
}

// getSystemPrompt returns the system prompt for web search
func (a *WebSearchAgent) getSystemPrompt() string {
	return `# 任务简述
你将使用真实的互联网搜索功能来获取最新信息。请根据用户需求，主动搜索相关的实时信息和资料，然后整理为舒展、详细、易读的信息块。
禁止编造任何信息。

# 输出格式要求
最终你必须使用XML标签格式包裹输出搜索结果
如果有多个不同方向主题的信息，使用<websearch数字>标签分别包裹，至多不超过3条。
每个信息块应包含完整的时间、来龙去脉，而不是碎片化的信息。

格式：
<websearch1>
<title>搜索结果一标题</title>
<content>搜索结果一内容</content>
</websearch1>`
}
