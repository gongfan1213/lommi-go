package orchestrator

import (
	"context"
	"sync"
	"time"

	"regexp"

	"github.com/blueplan/loomi-go/internal/loomi/agents"
	contextx "github.com/blueplan/loomi-go/internal/loomi/context"
	"github.com/blueplan/loomi-go/internal/loomi/database"
	"github.com/blueplan/loomi-go/internal/loomi/events"
	"github.com/blueplan/loomi-go/internal/loomi/llm"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/notes"
	"github.com/blueplan/loomi-go/internal/loomi/pool"
	stopx "github.com/blueplan/loomi-go/internal/loomi/stop"
	"github.com/blueplan/loomi-go/internal/loomi/tokens"
	"github.com/blueplan/loomi-go/internal/loomi/types"
	"github.com/blueplan/loomi-go/internal/loomi/utils"
)

type Orchestrator struct {
	logger         *logx.Logger
	llm            llm.Client
	maxConcurrent  int
	outputInterval time.Duration
	lastOutputTime time.Time
	outputMu       sync.Mutex

	// deps
	ctxMgr   contextx.Manager
	notesSvc notes.Service
	stopMgr  stopx.Manager
	poolMgr  pool.Manager
	tokenAcc tokens.Accumulator
	persist  *database.PersistenceManager
	queue    *utils.LayeredQueue

	// stats
	concurrentPeaks []int
}

type Request struct {
	Query      string
	UserID     string
	SessionID  string
	AutoMode   bool
	Selections []string
}

func New(logger *logx.Logger, client llm.Client) *Orchestrator {
	return &Orchestrator{logger: logger, llm: client, maxConcurrent: 8, outputInterval: 10 * time.Second}
}

func (o *Orchestrator) WithDeps(ctxMgr contextx.Manager, notesSvc notes.Service, stopMgr stopx.Manager, poolMgr pool.Manager, tokenAcc tokens.Accumulator) *Orchestrator {
	o.ctxMgr = ctxMgr
	o.notesSvc = notesSvc
	o.stopMgr = stopMgr
	o.poolMgr = poolMgr
	o.tokenAcc = tokenAcc
	return o
}

func (o *Orchestrator) WithPersistence(p *database.PersistenceManager) *Orchestrator {
	o.persist = p
	return o
}

func (o *Orchestrator) WithQueue(q *utils.LayeredQueue) *Orchestrator { o.queue = q; return o }

func (o *Orchestrator) Process(ctx context.Context, req Request, emit func(ev events.StreamEvent) error) error {
	o.logger.Info(ctx, "orchestrator.start")
	defer o.logger.Info(ctx, "orchestrator.end")

	if o.poolMgr != nil {
		_, _ = o.poolMgr.Prewarm(context.Background(), []string{"high_priority", "normal"})
	}
	if o.tokenAcc != nil {
		_ = o.tokenAcc.Init(req.UserID, req.SessionID)
	}

	// 停止检查（占位）
	if o.stopMgr != nil {
		if err := o.stopMgr.Check(req.UserID, req.SessionID); err != nil {
			return err
		}
	}

	// 占位：第一轮决策（与 Python 一致，单次 LLM 决策 + 并发 action）
	messages := []llm.Message{{Role: "system", Content: "orchestrator system"}, {Role: "user", Content: req.Query}}
	buf := ""
	onChunk := func(ctx context.Context, chunk string) error {
		buf += chunk
		// 思考片段按策略转发（节流）
		if o.shouldEmit() {
			_ = emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentThought, Data: chunk})
		}
		return nil
	}
	if err := o.llm.SafeStreamCall(ctx, req.UserID, req.SessionID, messages, onChunk); err != nil {
		return err
	}
	_ = time.Now() // 预留统计点
	// 同步保存一次上下文（与 Python 行为一致：在决策后持久化上下文摘要）
	if o.persist != nil {
		_ = o.persist.SaveContext(ctx, req.UserID, req.SessionID, map[string]any{"orchestrator_buf": buf})
	}

	// 将任务入队（LayeredQueue 集成点）
	if o.queue != nil {
		_ = o.queue.Push(ctx, "orchestrator", req.SessionID)
		_ = o.queue.MarkProcessing(ctx, "orchestrator", req.SessionID)
	}

	// 占位：根据 buf 解析 Observe/Think/Actions，并并发执行子 agent，再合并结果
	// 后续将完整实现：文件上下文、notes 更新、连接池预热、并发信号量、计费摘要、停止/恢复

	actions := o.parseActions(buf)
	if len(actions) > 0 {
		if err := o.executeParallel(ctx, actions, req, emit); err != nil {
			return err
		}
	}

	if o.tokenAcc != nil {
		if summary, err := o.tokenAcc.Summary(req.UserID, req.SessionID); err == nil {
			_ = emit(events.StreamEvent{Type: events.LLMChunk, Content: events.ContentOrchestratorMessage, Data: summary})
		}
	}

	if o.queue != nil {
		_ = o.queue.MarkCompleted(ctx, "orchestrator", req.SessionID)
	}

	return nil
}

