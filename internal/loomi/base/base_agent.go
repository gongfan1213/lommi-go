//go:build !api_lite

package base

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	contextx "github.com/blueplan/loomi-go/internal/loomi/context"
	"github.com/blueplan/loomi-go/internal/loomi/events"
	"github.com/blueplan/loomi-go/internal/loomi/llm"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/notes"
	poolx "github.com/blueplan/loomi-go/internal/loomi/pool"
	stopx "github.com/blueplan/loomi-go/internal/loomi/stop"
	"github.com/blueplan/loomi-go/internal/loomi/tokens"
	"github.com/blueplan/loomi-go/internal/loomi/types"
	"github.com/blueplan/loomi-go/internal/loomi/utils/markdown"
)

// BaseLoomiAgent is the foundation for all Loomi agents
type BaseLoomiAgent struct {
	AgentName        string
	Logger           *logx.Logger
	LLMClient        llm.Client
	ContextManager   contextx.Manager
	NotesService     notes.Service
	StopManager      stopx.Manager
	TokenAccumulator tokens.Accumulator
	PoolManager      poolx.Manager

	// Session management
	CurrentUserID    string
	CurrentSessionID string
	CurrentTokenKey  string

	// Performance configuration
	EnableThoughtStreaming bool
	ThoughtMinLength       int
	ThoughtBatchSize       int
	EnableFastMode         bool

	// Redis pool type
	RedisPoolType string

	// Stream storage
	StreamStorageEnabled bool

	// Disconnect detection
	LastDisconnectLogTime map[string]time.Time
	DisconnectLogInterval time.Duration
}

// AgentRequest represents a request to an agent
type AgentRequest struct {
	Instruction string
	UserID      string
	SessionID   string
	UseFiles    bool
	FileIDs     []string
	AutoMode    bool
	Selections  []string
}

// NewBaseLoomiAgent creates a new base Loomi agent
func NewBaseLoomiAgent(agentName string, logger *logx.Logger, llmClient llm.Client) *BaseLoomiAgent {
	return &BaseLoomiAgent{
		AgentName:              agentName,
		Logger:                 logger,
		LLMClient:              llmClient,
		EnableThoughtStreaming: true,
		ThoughtMinLength:       10,
		ThoughtBatchSize:       5,
		EnableFastMode:         false,
		RedisPoolType:          determineRedisPoolType(agentName),
		StreamStorageEnabled:   true,
		LastDisconnectLogTime:  make(map[string]time.Time),
		DisconnectLogInterval:  60 * time.Second,
	}
}

// WithDependencies sets the agent's dependencies
func (a *BaseLoomiAgent) WithDependencies(
	ctxMgr contextx.Manager,
	notesSvc notes.Service,
	stopMgr stopx.Manager,
	poolMgr poolx.Manager,
	tokenAcc tokens.Accumulator,
) *BaseLoomiAgent {
	a.ContextManager = ctxMgr
	a.NotesService = notesSvc
	a.StopManager = stopMgr
	a.PoolManager = poolMgr
	a.TokenAccumulator = tokenAcc
	return a
}

// WithDefaultDependencies sets default in-memory implementations for dependencies
func (a *BaseLoomiAgent) WithDefaultDependencies() *BaseLoomiAgent {
	a.ContextManager = nil // Will use fallback methods
	a.NotesService = notes.NewInmem()
	a.StopManager = stopx.NewInmem()
	a.PoolManager = poolx.NewInmem()
	a.TokenAccumulator = tokens.NewInmem()
	return a
}

// ProcessRequest is the default implementation that should be overridden by specific agents
func (a *BaseLoomiAgent) ProcessRequest(
	ctx context.Context,
	req types.AgentRequest,
	emit func(ev events.StreamEvent) error,
) error {
	// Default implementation: return a system message
	msg := fmt.Sprintf("🔧 %s 基类默认响应：请在子类中实现具体的ProcessRequest方法", a.AgentName)
	a.Logger.Warn(ctx, "使用了默认的ProcessRequest实现，应在子类中重写")

	return emit(events.StreamEvent{
		Type:    events.LLMChunk,
		Content: events.ContentSystemMessage,
		Data:    msg,
	})
}

// SetCurrentSession sets the current session information
func (a *BaseLoomiAgent) SetCurrentSession(userID, sessionID string) {
	a.CurrentUserID = userID
	a.CurrentSessionID = sessionID
}

// ClearCurrentSession clears the current session information
func (a *BaseLoomiAgent) ClearCurrentSession() {
	a.CurrentUserID = ""
	a.CurrentSessionID = ""
}

