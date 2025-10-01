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

// BrandAnalysisAgent handles brand analysis requests
type BrandAnalysisAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

// NewBrandAnalysisAgent creates a new BrandAnalysis agent
func NewBrandAnalysisAgent(logger *logx.Logger, client llm.Client) *BrandAnalysisAgent {
	baseAgent := base.NewBaseLoomiAgent("loomi_brand_analysis_agent", logger, client)
	return &BrandAnalysisAgent{
		BaseLoomiAgent: baseAgent,
		xmlParser:      xmlx.NewLoomiXMLParser(),
	}
}

// ProcessRequest processes brand analysis requests
func (a *BrandAnalysisAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing brand analysis request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "brand_analysis", req.AutoMode, req.Selections)
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

	// Extract other content outside of brand_analysis tags
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<brand_analysis\\d+>`})

	// Parse brand analyses with unique IDs
	analyses, err := a.parseBrandAnalysesWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse brand analysis results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiBrandAnalysis,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "Brand analysis parsing completed",
		logx.KV("analyses_count", len(analyses)))

	// Send brand analysis results
	if len(analyses) > 0 {
		// Only keep part after || in instruction
		displayInstruction := a.ProcessInstruction(req.Instruction)

		// Build metadata
		metadata := map[string]any{"instruction": displayInstruction}
		if otherContent != "" {
			metadata["agent_other_message"] = otherContent
		}

		analysisEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiBrandAnalysis,
			Data:    analyses,
			Meta:    metadata,
		}
		if err := emit(analysisEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createBrandAnalysisNotes(ctx, req, analyses); err != nil {
			a.Logger.Error(ctx, "Failed to create brand analysis notes", logx.KV("error", err))
		}
	} else {
		// Send raw response if no analyses parsed
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiBrandAnalysis,
			Data:    llmResponse,
		})
	}

	return nil
}

// Process adapts to loomi.Agent interface by delegating to ProcessRequest
func (a *BrandAnalysisAgent) Process(ctx context.Context, req types.AgentRequest, emit func(ev events.StreamEvent) error) error {
	return a.ProcessRequest(ctx, req, emit)
}

// parseBrandAnalysesWithUniqueIDs parses LLM response and assigns unique IDs
func (a *BrandAnalysisAgent) parseBrandAnalysesWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.UnifiedConfigs["brand_analysis"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	analyses := make([]map[string]any, 0, len(parseResults))
	for _, result := range parseResults {
		// Get unique ID per analysis
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "brand_analysis")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Ensure markdown compatibility
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("品牌分析 %d", len(analyses)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		analyses = append(analyses, map[string]any{
			"id":      fmt.Sprintf("brand_analysis%d", uniqueID),
			"title":   title,
			"type":    result.Type,
			"content": content,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to brand analyses",
		logx.KV("count", len(analyses)))

	// Fallback: if fewer than 3 results and response has content, wrap whole response as one analysis
	if len(analyses) < 3 && len(response) > 0 {
		clean := a.xmlParser.CleanXMLTags(response, "brand_analysis")
		if clean != "" {
			uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "brand_analysis")
			if err == nil {
				analyses = append(analyses, map[string]any{
					"id":      fmt.Sprintf("brand_analysis%d", uniqueID),
					"title":   fmt.Sprintf("品牌分析 %d", len(analyses)+1),
					"type":    "brand_analysis",
					"content": a.EnsureMarkdownCompatibility(clean),
				})
			}
		}
	}

	return analyses, nil
}

// createBrandAnalysisNotes creates notes for brand analysis results
func (a *BrandAnalysisAgent) createBrandAnalysisNotes(
	ctx context.Context,
	req types.AgentRequest,
	analyses []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating brand analysis notes",
		logx.KV("count", len(analyses)))

	for _, analysis := range analyses {
		id := analysis["id"].(string)
		title := analysis["title"].(string)
		content := analysis["content"].(string)

		if err := a.CreateNote(ctx, req.UserID, req.SessionID, "brand_analysis", id, content, title, "", nil); err != nil {
			a.Logger.Error(ctx, "Failed to create brand analysis note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created brand analysis note", logx.KV("id", id))
		}
	}

	return nil
}

// getSystemPrompt returns the system prompt for brand analysis
func (a *BrandAnalysisAgent) getSystemPrompt() string {
	return "xxxxx"
}

// ProcessInstruction keeps content after "||" if present
func (a *BrandAnalysisAgent) ProcessInstruction(instruction string) string {
	if idx := strings.LastIndex(instruction, "||"); idx != -1 {
		return strings.TrimSpace(instruction[idx+2:])
	}
	return instruction
}
