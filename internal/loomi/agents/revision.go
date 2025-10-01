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

type RevisionAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

func NewRevisionAgent(logger *logx.Logger, client llm.Client) *RevisionAgent {
	base := base.NewBaseLoomiAgent("loomi_revision_agent", logger, client)
	return &RevisionAgent{BaseLoomiAgent: base, xmlParser: xmlx.NewLoomiXMLParser()}
}

func (a *RevisionAgent) Process(ctx context.Context, req types.AgentRequest, emit func(ev events.StreamEvent) error) error {
	return a.ProcessRequest(ctx, req, emit)
}

func (a *RevisionAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing revision request",
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

	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "revision", req.AutoMode, req.Selections)
	if err != nil {
		a.Logger.Error(ctx, "Failed to build prompt", logx.KV("error", err))
		userPrompt = req.Instruction
	}

	messages := []llm.Message{{Role: "system", Content: a.getSystemPrompt()}, {Role: "user", Content: userPrompt}}

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

	config := xmlx.UnifiedConfigs["revision"]
	parsed := a.xmlParser.ParseEnhanced(llmResponse, config, 1)
	items := make([]map[string]any, 0, len(parsed))
	for _, r := range parsed {
		uid, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "revision")
		if err != nil {
			a.Logger.Error(ctx, "NextActionID failed", logx.KV("error", err))
			continue
		}
		title := a.EnsureMarkdownCompatibility(r.Title)
		if title == "" {
			title = fmt.Sprintf("修订建议 %d", len(items)+1)
		}
		content := a.EnsureMarkdownCompatibility(r.Content)
		items = append(items, map[string]any{"id": fmt.Sprintf("revision%d", uid), "title": title, "content": content, "type": r.Type})
	}

	if len(items) > 0 {
		ev := events.StreamEvent{Type: events.LLMChunk, Content: events.ContentLoomiRevision, Data: items, Meta: map[string]any{"instruction": req.Instruction}}
		if err := emit(ev); err != nil {
			return err
		}
		for _, it := range items {
			_ = a.CreateNote(ctx, req.UserID, req.SessionID, "revision", it["id"].(string), it["content"].(string), it["title"].(string), "", nil)
		}
		return nil
	}

	return emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentLoomiRevision, Data: llmResponse})
}

func (a *RevisionAgent) getSystemPrompt() string {
	return `xxxx`
}