// CheckAndRaiseIfStopped checks if the current session has been stopped
func (a *BaseLoomiAgent) CheckAndRaiseIfStopped(ctx context.Context, userID, sessionID string) error {
	if a.StopManager == nil {
		return nil
	}

	stopped, err := a.StopManager.IsStopped(userID, sessionID)
	if err != nil {
		return err
	}

	if stopped {
		return fmt.Errorf("execution stopped for session %s", sessionID)
	}

	return nil
}

// ClearStopState clears the stop state for a session
func (a *BaseLoomiAgent) ClearStopState(ctx context.Context, userID, sessionID string) error {
	if a.StopManager == nil {
		return nil
	}

	return a.StopManager.ClearStopState(userID, sessionID)
}

// GetNextActionID gets the next available action ID for a session
func (a *BaseLoomiAgent) GetNextActionID(ctx context.Context, userID, sessionID, action string) (int, error) {
	if a.ContextManager == nil {
		// Fallback: count existing notes
		if a.NotesService != nil {
			notes, err := a.NotesService.GetByAction(userID, sessionID, action)
			if err != nil {
				return 0, err
			}
			return len(notes) + 1, nil
		}
		// Emergency fallback: timestamp-based ID
		return int(time.Now().UnixNano() % 1000000), nil
	}

	return a.ContextManager.NextActionID(userID, sessionID, action)
}

// CreateNote creates a new note
func (a *BaseLoomiAgent) CreateNote(
	ctx context.Context,
	userID, sessionID, action, name, contextStr string,
	title, coverTitle string,
	selectStatus *int,
) error {
	if a.NotesService == nil {
		return fmt.Errorf("notes service not available")
	}

	// Auto-determine select status if not provided
	if selectStatus == nil {
		autoSelect := a.getAutoSelectStatus(action)
		selectStatus = &autoSelect
	}

	return a.NotesService.Create(userID, sessionID, action, name, title, contextStr, *selectStatus)
}

// ShouldEmitThought determines if thought content should be emitted
func (a *BaseLoomiAgent) ShouldEmitThought(content string) bool {
	if !a.EnableThoughtStreaming {
		return false
	}

	if len(strings.TrimSpace(content)) < a.ThoughtMinLength {
		return false
	}

	return true
}

// SafeStreamCall performs a safe streaming LLM call with stop checking
func (a *BaseLoomiAgent) SafeStreamCall(
	ctx context.Context,
	userID, sessionID string,
	messages []llm.Message,
	onChunk func(ctx context.Context, chunk string) error,
) error {
	// Check stop status before starting
	if err := a.CheckAndRaiseIfStopped(ctx, userID, sessionID); err != nil {
		return err
	}

	// Set current session
	a.SetCurrentSession(userID, sessionID)
	defer a.ClearCurrentSession()

	// Prepare token accumulator key
	if a.TokenAccumulator != nil {
		_, err := a.TokenAccumulator.Initialize(userID, sessionID)
		if err != nil {
			a.Logger.Error(ctx, "Failed to initialize token accumulator", logx.KV("error", err))
		}
	}

	// Perform streaming call
	chunkCount := 0
	stopCheckInterval := 10

	err := a.LLMClient.SafeStreamCall(ctx, userID, sessionID, messages, func(ctx context.Context, chunk string) error {
		// Check stop status periodically
		chunkCount++
		if chunkCount%stopCheckInterval == 0 {
			if err := a.CheckAndRaiseIfStopped(ctx, userID, sessionID); err != nil {
				return err
			}
		}
		// 近似按 chunk 长度统计 token（可替换为 provider 返回的token用量）
		if a.TokenAccumulator != nil {
			n := len([]rune(chunk)) / 4
			if n < 1 {
				n = 1
			}
			a.TokenAccumulator.Add(userID, sessionID, n)
		}
		return onChunk(ctx, chunk)
	})

	// Final stop check
	if err := a.CheckAndRaiseIfStopped(ctx, userID, sessionID); err != nil {
		if a.TokenAccumulator != nil {
			// 若 LLM client 支持用量，优先按 provider 用量补一次精确统计
			if up, ok := a.LLMClient.(interface {
				LastUsage(string, string) (int, int, bool)
			}); ok {
				if p, c, ok2 := up.LastUsage(userID, sessionID); ok2 {
					// 简单合并：prompt+completion
					total := p + c
					if total > 0 {
						a.TokenAccumulator.Add(userID, sessionID, total)
					}
				}
			}
			if _, sumErr := a.TokenAccumulator.Summary(userID, sessionID); sumErr != nil {
				a.Logger.Error(ctx, "token.summary.error", logx.KV("error", sumErr))
			}
			if cleanErr := a.TokenAccumulator.Cleanup(userID, sessionID); cleanErr != nil {
				a.Logger.Error(ctx, "token.cleanup.error", logx.KV("error", cleanErr))
			}
		}
		return err
	}

	return err
}

