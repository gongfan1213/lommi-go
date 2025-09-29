package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blueplan/loomi-go/internal/loomi/base"
	"github.com/blueplan/loomi-go/internal/loomi/events"
	"github.com/blueplan/loomi-go/internal/loomi/llm"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	mmp "github.com/blueplan/loomi-go/internal/loomi/tools/multimodal"
	"github.com/blueplan/loomi-go/internal/loomi/types"
)

// LoomiConcierge handles user interaction, context management, and task delegation
type LoomiConcierge struct {
	*base.BaseLoomiAgent

	// Configuration
	autoMode       bool
	userSelections []string

	// Execution state
	executedActions map[string]bool
	totalLLMCalls   int
	totalActions    int

	// Optional multimodal processor (Noop by default)
	mmProcessor mmp.Processor
}

// NewLoomiConcierge creates a new concierge agent
func NewLoomiConcierge(logger *logx.Logger, client llm.Client) *LoomiConcierge {
	baseAgent := base.NewBaseLoomiAgent("loomi_concierge", logger, client)

	concierge := &LoomiConcierge{
		BaseLoomiAgent: baseAgent,

		// Initialize execution state
		executedActions: make(map[string]bool),
		mmProcessor:     &mmp.Noop{},
	}

	concierge.Logger.Info(context.Background(), "LoomiConcierge initialized",
		logx.KV("agent_name", "loomi_concierge"),
	)

	return concierge
}

