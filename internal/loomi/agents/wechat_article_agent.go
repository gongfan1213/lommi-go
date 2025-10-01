package agents

import (
	"context"
	"fmt"

	"github.com/blueplan/loomi-go/internal/loomi/base"
	"github.com/blueplan/loomi-go/internal/loomi/events"
	"github.com/blueplan/loomi-go/internal/loomi/llm"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/types"
	xmlx "github.com/blueplan/loomi-go/internal/loomi/utils/xml"
)

// WeChatArticleAgent handles WeChat article creation
type WeChatArticleAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

// NewWeChatArticleAgent creates a new WeChat article agent
func NewWeChatArticleAgent(logger *logx.Logger, client llm.Client) *WeChatArticleAgent {
	baseAgent := base.NewBaseLoomiAgent("loomi_wechat_article_agent", logger, client)
	return &WeChatArticleAgent{
		BaseLoomiAgent: baseAgent,
		xmlParser:      xmlx.NewLoomiXMLParser(),
	}
}

// ProcessRequest processes WeChat article creation requests
func (a *WeChatArticleAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing WeChat article creation request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "wechat_article", req.AutoMode, req.Selections)
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

	// Extract other content outside of wechat_article tags
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<wechat_article\\d+>`})

	// Parse WeChat article results with unique IDs
	articles, err := a.parseWeChatArticlesWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse WeChat article results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiConcierge,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "WeChat article parsing completed",
		logx.KV("articles_count", len(articles)))

	// Send WeChat article results
	if len(articles) > 0 {
		// Build metadata
		metadata := map[string]any{"instruction": req.Instruction}
		if otherContent != "" {
			metadata["agent_other_message"] = otherContent
		}

		articleEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiWeChatArticle,
			Data:    articles,
			Meta:    metadata,
		}

		if err := emit(articleEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createWeChatArticleNotes(ctx, req, articles); err != nil {
			a.Logger.Error(ctx, "Failed to create WeChat article notes", logx.KV("error", err))
		}

		a.Logger.Info(ctx, "WeChat article processing completed",
			logx.KV("notes_count", len(articles)))
	} else {
		// Send raw response if no articles found
		a.Logger.Info(ctx, "Sending raw response (no WeChat articles parsed)")
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiWeChatArticle,
			Data:    llmResponse,
		})
	}

	return nil
}

// parseWeChatArticlesWithUniqueIDs parses LLM response and assigns unique IDs
func (a *WeChatArticleAgent) parseWeChatArticlesWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.ContentConfigs["wechat_article"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	// Assign unique IDs to each article
	articles := make([]map[string]any, 0, len(parseResults))

	for _, result := range parseResults {
		// Get unique ID for this article
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "wechat_article")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Ensure markdown compatibility
		title := a.EnsureMarkdownCompatibility(result.Title)
		content := a.EnsureMarkdownCompatibility(result.Content)

		articles = append(articles, map[string]any{
			"id":           fmt.Sprintf("wechat_article%d", uniqueID),
			"title":        title,
			"content":      content,
			"full_content": result.Content,
			"type":         result.Type,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to WeChat articles",
		logx.KV("count", len(articles)))

	return articles, nil
}

// createWeChatArticleNotes creates notes for WeChat article results
func (a *WeChatArticleAgent) createWeChatArticleNotes(
	ctx context.Context,
	req types.AgentRequest,
	articles []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating WeChat article notes",
		logx.KV("count", len(articles)))

	for _, article := range articles {
		id := article["id"].(string)
		title := article["title"].(string)
		content := article["content"].(string)

		err := a.CreateNote(ctx, req.UserID, req.SessionID, "wechat_article", id, content, title, "", nil)
		if err != nil {
			a.Logger.Error(ctx, "Failed to create WeChat article note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created WeChat article note", logx.KV("id", id))
		}
	}

	return nil
}

// getSystemPrompt returns the system prompt for WeChat article creation
func (a *WeChatArticleAgent) getSystemPrompt() string {
	return ``
}
