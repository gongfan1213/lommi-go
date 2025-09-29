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

type TikTokScriptAgent struct {
	*base.BaseLoomiAgent
	xmlParser *xmlx.LoomiXMLParser
}

func NewTikTokScriptAgent(logger *logx.Logger, client llm.Client) *TikTokScriptAgent {
	base := base.NewBaseLoomiAgent("loomi_tiktok_script_agent", logger, client)
	return &TikTokScriptAgent{BaseLoomiAgent: base, xmlParser: xmlx.NewLoomiXMLParser()}
}

func (a *TikTokScriptAgent) Process(ctx context.Context, req types.AgentRequest, emit func(ev events.StreamEvent) error) error {
	// Delegate to ProcessRequest for uniform behavior
	return a.ProcessRequest(ctx, req, emit)
}

func (a *TikTokScriptAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	a.Logger.Info(ctx, "Processing TikTok script request",
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

	userPrompt, err := a.BuildCleanAgentPrompt(ctx, req.UserID, req.SessionID, req.Instruction, "tiktok_script", req.AutoMode, req.Selections)
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

	// Parse results with cover_text and hook, and assign unique IDs
	config := xmlx.UnifiedConfigs["tiktok_script"]
	parsed := a.xmlParser.ParseEnhanced(llmResponse, config, 1)
	items := make([]map[string]any, 0, len(parsed))
	for _, r := range parsed {
		uid, err := a.GetNextActionID(ctx, req.UserID, req.SessionID, "tiktok_script")
		if err != nil {
			a.Logger.Error(ctx, "NextActionID failed", logx.KV("error", err))
			continue
		}
		title := a.EnsureMarkdownCompatibility(r.Title)
		content := a.EnsureMarkdownCompatibility(r.Content)
		coverText := a.EnsureMarkdownCompatibility(r.CoverText)
		hook := a.EnsureMarkdownCompatibility(r.Hook)
		if title == "" {
			title = fmt.Sprintf("抖音脚本 %d", len(items)+1)
		}
		items = append(items, map[string]any{
			"id":         fmt.Sprintf("tiktok_script%d", uid),
			"title":      title,
			"content":    content,
			"cover_text": coverText,
			"hook":       hook,
			"type":       r.Type,
		})
	}

	if len(items) > 0 {
		meta := map[string]any{"instruction": req.Instruction}
		ev := events.StreamEvent{Type: events.LLMChunk, Content: events.ContentLoomiTikTokScript, Data: items, Meta: meta}
		if err := emit(ev); err != nil {
			return err
		}
		// Create notes (store pure content; pass title/cover to DB)
		for _, it := range items {
			_ = a.CreateNote(ctx, req.UserID, req.SessionID, "tiktok_script", it["id"].(string), it["content"].(string), it["title"].(string), it["cover_text"].(string), nil)
		}
		return nil
	}

	// Fallback raw
	return emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentLoomiTikTokScript, Data: llmResponse})
}

func (a *TikTokScriptAgent) getSystemPrompt() string {
	return `你是一个极具网感的抖音短视频口播稿写手。你需要产出1篇没有AI味的抖音口播稿。
口播稿是指，出镜角色对着镜头直接念的稿子或旁白稿。
（此处省略若干详细规则，保持与 Python 提示一致的核心约束）

请使用如下 XML 输出：

<tiktok_script1>
<title>标题</title>
<content>口播稿正文</content>
</tiktok_script1>`
}