// ProcessRequest processes concierge workflow requests
func (a *LoomiConcierge) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	// Emit pre-plan event (loomi_plan_concierge) similar to Python
	if err := emit(events.StreamEvent{
		Type:    events.LLMChunk,
		Content: events.ContentLoomiPlanConcierge,
		Data: []map[string]any{{
			"action_type": "concierge",
			"instruction": req.Instruction,
			"user_id":     req.UserID,
			"session_id":  req.SessionID,
			"status":      "starting",
			"message":     "即将开始接待员任务...",
		}},
		Meta: map[string]any{"action_type": "concierge", "plan_type": "loomi_plan"},
	}); err != nil {
		a.Logger.Error(ctx, "Failed to emit pre-plan event", logx.KV("error", err))
	}
	a.Logger.Info(ctx, "Processing concierge workflow request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "concierge", req.AutoMode, req.Selections)
	if err != nil {
		a.Logger.Error(ctx, "Failed to build prompt", logx.KV("error", err))
		userPrompt = req.Instruction
	}

	// Set execution mode and user selections
	a.autoMode = req.AutoMode
	a.userSelections = req.Selections
	a.totalLLMCalls = 0
	a.totalActions = 0

	// Before LLM call: optional multimodal processing
	// 1) file_ids 模式
	if len(req.FileIDs) > 0 && a.mmProcessor != nil && a.mmProcessor.Enabled() {
		a.Logger.Info(ctx, "Detected file_ids for multimodal processing", logx.KV("count", len(req.FileIDs)))
		// Provide file IDs to processor if supported
		a.mmProcessor.SetFileIDs(req.FileIDs)
		// 这里保留与 Python 逻辑一致的用户提示处理：文件分析描述合并进上下文
		if results, err := a.mmProcessor.Process(ctx, req.UserID, req.SessionID, req.Instruction); err == nil && len(results) > 0 {
			if errSave := a.mmProcessor.SaveAsNotes(ctx, req.UserID, req.SessionID, a.AgentName, results); errSave != nil {
				a.Logger.Error(ctx, "Save multimodal notes failed", logx.KV("error", errSave))
			} else {
				a.Logger.Info(ctx, "Saved multimodal notes", logx.KV("count", len(results)))
			}
			// 将简要提示加入 userPrompt（不改变原逻辑结构，只补充上下文）
			userPrompt = userPrompt + "\n\n[系统已完成文件多模态分析，相关知识点已保存为 Notes，可在后续分析中引用。]"
		} else if err != nil {
			a.Logger.Error(ctx, "Multimodal processing (file_ids) failed", logx.KV("error", err))
		}
	} else if a.mmProcessor != nil && a.mmProcessor.Enabled() {
		// 2) 文本引用触发的多模态
		if shouldUse, err := a.mmProcessor.ShouldUse(ctx, req.UserID, req.SessionID, req.Instruction); err == nil && shouldUse {
			a.Logger.Info(ctx, "Detected text-based multimodal references, starting processing")
			if results, err := a.mmProcessor.Process(ctx, req.UserID, req.SessionID, req.Instruction); err == nil && len(results) > 0 {
				if errSave := a.mmProcessor.SaveAsNotes(ctx, req.UserID, req.SessionID, a.AgentName, results); errSave != nil {
					a.Logger.Error(ctx, "Save multimodal notes failed", logx.KV("error", errSave))
				} else {
					a.Logger.Info(ctx, "Saved multimodal notes", logx.KV("count", len(results)))
				}
				userPrompt = userPrompt + "\n\n[系统已完成文件多模态分析，相关知识点已保存为 Notes，可在后续分析中引用。]"
			} else if err != nil {
				a.Logger.Error(ctx, "Multimodal processing (text) failed", logx.KV("error", err))
			}
		}
	}

	// Prepare messages
	messages := []llm.Message{
		{Role: "system", Content: a.getSystemPrompt()},
		{Role: "user", Content: userPrompt},
	}

	// Collect LLM response
	a.Logger.Info(ctx, "Starting LLM response collection")
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

	// Process XML tags in concierge response
	processedText, orchestratorCalls, webSearchCalls := a.processXMLTags(ctx, llmResponse, req.UserID, req.SessionID)
	a.Logger.Info(ctx, "XML tag processing completed",
		logx.KV("orchestrator_calls", len(orchestratorCalls)),
		logx.KV("web_search_calls", len(webSearchCalls)))

	// Parse concierge results with unique IDs
	conciergeResults, err := a.parseConciergeResultsWithUniqueIDs(ctx, processedText, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse concierge results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentConciergeMessage,
			Data:    processedText,
		})
	}

	a.Logger.Info(ctx, "Concierge parsing completed",
		logx.KV("results_count", len(conciergeResults)))

	// Send concierge results
	if len(conciergeResults) > 0 {
		// Build metadata
		metadata := map[string]any{"instruction": req.Instruction, "raw_response": llmResponse}

		conciergeEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentConciergeMessage,
			Data:    conciergeResults,
			Meta:    metadata,
		}

		if err := emit(conciergeEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createConciergeNotes(ctx, req, conciergeResults); err != nil {
			a.Logger.Error(ctx, "Failed to create concierge notes", logx.KV("error", err))
		}

		a.Logger.Info(ctx, "Concierge processing completed",
			logx.KV("notes_count", len(conciergeResults)))
	} else {
		// Send raw response if no results found
		a.Logger.Info(ctx, "Sending raw response (no concierge results parsed)")
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentConciergeMessage,
			Data:    processedText,
		})
	}

	// Handle orchestrator calls
	if len(orchestratorCalls) > 0 {
		a.Logger.Info(ctx, "Processing orchestrator calls",
			logx.KV("count", len(orchestratorCalls)))

		for _, call := range orchestratorCalls {
			if err := a.triggerOrchestrator(ctx, req.UserID, req.SessionID, call, emit); err != nil {
				a.Logger.Error(ctx, "Failed to trigger orchestrator",
					logx.KV("instruction", call),
					logx.KV("error", err))
			}
		}
	}

	// Handle web search calls
	if len(webSearchCalls) > 0 {
		a.Logger.Info(ctx, "Processing web search calls",
			logx.KV("count", len(webSearchCalls)))

		for _, search := range webSearchCalls {
			if err := a.triggerWebSearch(ctx, req.UserID, req.SessionID, search, emit); err != nil {
				a.Logger.Error(ctx, "Failed to trigger web search",
					logx.KV("keyword", search),
					logx.KV("error", err))
			}
		}

		// After web search finishes, re-analyze based on updated context (parity with Python concierge)
		a.Logger.Info(ctx, "Re-analyzing after web search results")
		// Rebuild prompt to include latest context (search results were added to history by WebSearchAgent)
		reAnalysisPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, "基于上述搜索结果，请分析用户的需求并提供建议", "concierge", req.AutoMode, req.Selections)
		if err != nil {
			a.Logger.Error(ctx, "Failed to build re-analysis prompt", logx.KV("error", err))
			reAnalysisPrompt = "基于上述搜索结果，请分析用户的需求并提供建议"
		}

		messages2 := []llm.Message{
			{Role: "system", Content: a.getSystemPrompt()},
			{Role: "user", Content: reAnalysisPrompt},
		}

		reResponse := ""
		err = a.SafeStreamCall(ctx, req.UserID, req.SessionID, messages2, func(ctx context.Context, chunk string) error {
			reResponse += chunk
			if a.ShouldEmitThought(chunk) {
				thoughtEvent := events.StreamEvent{Type: events.LLMChunk, Content: events.ContentThought, Data: chunk}
				if err := emit(thoughtEvent); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Parse and emit re-analysis results (no further websearch/orchestrator triggers to avoid loops)
		reConciergeResults, err := a.parseConciergeResultsWithUniqueIDs(ctx, reResponse, req)
		if err != nil || len(reConciergeResults) == 0 {
			_ = emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentConciergeMessage, Data: reResponse, Meta: map[string]any{"re_analysis": true, "based_on_search": true}})
		} else {
			ev := events.StreamEvent{Type: events.LLMChunk, Content: events.ContentConciergeMessage, Data: reConciergeResults, Meta: map[string]any{"re_analysis": true, "based_on_search": true}}
			if err := emit(ev); err != nil {
				return err
			}
			// Also create notes for re-analysis items
			if err := a.createConciergeNotes(ctx, req, reConciergeResults); err != nil {
				a.Logger.Error(ctx, "Failed to create notes for re-analysis", logx.KV("error", err))
			}
		}
	}

	return nil
}

