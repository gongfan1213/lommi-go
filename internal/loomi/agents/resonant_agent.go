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

// ResonantAgent handles emotional resonance analysis
type ResonantAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

// NewResonantAgent creates a new resonant agent
func NewResonantAgent(logger *logx.Logger, client llm.Client) *ResonantAgent {
	baseAgent := base.NewBaseLoomiAgent("loomi_resonant_agent", logger, client)
	return &ResonantAgent{
		BaseLoomiAgent: baseAgent,
		xmlParser:      xmlx.NewLoomiXMLParser(),
	}
}

// ProcessRequest processes resonant analysis requests
func (a *ResonantAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing resonant analysis request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "resonant", req.AutoMode, req.Selections)
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

	// Extract other content outside of resonant tags
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<resonant\\d+>`})

	// Parse resonant results with unique IDs
	resonants, err := a.parseResonantsWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse resonant results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiResonant,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "Resonant parsing completed",
		logx.KV("resonants_count", len(resonants)))

	// Send resonant results
	if len(resonants) > 0 {
		// Process instruction for display
		displayInstruction := a.processInstruction(req.Instruction)

		// Build metadata
		metadata := map[string]any{"instruction": displayInstruction}
		if otherContent != "" {
			metadata["agent_other_message"] = otherContent
		}

		resonantEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiResonant,
			Data:    resonants,
			Meta:    metadata,
		}

		if err := emit(resonantEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createResonantNotes(ctx, req, resonants); err != nil {
			a.Logger.Error(ctx, "Failed to create resonant notes", logx.KV("error", err))
		}

		a.Logger.Info(ctx, "Resonant processing completed",
			logx.KV("notes_count", len(resonants)))
	} else {
		// Send raw response if no resonants found
		a.Logger.Info(ctx, "Sending raw response (no resonants parsed)")
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiResonant,
			Data:    llmResponse,
		})
	}

	return nil
}

// Process adapts to loomi.Agent interface by delegating to ProcessRequest
func (a *ResonantAgent) Process(ctx context.Context, req types.AgentRequest, emit func(ev events.StreamEvent) error) error {
	return a.ProcessRequest(ctx, req, emit)
}

// parseResonantsWithUniqueIDs parses LLM response and assigns unique IDs
func (a *ResonantAgent) parseResonantsWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.UnifiedConfigs["resonant"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	// Assign unique IDs to each resonant
	resonants := make([]map[string]any, 0, len(parseResults))

	for _, result := range parseResults {
		// Get unique ID for this resonant
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "resonant")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Ensure markdown compatibility
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("共鸣分析 %d", len(resonants)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		resonants = append(resonants, map[string]any{
			"id":      fmt.Sprintf("resonant%d", uniqueID),
			"title":   title,
			"content": content,
			"type":    result.Type,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to resonants",
		logx.KV("count", len(resonants)))

	return resonants, nil
}

// createResonantNotes creates notes for resonant analysis results
func (a *ResonantAgent) createResonantNotes(
	ctx context.Context,
	req types.AgentRequest,
	resonants []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating resonant notes",
		logx.KV("count", len(resonants)))

	for _, resonant := range resonants {
		id := resonant["id"].(string)
		title := resonant["title"].(string)
		content := resonant["content"].(string)

		err := a.CreateNote(ctx, req.UserID, req.SessionID, "resonant", id, content, title, "", nil)
		if err != nil {
			a.Logger.Error(ctx, "Failed to create resonant note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created resonant note", logx.KV("id", id))
		}
	}

	return nil
}

// getSystemPrompt returns the system prompt for resonant analysis
func (a *ResonantAgent) getSystemPrompt() string {
	return `深思品味提供的1个或多个词其背后隐含的情绪、联想，寻找这几个词之间高维的联系。

## 输出结构：
你的最终输出必须，也只能严格遵循下面的结构。不要添加任何额外的解释或对话。
[此处先输出一段50-100字的，你的核心思考过程]
格式示例：
<resonant1>
<title>小标题一</title>
<content>内容</content>
</resonant1>`
}

// processInstruction processes the instruction for display
func (a *ResonantAgent) processInstruction(instruction string) string {
	// Extract the part after || if present
	if idx := strings.LastIndex(instruction, "||"); idx != -1 {
		return strings.TrimSpace(instruction[idx+2:])
	}
	return instruction
}
