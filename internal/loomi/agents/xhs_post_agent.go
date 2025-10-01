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

// XHSPostAgent handles Xiaohongshu content creation
type XHSPostAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

// NewXHSPostAgent creates a new XHS post agent
func NewXHSPostAgent(logger *logx.Logger, client llm.Client) *XHSPostAgent {
	baseAgent := base.NewBaseLoomiAgent("loomi_xhs_post_agent", logger, client)
	return &XHSPostAgent{
		BaseLoomiAgent: baseAgent,
		xmlParser:      xmlx.NewLoomiXMLParser(),
	}
}

// ProcessRequest processes XHS content creation requests
func (a *XHSPostAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing XHS post creation request",
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
	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "xhs_post", req.AutoMode, req.Selections)
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

	// Extract other content outside of xhs_post tags
	otherContent := a.ExtractOtherContent(llmResponse, []string{`<xhs_post\\d+>`})

	// Parse XHS post results with unique IDs
	posts, err := a.parseXHSPostsWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse XHS post results", logx.KV("error", err))
		// Send raw response if parsing fails
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiXHSPost,
			Data:    llmResponse,
		})
	}

	a.Logger.Info(ctx, "XHS post parsing completed",
		logx.KV("posts_count", len(posts)))

	// Send XHS post results
	if len(posts) > 0 {
		// Build metadata
		metadata := map[string]any{"instruction": req.Instruction}
		if otherContent != "" {
			metadata["agent_other_message"] = otherContent
		}

		postEvent := events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiXHSPost,
			Data:    posts,
			Meta:    metadata,
		}

		if err := emit(postEvent); err != nil {
			return err
		}

		// Create notes concurrently
		if err := a.createXHSPostNotes(ctx, req, posts); err != nil {
			a.Logger.Error(ctx, "Failed to create XHS post notes", logx.KV("error", err))
		}

		a.Logger.Info(ctx, "XHS post processing completed",
			logx.KV("notes_count", len(posts)))
	} else {
		// Send raw response if no posts found
		a.Logger.Info(ctx, "Sending raw response (no XHS posts parsed)")
		return emit(events.StreamEvent{
			Type:    events.LLMChunk,
			Content: events.ContentLoomiXHSPost,
			Data:    llmResponse,
		})
	}

	return nil
}

// parseXHSPostsWithUniqueIDs parses LLM response and assigns unique IDs
func (a *XHSPostAgent) parseXHSPostsWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	// Parse using enhanced XML parser
	config := xmlx.ContentConfigs["xhs_post"]
	parseResults := a.xmlParser.ParseEnhanced(response, config, 1)

	// Assign unique IDs to each post
	posts := make([]map[string]any, 0, len(parseResults))

	for _, result := range parseResults {
		// Get unique ID for this post
		uniqueID, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "xhs_post")
		if err != nil {
			a.Logger.Error(ctx, "Failed to get next action ID", logx.KV("error", err))
			continue
		}

		// Extract title and cover text from content
		title, coverText, content := a.extractTitleAndCoverText(result.Content)

		// Ensure markdown compatibility
		title = a.EnsureMarkdownCompatibility(title)
		coverText = a.EnsureMarkdownCompatibility(coverText)
		content = a.EnsureMarkdownCompatibility(content)

		posts = append(posts, map[string]any{
			"id":           fmt.Sprintf("xhs_post%d", uniqueID),
			"title":        title,
			"cover_text":   coverText,
			"content":      content,
			"full_content": result.Content,
			"type":         result.Type,
		})
	}

	a.Logger.Info(ctx, "Assigned unique IDs to XHS posts",
		logx.KV("count", len(posts)))

	return posts, nil
}

// extractTitleAndCoverText extracts title and cover text from content
func (a *XHSPostAgent) extractTitleAndCoverText(content string) (string, string, string) {
	// Pattern for title and cover text extraction
	titlePattern := `标题[:：]\s*(.+?)$`
	titleRe := regexp.MustCompile(titlePattern)
	titleMatch := titleRe.FindStringSubmatch(content)

	var title string
	var cleanContent string

	if len(titleMatch) > 1 {
		title = strings.TrimSpace(titleMatch[1])
		// Remove title line from content
		lines := strings.Split(content, "\n")
		contentLines := []string{}
		titleFound := false

		for _, line := range lines {
			if !titleFound && titleRe.MatchString(line) {
				titleFound = true
				continue
			}
			if titleFound {
				contentLines = append(contentLines, line)
			}
		}

		cleanContent = strings.Join(contentLines, "\n")
	} else {
		// Try to extract from first line
		lines := strings.Split(content, "\n")
		if len(lines) > 0 {
			firstLine := strings.TrimSpace(lines[0])
			if len(firstLine) <= 50 && !strings.Contains(firstLine, "。") && !strings.Contains(firstLine, "？") && !strings.Contains(firstLine, "！") {
				title = firstLine
				cleanContent = strings.Join(lines[1:], "\n")
			} else {
				title = ""
				cleanContent = content
			}
		} else {
			title = ""
			cleanContent = content
		}
	}

	// Extract cover text (first paragraph or first few lines)
	coverText := ""
	if cleanContent != "" {
		paragraphs := strings.Split(cleanContent, "\n\n")
		if len(paragraphs) > 0 {
			coverText = strings.TrimSpace(paragraphs[0])
			// Limit cover text length
			if len(coverText) > 100 {
				coverText = coverText[:100] + "..."
			}
		}
	}

	return title, coverText, cleanContent
}

// createXHSPostNotes creates notes for XHS post results
func (a *XHSPostAgent) createXHSPostNotes(
	ctx context.Context,
	req types.AgentRequest,
	posts []map[string]any,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	a.Logger.Info(ctx, "Creating XHS post notes",
		logx.KV("count", len(posts)))

	for _, post := range posts {
		id := post["id"].(string)
		title := post["title"].(string)
		coverText := post["cover_text"].(string)
		content := post["content"].(string)

		err := a.CreateNote(ctx, req.UserID, req.SessionID, "xhs_post", id, content, title, coverText, nil)
		if err != nil {
			a.Logger.Error(ctx, "Failed to create XHS post note",
				logx.KV("id", id),
				logx.KV("error", err))
		} else {
			a.Logger.Info(ctx, "Created XHS post note", logx.KV("id", id))
		}
	}

	return nil
}

// getSystemPrompt returns the system prompt for XHS post creation
func (a *XHSPostAgent) getSystemPrompt() string {
	return `xxxxx`
}
