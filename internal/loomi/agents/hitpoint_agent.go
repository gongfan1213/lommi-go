package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/blueplan/loomi-go/internal/loomi/base"
	"github.com/blueplan/loomi-go/internal/loomi/events"
	"github.com/blueplan/loomi-go/internal/loomi/llm"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/types"
	xmlx "github.com/blueplan/loomi-go/internal/loomi/utils/xml"
)

// HitpointAgent handles content targeting analysis
type HitpointAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

// NewHitpointAgent creates a new hitpoint agent
func NewHitpointAgent(logger *logx.Logger, client llm.Client) *HitpointAgent {
	baseAgent := base.NewBaseLoomiAgent("loomi_hitpoint_agent", logger, client)
	return &HitpointAgent{
		BaseLoomiAgent: baseAgent,
		xmlParser:      xmlx.NewLoomiXMLParser(),
	}
}

// ProcessRequest processes hitpoint analysis requests
func (a *HitpointAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing hitpoint analysis request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "hitpoint", req.AutoMode, req.Selections)
	if err != nil {
		a.Logger.Error(ctx, "Failed to build prompt", logx.KV("error", err))
		userPrompt = req.Instruction
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

	// Parse hitpoint results with unique IDs
	hitpoints, err := a.parseHitpointsWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse hitpoint results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiHitpoint,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "Hitpoint parsing completed",
		logx.KV("hitpoints_count", len(hitpoints)))

	// Send hitpoint results
	if len(hitpoints) > 0 {
		// Process instruction for display
		displayInstruction := a.processInstruction(req.Instruction)

		hitpointEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiHitpoint,
			Data:    hitpoints,
			Meta:    map[string]any{"instruction": displayInstruction},
		}

		if err := emit(hitpointEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createHitpointNotes(ctx, req, hitpoints); err != nil {
			a.Logger.Error(ctx, "Failed to create hitpoint notes", logx.KV("error", err))
		}

		a.Logger.Info(ctx, "Hitpoint processing completed",
			logx.KV("notes_count", len(hitpoints)))
	} else {
		// Send raw response if no hitpoints found
		a.Logger.Info(ctx, "Sending raw response (no hitpoints parsed)")
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiHitpoint,
			Data:    llmResponse,
		})
	}

	return nil
}

// parseHitpointsWithUniqueIDs parses LLM response and assigns unique IDs
func (a *HitpointAgent) parseHitpointsWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.UnifiedConfigs["hitpoint"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	// Assign unique IDs to each hitpoint
	hitpoints := make([]map[string]any, 0, len(parseResults))

	for _, result := range parseResults {
		// Get unique ID for this hitpoint
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "hitpoint")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Ensure markdown compatibility
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("打点分析 %d", len(hitpoints)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		hitpoints = append(hitpoints, map[string]any{
			"id":      fmt.Sprintf("hitpoint%d", uniqueID),
			"title":   title,
			"content": content,
			"type":    result.Type,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to hitpoints",
		logx.KV("count", len(hitpoints)))

	return hitpoints, nil
}

// createHitpointNotes creates notes for hitpoint analysis results
func (a *HitpointAgent) createHitpointNotes(
	ctx context.Context,
	req types.AgentRequest,
	hitpoints []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating hitpoint notes",
		logx.KV("count", len(hitpoints)))

	for _, hitpoint := range hitpoints {
		id := hitpoint["id"].(string)
		title := hitpoint["title"].(string)
		content := hitpoint["content"].(string)

		err := a.CreateNote(ctx, req.UserID, req.SessionID, "hitpoint", id, content, title, "", nil)
		if err != nil {
			a.Logger.Error(ctx, "Failed to create hitpoint note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created hitpoint note", logx.KV("id", id))
		}
	}

	return nil
}

// getSystemPrompt returns the system prompt for hitpoint analysis
func (a *HitpointAgent) getSystemPrompt() string {
	return `你的任务是思考【写作任务背景】怎么写效果最优，最终得出至多3条能爆的"内容策略"。
对于[参考资料]，你只是合理参考，保持独立思考。

# 什么是内容策略？
**"内容策略"是对一篇内容从什么出发、往什么方向写、到底要说一个什么事儿、达到什么效果的核心指导。**
**你需要先根据任务需求，确认内容策略的侧重，比如用户要写推广软文，就要重点提供说服力、种草力来真正推动消费决策和品牌认知；如果用户是要写一篇娱乐内容来起号，可能着重提供吸睛的叙事张力，吸引点击和停留，等等**

要求：
- 内容策略必须真实立体：要符合简中现状，考虑现实场景，不能显得假；
- 不要试图“类比”、“包装”、“定义”，你的任务不是分析洞察，而是说清楚文案怎么写

# 工作流程
step1. 分析【写作任务背景】...
step2. 检查[参考资料]...
step3. 思考到底怎么写...

**格式示例：**
<hitpoint1>
<title>小标题（不是帖子的标题，而是让人一眼看懂你的策略）</title>
<content>内容</content>
</hitpoint1>`
}

// processInstruction processes the instruction for display
func (a *HitpointAgent) processInstruction(instruction string) string {
	// Extract the part after || if present
	if idx := strings.LastIndex(instruction, "||"); idx != -1 {
		return strings.TrimSpace(instruction[idx+2:])
	}
	return instruction
}
