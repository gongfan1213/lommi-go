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

// ContentAnalysisAgent handles content analysis requests
type ContentAnalysisAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

// NewContentAnalysisAgent creates a new ContentAnalysis agent
func NewContentAnalysisAgent(logger *logx.Logger, client llm.Client) *ContentAnalysisAgent {
	baseAgent := base.NewBaseLoomiAgent("loomi_content_analysis_agent", logger, client)
	return &ContentAnalysisAgent{
		BaseLoomiAgent: baseAgent,
		xmlParser:      xmlx.NewLoomiXMLParser(),
	}
}

// ProcessRequest processes content analysis requests
func (a *ContentAnalysisAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing content analysis request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "content_analysis", req.AutoMode, req.Selections)
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

	// Extract other content outside of content_analysis tags
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<content_analysis\\d+>`})

	// Parse content analyses with unique IDs
	analyses, err := a.parseContentAnalysesWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse content analysis results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiContentAnalysis,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "Content analysis parsing completed",
		logx.KV("analyses_count", len(analyses)))

	// Send content analysis results
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
			Content: events.ContentLoomiContentAnalysis,
			Data:    analyses,
			Meta:    metadata,
		}
		if err := emit(analysisEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createContentAnalysisNotes(ctx, req, analyses); err != nil {
			a.Logger.Error(ctx, "Failed to create content analysis notes", logx.KV("error", err))
		}
	} else {
		// Send raw response if no analyses parsed
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiContentAnalysis,
			Data:    llmResponse,
		})
	}

	return nil
}

// Process adapts to loomi.Agent interface by delegating to ProcessRequest
func (a *ContentAnalysisAgent) Process(ctx context.Context, req types.AgentRequest, emit func(ev events.StreamEvent) error) error {
	return a.ProcessRequest(ctx, req, emit)
}

// parseContentAnalysesWithUniqueIDs parses LLM response and assigns unique IDs
func (a *ContentAnalysisAgent) parseContentAnalysesWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.UnifiedConfigs["content_analysis"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	analyses := make([]map[string]any, 0, len(parseResults))
	for _, result := range parseResults {
		// Get unique ID per analysis
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "content_analysis")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Ensure markdown compatibility
		title := a.EnsureMarkdownCompatibility(result.Title)
		if title == "" {
			title = fmt.Sprintf("内容分析 %d", len(analyses)+1)
		}
		content := a.EnsureMarkdownCompatibility(result.Content)

		analyses = append(analyses, map[string]any{
			"id":      fmt.Sprintf("content_analysis%d", uniqueID),
			"title":   title,
			"content": content,
			"type":    result.Type,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to content analyses",
		logx.KV("count", len(analyses)))

	return analyses, nil
}

// createContentAnalysisNotes creates notes for content analysis results
func (a *ContentAnalysisAgent) createContentAnalysisNotes(
	ctx context.Context,
	req types.AgentRequest,
	analyses []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating content analysis notes",
		logx.KV("count", len(analyses)))

	for _, analysis := range analyses {
		id := analysis["id"].(string)
		title := analysis["title"].(string)
		content := analysis["content"].(string)

		if err := a.CreateNote(ctx, req.UserID, req.SessionID, "content_analysis", id, content, title, "", nil); err != nil {
			a.Logger.Error(ctx, "Failed to create content analysis note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created content analysis note", logx.KV("id", id))
		}
	}

	return nil
}

// getSystemPrompt returns the system prompt for content analysis
func (a *ContentAnalysisAgent) getSystemPrompt() string {
	return "xxxxx"
}

// ProcessInstruction keeps content after "||" if present
func (a *ContentAnalysisAgent) ProcessInstruction(instruction string) string {
	if idx := strings.LastIndex(instruction, "||"); idx != -1 {
		return strings.TrimSpace(instruction[idx+2:])
	}
	return instruction
}
