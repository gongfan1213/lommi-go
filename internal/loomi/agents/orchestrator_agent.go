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
	"github.com/blueplan/loomi-go/internal/loomi/types"
	xmlx "github.com/blueplan/loomi-go/internal/loomi/utils/xml"
)

// LoomiOrchestrator handles workflow orchestration with ReAct patterns
type LoomiOrchestrator struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser

	// Configuration
	maxIterations        int
	mergeableActions     map[string]bool
	maxConcurrentAgents  int
	outputInterval       float64
	noIntervalAgentTypes map[string]bool

	// Execution state
	autoMode        bool
	userSelections  []string
	executedActions map[string]bool
	totalLLMCalls   int
	totalActions    int
}

// NewLoomiOrchestrator creates a new orchestrator agent
func NewLoomiOrchestrator(logger *logx.Logger, client llm.Client) *LoomiOrchestrator {
	baseAgent := base.NewBaseLoomiAgent("loomi_orchestrator", logger, client)

	orchestrator := &LoomiOrchestrator{
		BaseLoomiAgent:      baseAgent,
		xmlParser:           xmlx.NewLoomiXMLParser(),
		maxIterations:       4,
		maxConcurrentAgents: 8,
		outputInterval:      10.0,

		// Initialize mergeable actions
		mergeableActions: map[string]bool{
			"hitpoint":       true,
			"xhs_post":       true,
			"wechat_article": true,
			"tiktok_script":  true,
		},

		// Initialize no-interval agent types
		noIntervalAgentTypes: map[string]bool{
			"hitpoint":       true,
			"tiktok_script":  true,
			"xhs_post":       true,
			"wechat_article": true,
			"revision":       true,
		},

		// Initialize execution state
		executedActions: make(map[string]bool),
	}

	orchestrator.Logger.Info(context.Background(), "LoomiOrchestrator initialized",
		logx.KV("max_iterations", orchestrator.maxIterations),
		logx.KV("max_concurrent_agents", orchestrator.maxConcurrentAgents),
		logx.KV("mergeable_actions", orchestrator.getMergeableActionsList()),
	)

	return orchestrator
}

// ProcessRequest processes orchestrator workflow requests
func (a *LoomiOrchestrator) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing orchestrator workflow request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "orchestrator", req.AutoMode, req.Selections)
	if err != nil {
		a.Logger.Error(ctx, "Failed to build prompt", logx.KV("error", err))
		userPrompt = req.Instruction
	}

	// Set execution mode and user selections
	a.autoMode = req.AutoMode
	a.userSelections = req.Selections
	a.totalLLMCalls = 0
	a.totalActions = 0

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

	// Parse orchestrator results with unique IDs
	orchestratorResults, err := a.parseOrchestratorResultsWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse orchestrator results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiOrchestrator,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "Orchestrator parsing completed",
		logx.KV("results_count", len(orchestratorResults)))

	// Send orchestrator results
	if len(orchestratorResults) > 0 {
		// Build metadata
		metadata := map[string]any{"instruction": req.Instruction}

		orchestratorEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiOrchestrator,
			Data:    orchestratorResults,
			Meta:    metadata,
		}

		if err := emit(orchestratorEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createOrchestratorNotes(ctx, req, orchestratorResults); err != nil {
			a.Logger.Error(ctx, "Failed to create orchestrator notes", logx.KV("error", err))
		}

		a.Logger.Info(ctx, "Orchestrator processing completed",
			logx.KV("notes_count", len(orchestratorResults)))
	} else {
		// Send raw response if no results found
		a.Logger.Info(ctx, "Sending raw response (no orchestrator results parsed)")
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiOrchestrator,
			Data:    llmResponse,
		})
	}

	// Execute actions requested by orchestrator via <execute action="..." instruction="..." /> tags
	executeCalls := a.ProcessExecuteTags(llmResponse)
	if len(executeCalls) > 0 {
		a.Logger.Info(ctx, "Detected execute actions from orchestrator",
			logx.KV("count", len(executeCalls)))
		for _, call := range executeCalls {
			action := strings.TrimSpace(call["action"])
			instruction := strings.TrimSpace(call["instruction"])
			if action == "" || instruction == "" {
				continue
			}
			// Build downstream request
			subReq := types.AgentRequest{
				UserID:      req.UserID,
				SessionID:   req.SessionID,
				Instruction: instruction,
				AutoMode:    a.autoMode,
				Selections:  a.userSelections,
			}
			if err := a.executeAction(ctx, action, subReq, emit); err != nil {
				a.Logger.Error(ctx, "Execute action failed",
					logx.KV("action", action),
					logx.KV("error", err))
			}
		}
	}

	return nil
}

