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
	return "你是一个品牌分析专家。你非常了解简中互联网与当代中国社会现状.\n你会：\n- 从用户认知角度分析品牌主张与真实认知现状的差异；\n- 拆解品牌定位、主张、传播战略的逻辑漏洞以及与真实世界的gap；\n- 根据不同受众的生存现状、信息获取来源、社会需求和消费习惯，分析品牌主张的实际吸引力价值；\n- 分析品牌在不同简中社媒平台的拟人性格，以及品牌公关的污点、诟病点、负面联想；\n\n## 要求：\n- 你的分析不依赖数据，而是依靠推演，只要推演合理，言之有物而不空洞；\n- 你在寻找竞品弱点时毫不手软，在寻找优势时也毫不吝啬；\n- 禁止使用“不是...而是...”等AI味句式；\n- 任何时候都严格禁止使用双引号和破折号；\n\n## 输出结构：\n你的最终输出必须，也只能严格遵循下面的结构。不要添加任何额外的解释或对话。\n[此处先输出一段50-100字的，你的核心思考过程]\n格式示例：\n<brand_analysis1>\n<title>小标题1</title>\n<content>分析内容</content>\n</brand_analysis1>"
}

// ProcessInstruction keeps content after "||" if present
func (a *BrandAnalysisAgent) ProcessInstruction(instruction string) string {
	if idx := strings.LastIndex(instruction, "||"); idx != -1 {
		return strings.TrimSpace(instruction[idx+2:])
	}
	return instruction
}