func (o *Orchestrator) shouldEmit() bool {
	o.outputMu.Lock()
	defer o.outputMu.Unlock()
	now := time.Now()
	if o.lastOutputTime.IsZero() || now.Sub(o.lastOutputTime) >= o.outputInterval {
		o.lastOutputTime = now
		return true
	}
	return false
}

type actionItem struct{ ActionType, Instruction string }

// parseActions: 占位解析，后续替换为基于 XML 的解析
func (o *Orchestrator) parseActions(response string) []actionItem {
	// 简化版解析：匹配 <Action type="xxx">...<\/Action>
	re := regexp.MustCompile(`(?s)<Action\s+type=\"([a-zA-Z0-9_\-]+)\">(.*?)</Action>`)
	matches := re.FindAllStringSubmatch(response, -1)
	if len(matches) == 0 {
		return nil
	}
	items := make([]actionItem, 0, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		items = append(items, actionItem{ActionType: m[1], Instruction: m[2]})
	}
	return items
}

func (o *Orchestrator) executeParallel(ctx context.Context, items []actionItem, req Request, emit func(ev events.StreamEvent) error) error {
	if len(items) == 0 {
		return nil
	}
	sem := make(chan struct{}, o.maxConcurrent)
	errCh := make(chan error, len(items))
	done := make(chan struct{})
	active := 0
	peak := 0
	mu := sync.Mutex{}

	for _, it := range items {
		sem <- struct{}{}
		go func(ai actionItem) {
			defer func() { <-sem }()
			mu.Lock()
			active++
			if active > peak {
				peak = active
			}
			mu.Unlock()
			ag := o.createAgent(ai.ActionType)
			if ag == nil {
				// 未知类型，忽略
				errCh <- nil
				mu.Lock()
				active--
				mu.Unlock()
				return
			}
			aReq := types.AgentRequest{
				Instruction: ai.Instruction,
				UserID:      req.UserID,
				SessionID:   req.SessionID,
				UseFiles:    false,
				AutoMode:    req.AutoMode,
				Selections:  req.Selections,
			}
			err := ag.ProcessRequest(ctx, aReq, emit)
			errCh <- err
			mu.Lock()
			active--
			mu.Unlock()
		}(it)
	}

	go func() {
		// 等待所有 goroutine 完成
		for i := 0; i < cap(sem); i++ {
			sem <- struct{}{}
		}
		close(done)
	}()

	// 收集错误
	var firstErr error
	for i := 0; i < len(items); i++ {
		if e := <-errCh; e != nil && firstErr == nil {
			firstErr = e
		}
	}
	<-done
	// 记录峰值
	o.concurrentPeaks = append(o.concurrentPeaks, peak)
	return firstErr
}

func (o *Orchestrator) createAgent(actionType string) types.Agent {
	switch actionType {
	case "knowledge":
		return agents.NewKnowledgeAgent(o.logger, o.llm)
	case "persona":
		return agents.NewPersonaAgent(o.logger, o.llm)
	case "websearch":
		return agents.NewWebSearchAgent(o.logger, o.llm)
	case "resonant":
		return agents.NewResonantAgent(o.logger, o.llm)
	case "revision":
		return agents.NewRevisionAgent(o.logger, o.llm)
	case "tiktok_script":
		return agents.NewTikTokScriptAgent(o.logger, o.llm)
	case "brand_analysis":
		return agents.NewBrandAnalysisAgent(o.logger, o.llm)
	case "content_analysis":
		return agents.NewContentAnalysisAgent(o.logger, o.llm)
	default:
		return nil
	}
}
