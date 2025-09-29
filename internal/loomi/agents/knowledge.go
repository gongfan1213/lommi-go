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

// KnowledgeAgent provides knowledge responses with structured XML output parsing
type KnowledgeAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

// NewKnowledgeAgent creates a new knowledge agent
func NewKnowledgeAgent(logger *logx.Logger, client llm.Client) *KnowledgeAgent {
	base := base.NewBaseLoomiAgent("loomi_knowledge_agent", logger, client)
	return &KnowledgeAgent{BaseLoomiAgent: base, xmlParser: xmlx.NewLoomiXMLParser()}
}

// ProcessRequest handles knowledge requests
func (a *KnowledgeAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing knowledge request",
		logx.KV("user_id", req.UserID),
		logx.KV("session_id", req.SessionID),
		logx.KV("instruction_length", len(req.Instruction)))

	a.SetCurrentSession(req.UserID, req.SessionID)
	defer a.ClearCurrentSession()

	if err := a.ClearStopState(ctx, req.UserID, req.SessionID); err != nil {
		a.Logger.Error(ctx, "Failed to clear stop state", logx.KV("error", err))
	}
	if err := a.CheckAndRaiseIfStopped(ctx, req.UserID, req.SessionID); err != nil {
		return err
	}

	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "knowledge", req.AutoMode, req.Selections)
	if err != nil {
		a.Logger.Error(ctx, "Failed to build prompt", logx.KV("error", err))
		userPrompt = req.Instruction
	}

	messages := []llm.Message{
		{Role: "system", Content: a.getSystemPrompt()},
		{Role: "user", Content: userPrompt},
	}

	a.Logger.Info(ctx, "Starting LLM response collection")
	llmResponse := ""
	err = a.SafeStreamCall(ctx, req.UserID, req.SessionID, messages, func(ctx context.Context, chunk string) error {
		llmResponse += chunk
		if a.ShouldEmitThought(chunk) {
			if err := emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentThought, Data: chunk}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	a.Logger.Info(ctx, "LLM response collection completed", logx.KV("response_length", len(llmResponse)))
	if err := a.CheckAndRaiseIfStopped(ctx, req.UserID, req.SessionID); err != nil {
		return err
	}

	otherContent := a.ExtractOtherContent(llmResponse, []string{`<knowledge\\d+>`})

	// Parse knowledge with unique IDs
	items, err := a.parseKnowledgeWithUniqueIDs(ctx, llmResponse, req)
	if err != nil {
		a.Logger.Error(ctx, "Failed to parse knowledge", logx.KV("error", err))
		return emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentLoomiKnowledge, Data: llmResponse})
	}

	if len(items) > 0 {
		displayInstruction := a.processInstruction(req.Instruction)
		meta := map[string]any{"instruction": displayInstruction}
		if otherContent != "" {
			meta["agent_other_message"] = otherContent
		}
		if err := emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentLoomiKnowledge, Data: items, Meta: meta}); err != nil {
			return err
		}

		// Create notes
		for _, it := range items {
			_ = a.CreateNote(ctx, req.UserID, req.SessionID, "knowledge", it["id"].(string), it["content"].(string), it["title"].(string), "", nil)
		}
		return nil
	}

	return emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentLoomiKnowledge, Data: llmResponse})
}

func (a *KnowledgeAgent) parseKnowledgeWithUniqueIDs(
	ctx context.Context,
	response string,
	req types.AgentRequest,
) ([]map[string]any, error) {
	config := xmlx.UnifiedConfigs["knowledge"]
	parsed := a.xmlParser.ParseEnhanced(response, config, 1)
	items := make([]map[string]any, 0, len(parsed))
	for _, r := range parsed {
		uid, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "knowledge")
		if err != nil {
			a.Logger.Error(ctx, "NextActionID failed", logx.KV("error", err))
			continue
		}
		title := a.EnsureMarkdownCompatibility(r.Title)
		if title == "" {
			title = fmt.Sprintf("知识点 %d", len(items)+1)
		}
		content := a.EnsureMarkdownCompatibility(r.Content)
		items = append(items, map[string]any{
			"id":      fmt.Sprintf("knowledge%d", uid),
			"title":   title,
			"content": content,
			"type":    r.Type,
		})
	}
	return items, nil
}

func (a *KnowledgeAgent) processInstruction(instruction string) string {
	if idx := strings.LastIndex(instruction, "||"); idx != -1 {
		return strings.TrimSpace(instruction[idx+2:])
	}
	return instruction
}

func (a *KnowledgeAgent) getSystemPrompt() string {
	return "你是一个专业的知识查询助手。请基于你内在知识提供高质量、结构化的答案，并使用 <knowledgeN> 包裹的 XML 结构输出结果。"
}