// executeAction maps action type to a concrete agent and runs it (logic mirrors Python _create_agent_by_type)
func (a *LoomiOrchestrator) executeAction(
	ctx context.Context,
	action string,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	switch action {
	case "xhs_post":
		ag := NewXHSPostAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "wechat_article":
		ag := NewWeChatArticleAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "hitpoint":
		ag := NewHitpointAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "persona":
		ag := NewPersonaAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "websearch":
		ag := NewWebSearchAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "tiktok_script":
		ag := NewTikTokScriptAgent(a.Logger, a.LLMClient)
		return ag.Process(ctx, req, emit)
	case "brand_analysis":
		ag := NewBrandAnalysisAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "content_analysis":
		ag := NewContentAnalysisAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "knowledge":
		ag := NewKnowledgeAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "resonant":
		ag := NewResonantAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	case "revision":
		ag := NewRevisionAgent(a.Logger, a.LLMClient)
		return ag.ProcessRequest(ctx, req, emit)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// parseOrchestratorResultsWithUniqueIDs parses LLM response and assigns unique IDs
func (a *LoomiOrchestrator) parseOrchestratorResultsWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.UnifiedConfigs["orchestrator"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	// Assign unique IDs to each orchestrator result
	results := make([]map[string]any, 0, len(parseResults))

	for _, result := range parseResults {
		// Get unique ID for this orchestrator result
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "orchestrator")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Ensure markdown compatibility
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("编排分析 %d", len(results)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		results = append(results, map[string]any{
			"id":      fmt.Sprintf("orchestrator%d", uniqueID),
			"title":   title,
			"content": content,
			"type":    result.Type,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to orchestrator results",
		logx.KV("count", len(results)))

	return results, nil
}

// createOrchestratorNotes creates notes for orchestrator analysis results
func (a *LoomiOrchestrator) createOrchestratorNotes(
	ctx context.Context,
	req types.AgentRequest,
	results []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating orchestrator notes",
		logx.KV("count", len(results)))

	for _, result := range results {
		id := result["id"].(string)
		title := result["title"].(string)
		content := result["content"].(string)

		err := a.CreateNote(ctx, req.UserID, req.SessionID, "orchestrator", id, content, title, "", nil)
		if err != nil {
			a.Logger.Error(ctx, "Failed to create orchestrator note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created orchestrator note", logx.KV("id", id))
		}
	}

	return nil
}

// getSystemPrompt returns the system prompt for orchestrator analysis
func (a *LoomiOrchestrator) getSystemPrompt() string {
	return `你是一个智能编排专家。请基于用户的需求和上下文，制定合理的执行计划并协调各个专业agent的工作。

请按照以下格式输出分析结果：

<orchestrator1>
<title>编排计划标题</title>
<content>
详细的编排计划内容，包括：
- 需求分析：理解用户的核心需求和目标
- 执行策略：制定合理的执行顺序和并发策略
- 资源分配：确定需要调用的专业agent类型
- 预期结果：描述执行完成后的预期成果
- 风险控制：识别潜在问题并提供应对方案
</content>
</orchestrator1>

<orchestrator2>
<title>另一个编排角度</title>
<content>
从不同角度制定另一个编排计划...
</content>
</orchestrator2>

请提供1-2个不同的编排计划，每个计划都要有明确的标题和详细的内容。`
}

// Helper methods for orchestrator functionality

func (a *LoomiOrchestrator) getMergeableActionsList() []string {
	actions := make([]string, 0, len(a.mergeableActions))
	for action := range a.mergeableActions {
		actions = append(actions, action)
	}
	return actions
}

// SetMergeableActions sets the mergeable actions configuration
func (a *LoomiOrchestrator) SetMergeableActions(actions []string) {
	a.mergeableActions = make(map[string]bool)
	for _, action := range actions {
		a.mergeableActions[action] = true
	}
	a.Logger.Info(context.Background(), "Updated mergeable actions configuration",
		logx.KV("actions", actions))
}

// AddMergeableAction adds an action type to mergeable actions
func (a *LoomiOrchestrator) AddMergeableAction(actionType string) {
	a.mergeableActions[actionType] = true
	a.Logger.Info(context.Background(), "Added mergeable action type",
		logx.KV("action_type", actionType))
}

// RemoveMergeableAction removes an action type from mergeable actions
func (a *LoomiOrchestrator) RemoveMergeableAction(actionType string) {
	delete(a.mergeableActions, actionType)
	a.Logger.Info(context.Background(), "Removed mergeable action type",
		logx.KV("action_type", actionType))
}

// ProcessExecuteTags processes execute tags in orchestrator response
func (a *LoomiOrchestrator) ProcessExecuteTags(response string) []map[string]string {
	// Extract execute tags from response
	executePattern := regexp.MustCompile(`<execute\s+action="([^"]+)"\s+instruction="([^"]+)"\s*/>`)
	matches := executePattern.FindAllStringSubmatch(response, -1)

	executeActions := make([]map[string]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 3 {
			action := strings.TrimSpace(match[1])
			instruction := strings.TrimSpace(match[2])

			executeActions = append(executeActions, map[string]string{
				"action":      action,
				"instruction": instruction,
			})
		}
	}

	return executeActions
}

// ShouldBreakAfterRound determines if execution should break after current round
func (a *LoomiOrchestrator) ShouldBreakAfterRound(executedActions []string, roundNumber int) bool {
	// Define break actions
	breakActions := map[string]bool{
		"hitpoint":       true,
		"xhs_post":       true,
		"wechat_article": true,
		"tiktok_script":  true,
	}

	// Check if any executed action is a break action
	for _, action := range executedActions {
		if breakActions[action] {
			a.Logger.Info(context.Background(), "Detected break action, should break after round",
				logx.KV("round", roundNumber),
				logx.KV("action", action))
			return true
		}
	}

	return false
}

// ShouldFinish checks if orchestrator should finish execution
func (a *LoomiOrchestrator) ShouldFinish(response string) bool {
	// Check for ORCHESTRATOR_DECLARATION or other finish patterns
	finishPatterns := []string{
		`<ORCHESTRATOR_DECLARATION`,
		`<finish>`,
		`<complete>`,
		`<done>`,
	}

	for _, pattern := range finishPatterns {
		if strings.Contains(strings.ToUpper(response), strings.ToUpper(pattern)) {
			a.Logger.Info(context.Background(), "Detected finish pattern",
				logx.KV("pattern", pattern))
			return true
		}
	}

	return false
}

// GetExecutionStats returns current execution statistics
func (a *LoomiOrchestrator) GetExecutionStats() map[string]int {
	return map[string]int{
		"total_llm_calls": a.totalLLMCalls,
		"total_actions":   a.totalActions,
	}
}