// processXMLTags processes XML tags in concierge response
func (a *LoomiConcierge) processXMLTags(
	ctx context.Context,
	response string,
	userID string,
	sessionID string,
) (string, []string, []string) {
	processedText := response
	orchestratorCalls := []string{}
	webSearchCalls := []string{}

	// Process create_note tags
	processedText = a.processCreateNoteTags(ctx, processedText, userID, sessionID)

	// Process save_material tags
	processedText = a.processSaveMaterialTags(ctx, processedText, userID, sessionID)

	// Process call_orchestrator tags
	processedText, orchestratorCalls = a.processCallOrchestratorTags(ctx, processedText)

	// Process web_search tags
	processedText, webSearchCalls = a.processWebSearchTags(ctx, processedText)

	return processedText, orchestratorCalls, webSearchCalls
}

// processCreateNoteTags processes create_note XML tags
func (a *LoomiConcierge) processCreateNoteTags(
	ctx context.Context,
	text string,
	userID string,
	sessionID string,
) string {
	pattern := regexp.MustCompile(`<create_note>\s*<type>([^<]+)</type>\s*<id>([^<]+)</id>\s*<content>(.*?)</content>\s*</create_note>`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	processedText := text
	for _, match := range matches {
		if len(match) >= 4 {
			noteType := strings.TrimSpace(match[1])
			noteID := strings.TrimSpace(match[2])
			content := strings.TrimSpace(match[3])

			// Create the note
			err := a.CreateNote(ctx, userID, sessionID, noteType, noteID, content, "", "", nil)

			// Replace XML tag with confirmation message
			fullPattern := fmt.Sprintf(`<create_note>\s*<type>%s</type>\s*<id>%s</id>\s*<content>.*?</content>\s*</create_note>`,
				regexp.QuoteMeta(noteType), regexp.QuoteMeta(noteID))

			replacement := ""
			if err == nil {
				replacement = fmt.Sprintf("📝 已保存%s: %s", noteType, noteID)
			} else {
				replacement = fmt.Sprintf("❌ 保存%s失败: %s", noteType, noteID)
			}

			processedText = regexp.MustCompile(fullPattern).ReplaceAllString(processedText, replacement)
		}
	}

	return processedText
}

// processSaveMaterialTags processes save_material XML tags
func (a *LoomiConcierge) processSaveMaterialTags(
	ctx context.Context,
	text string,
	userID string,
	sessionID string,
) string {
	pattern := regexp.MustCompile(`<save_material>\s*<id>([^<]+)</id>\s*<content>(.*?)</content>\s*</save_material>`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	processedText := text
	for _, match := range matches {
		if len(match) >= 3 {
			materialID := strings.TrimSpace(match[1])
			content := strings.TrimSpace(match[2])

			// Create the material note
			noteName := fmt.Sprintf("material%s", materialID)
			err := a.CreateNote(ctx, userID, sessionID, "material", noteName, content, "", "", nil)

			// Replace XML tag with confirmation message
			fullPattern := fmt.Sprintf(`<save_material>\s*<id>%s</id>\s*<content>.*?</content>\s*</save_material>`,
				regexp.QuoteMeta(materialID))

			replacement := ""
			if err == nil {
				replacement = fmt.Sprintf("📝 已保存研究进展: %s", noteName)
			} else {
				replacement = fmt.Sprintf("❌ 保存研究进展失败: %s", noteName)
			}

			processedText = regexp.MustCompile(fullPattern).ReplaceAllString(processedText, replacement)
		}
	}

	return processedText
}

// processCallOrchestratorTags processes call_orchestrator XML tags
func (a *LoomiConcierge) processCallOrchestratorTags(
	ctx context.Context,
	text string,
) (string, []string) {
	pattern := regexp.MustCompile(`<call_orchestrator[^>]*>\s*(.*?)\s*</call_orchestrator>`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	processedText := text
	orchestratorCalls := []string{}

	for _, match := range matches {
		if len(match) >= 2 {
			instruction := strings.TrimSpace(match[1])
			orchestratorCalls = append(orchestratorCalls, instruction)

			// Remove XML tag from text
			fullPattern := fmt.Sprintf(`<call_orchestrator[^>]*>\s*%s\s*</call_orchestrator>`,
				regexp.QuoteMeta(match[1]))
			processedText = regexp.MustCompile(fullPattern).ReplaceAllString(processedText, "")
		}
	}

	return processedText, orchestratorCalls
}

// processWebSearchTags processes web_search XML tags
func (a *LoomiConcierge) processWebSearchTags(
	ctx context.Context,
	text string,
) (string, []string) {
	pattern := regexp.MustCompile(`<web_search(\d+)>\s*(.*?)\s*</web_search\1>`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	processedText := text
	webSearchCalls := []string{}

	for _, match := range matches {
		if len(match) >= 3 {
			keyword := strings.TrimSpace(match[2])
			webSearchCalls = append(webSearchCalls, keyword)

			// Replace XML tag with search indication
			fullPattern := fmt.Sprintf(`<web_search%s>\s*%s\s*</web_search%s>`,
				match[1], regexp.QuoteMeta(match[2]), match[1])
			replacement := fmt.Sprintf("🔍 即将搜索: %s", keyword)
			processedText = regexp.MustCompile(fullPattern).ReplaceAllString(processedText, replacement)
		}
	}

	return processedText, webSearchCalls
}

// parseConciergeResultsWithUniqueIDs parses LLM response and assigns unique IDs
func (a *LoomiConcierge) parseConciergeResultsWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	results := []map[string]any{}

	// Get unique ID for this concierge result
	uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "concierge")
	if err != nil {
		a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
		return results, err
	}

	// Parse confirm tags and regular text
	confirmPattern := regexp.MustCompile(`<confirm(\d+)>(.*?)</confirm\1>`)
	confirmMatches := confirmPattern.FindAllStringSubmatch(response, -1)

	currentPos := 0
	currentID := uniqueID

	// Process confirm tags
	for _, match := range confirmMatches {
		if len(match) >= 3 {
			// Add text before confirm tag
			startPos := strings.Index(response, match[0])
			if startPos > currentPos {
				textBefore := strings.TrimSpace(response[currentPos:startPos])
				if textBefore != "" {
					results = append(results, map[string]any{
						"id":      fmt.Sprintf("concierge%d", currentID),
						"content": textBefore,
						"type":    "message",
					})
					currentID++
				}
			}

			// Add confirm content
			confirmContent := strings.TrimSpace(match[2])
			if confirmContent != "" {
				results = append(results, map[string]any{
					"id":      fmt.Sprintf("concierge%d", currentID),
					"content": confirmContent,
					"type":    "confirm",
				})
				currentID++
			}

			currentPos = startPos + len(match[0])
		}
	}

	// Add remaining text
	if currentPos < len(response) {
		textAfter := strings.TrimSpace(response[currentPos:])
		if textAfter != "" {
			results = append(results, map[string]any{
				"id":      fmt.Sprintf("concierge%d", currentID),
				"content": textAfter,
				"type":    "message",
			})
		}
	}

	// If no confirm tags found, treat entire response as message
	if len(results) == 0 && response != "" {
		results = append(results, map[string]any{
			"id":      fmt.Sprintf("concierge%d", currentID),
			"content": strings.TrimSpace(response),
			"type":    "message",
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to concierge results",
		logx.KV("count", len(results)))

	return results, nil
}

// createConciergeNotes creates notes for concierge analysis results
func (a *LoomiConcierge) createConciergeNotes(
	ctx context.Context,
	req types.AgentRequest,
	results []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating concierge notes",
		logx.KV("count", len(results)))

	for _, result := range results {
		id := result["id"].(string)
		content := result["content"].(string)
		resultType := result["type"].(string)

		title := fmt.Sprintf("接待员分析 %s", id)
		if resultType == "confirm" {
			title = fmt.Sprintf("确认信息 %s", id)
		}

		err := a.CreateNote(ctx, req.UserID, req.SessionID, "concierge", id, content, title, "", nil)
		if err != nil {
			a.Logger.Error(ctx, "Failed to create concierge note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created concierge note", logx.KV("id", id))
		}
	}

	return nil
}

// triggerOrchestrator triggers orchestrator execution
func (a *LoomiConcierge) triggerOrchestrator(
	ctx context.Context,
	userID string,
	sessionID string,
	instruction string,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Triggering orchestrator",
		logx.KV("user_id", userID),
		logx.KV("session_id", sessionID),
		logx.KV("instruction", instruction))

	// Create orchestrator request
	orchestratorReq := types.AgentRequest{
		UserID:      userID,
		SessionID:   sessionID,
		Instruction: instruction,
		AutoMode:    a.autoMode,
		Selections:  a.userSelections,
	}

	// Create orchestrator instance
	orchestrator := NewLoomiOrchestrator(a.Logger, a.LLMClient)

	// Process orchestrator request
	return orchestrator.ProcessRequest(ctx, orchestratorReq, emit)
}

// triggerWebSearch triggers web search execution
func (a *LoomiConcierge) triggerWebSearch(
	ctx context.Context,
	userID string,
	sessionID string,
	keyword string,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Triggering web search",
		logx.KV("user_id", userID),
		logx.KV("session_id", sessionID),
		logx.KV("keyword", keyword))

	// Forward to WebSearchAgent to perform actual search and emit concierge-specific events upstream
	web := NewWebSearchAgent(a.Logger, a.LLMClient)
	// Wrap emit to remap nova3 websearch event types into concierge-specific ones for frontend
	remapEmit := func(ev events.StreamEvent) error {
		switch ev.Content {
		case events.ContentNova3ZhipuWebsearch, events.ContentNova3Websearch, events.ContentLoomiWebSearch:
			ev.Content = events.ContentConciergeWebsearch
		}
		return emit(ev)
	}
	return web.ProcessRequest(ctx, types.AgentRequest{
		UserID: userID, SessionID: sessionID, Instruction: keyword, AutoMode: a.autoMode, Selections: a.userSelections,
	}, remapEmit)
}

// getSystemPrompt returns the system prompt for concierge analysis
func (a *LoomiConcierge) getSystemPrompt() string {
	return `你是一个智能接待员专家。请基于用户的需求和上下文，提供友好的交互体验，管理用户知识库，并协调专业agent的工作。

请按照以下格式输出分析结果：

<confirm1>
确认信息或需要用户确认的内容
</confirm1>

<confirm2>
另一个确认角度
</confirm2>

普通文本内容将作为消息直接显示给用户。

请根据需要创建笔记、保存研究材料、调用编排器或进行网络搜索：

<create_note>
<type>笔记类型</type>
<id>笔记标识</id>
<content>笔记内容</content>
</create_note>

<save_material>
<id>材料标识</id>
<content>研究材料内容</content>
</save_material>

<call_orchestrator>编排器调用指令</call_orchestrator>

<web_search1>搜索关键词</web_search1>

请根据用户需求提供1-3个不同的确认选项或建议，每个确认都要有明确的内容。`
}

// Helper methods for concierge functionality

// GetExecutionStats returns current execution statistics
func (a *LoomiConcierge) GetExecutionStats() map[string]int {
	return map[string]int{
		"total_llm_calls": a.totalLLMCalls,
		"total_actions":   a.totalActions,
	}
}

// SetAutoMode sets the auto mode configuration
func (a *LoomiConcierge) SetAutoMode(autoMode bool) {
	a.autoMode = autoMode
	a.Logger.Info(context.Background(), "Updated auto mode configuration",
		logx.KV("auto_mode", autoMode))
}

// SetUserSelections sets the user selections configuration
func (a *LoomiConcierge) SetUserSelections(selections []string) {
	a.userSelections = selections
	a.Logger.Info(context.Background(), "Updated user selections configuration",
		logx.KV("selections", selections))
}
