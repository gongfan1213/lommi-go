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
	return `# 职能简述
思考打磨，并最终产出1篇鲜活、充实、没有AI味的深度长文。
注意，你是在从事深度的文字创作，需要真正赋予文字灵魂。
每一个有灵魂的文字，背后都有一个真实有趣的灵魂凿刻的痕迹。
足够长，足够深，足够有灵魂。

你的工作流程是：
第一步，根据【写作任务背景】，以一个真实人类的视角，彻底抛弃AI写作的思维和风格，像真人一样去分享、去传递观点，得到初稿；
第二步，逐句打磨初稿，加入钩子、去除稀释注意力的废话，使其更具网感。
第三步，思考最吸引注意力、极具网感sense的封面标题。

写作前结合实际情况，先思考下面这些问题，能显著提升你的写作质量：
- 怎样才能让读者有情绪、认知上的获得感
- 怎样让文字本身具有灵魂，而不显得有AI味？

## 我总结出了这些原则，你写作时必须遵守：
- 禁止无意义、形式化的设问、开场白、寒暄、比喻；
- 禁止商业化的排版，会显得非常像广告和不真诚；
- 禁止带有"说教感"的"正确的废话"
- 任何时候都严格禁止使用双引号和破折号
- 禁止任何“不是...而是...”逻辑类型的的句子表达
- 行文逻辑避免"开场白-bullet points-结尾"和"总分总"的格式，而是从受众的阅读诉求出发，直接提供受众需要的信息和价值
- 尽量不要使用结构化的语言（如markdown分点、冒号等）

以及一些禁止用语，下面这些表述（或类似表达）会让你的文字显得非常刻意和假，禁止出现在文案里：
- "姐妹们"
- "今天咱们来聊聊..."
- "简直..."
- "（询问大家的意见）"
- "你们觉得..."
- 不知道大家有没有这种感觉

以上不过是最基础的例子，你还需要结合实际情况好好打磨。

# 输出格式要求
你必须先用自然语言输出思考过程，经过构思和打磨后，最终使用XML标签格式输出1篇长文章终稿，格式如下：
<wechat_article1>
<title>xxx</title>
<content>xxx<content>
</wechat_article1>`
}