// BuildCleanAgentPrompt builds a clean prompt for agents
func (a *BaseLoomiAgent) BuildCleanAgentPrompt(
	ctx context.Context,
	userID, sessionID, instruction, action string,
	autoMode bool,
	userSelections []string,
) (string, error) {
	if a.ContextManager == nil {
		return instruction, nil
	}

	// Build context for the prompt
	contextStr, err := a.ContextManager.FormatContextForPrompt(userID, sessionID, a.AgentName, true, true, true, userSelections, false, false)
	if err != nil {
		a.Logger.Error(ctx, "Failed to format context for prompt", logx.KV("error", err))
		return instruction, nil
	}

	if contextStr != "" {
		return fmt.Sprintf("%s\n\n%s", instruction, contextStr), nil
	}

	return instruction, nil
}

// ExtractOtherContent extracts content outside of specified tags
func (a *BaseLoomiAgent) ExtractOtherContent(response string, tagPatterns []string) string {
	cleaned := response

	for _, pattern := range tagPatterns {
		// Convert pattern to regex
		regexPattern := strings.Replace(pattern, "\\d+", "\\d+", -1)
		startPattern := regexPattern
		endPattern := strings.Replace(regexPattern, "<", "</", 1)

		// Remove complete tag blocks
		fullPattern := fmt.Sprintf("%s.*?%s", startPattern, endPattern)
		re := regexp.MustCompile(fullPattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Clean up extra whitespace
	re := regexp.MustCompile(`\n\s*\n`)
	cleaned = re.ReplaceAllString(cleaned, "\n")
	cleaned = strings.TrimSpace(cleaned)

	// Return if there's substantial content
	if len(cleaned) > 10 {
		return cleaned
	}

	return ""
}

// EnsureMarkdownCompatibility ensures markdown content is compatible
func (a *BaseLoomiAgent) EnsureMarkdownCompatibility(text string) string {
	return markdown.EnsureCompatibility(text)
}

// Helper functions

func determineRedisPoolType(agentName string) string {
	if strings.Contains(strings.ToLower(agentName), "orchestrator") ||
		strings.Contains(strings.ToLower(agentName), "concierge") {
		return "high_priority"
	}
	return "normal"
}

func (a *BaseLoomiAgent) getAutoSelectStatus(action string) int {
	autoSelectActions := map[string]bool{
		"websearch":        true,
		"persona":          true,
		"brand_analysis":   true,
		"knowledge":        true,
		"content_analysis": true,
		"resonant":         true,
	}

	userSelectActions := map[string]bool{
		"hitpoint":       true,
		"xhs_post":       true,
		"wechat_article": true,
		"tiktok_script":  true,
		"revision":       true,
	}

	if autoSelectActions[action] {
		return 1
	}

	if userSelectActions[action] {
		return 0
	}

	a.Logger.Warn(context.Background(), "Unknown action type, defaulting to user selection", logx.KV("action", action))
	return 0
}

// GetAgentTemperature returns the temperature setting for the agent
func (a *BaseLoomiAgent) GetAgentTemperature() float64 {
	highTempAgents := map[string]bool{
		"loomi_hitpoint_agent":       true,
		"loomi_tiktok_script_agent":  true,
		"loomi_wechat_article_agent": true,
		"loomi_xhs_post_agent":       true,
		"loomi_revision_agent":       true,
	}

	switch a.AgentName {
	case "loomi_orchestrator":
		return 0.5
	case "loomi_concierge":
		return 0.4
	case "loomi_knowledge_agent":
		return 0.3
	case "loomi_websearch_agent":
		return 0.1
	default:
		if highTempAgents[a.AgentName] {
			return 0.6
		}
		return 0.4
	}
}

// GetThinkingBudget returns the thinking budget for the agent
func (a *BaseLoomiAgent) GetThinkingBudget() int {
	highBudgetAgents := map[string]bool{
		"loomi_hitpoint_agent":       true,
		"loomi_tiktok_script_agent":  true,
		"loomi_wechat_article_agent": true,
		"loomi_xhs_post_agent":       true,
		"loomi_orchestrator":         true,
	}

	if highBudgetAgents[a.AgentName] {
		return 500
	}
	return 128
}
