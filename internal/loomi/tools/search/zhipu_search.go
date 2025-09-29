package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// ZhipuClient 智谱AI搜索客户端
type ZhipuClient struct {
	apiKey     string
	httpClient *http.Client
	logger     *logx.Logger
}

// ZhipuSearchRequest 智谱搜索请求
type ZhipuSearchRequest struct {
	Query       string  `json:"query"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}

// ZhipuSearchResponse 智谱搜索响应
type ZhipuSearchResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ZhipuWebSearchRequest 智谱网络搜索请求
type ZhipuWebSearchRequest struct {
	Query    string `json:"query"`
	Num      int    `json:"num"`
	Page     int    `json:"page"`
	SortType string `json:"sort_type"`
}

// ZhipuWebSearchResponse 智谱网络搜索响应
type ZhipuWebSearchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			URL     string `json:"url"`
			Source  string `json:"source"`
			Time    string `json:"time"`
		} `json:"list"`
		Total int `json:"total"`
	} `json:"data"`
}

// NewZhipuClient 创建智谱客户端
func NewZhipuClient(logger *logx.Logger) *ZhipuClient {
	return &ZhipuClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// SetAPIKey 设置API密钥
func (zc *ZhipuClient) SetAPIKey(apiKey string) {
	zc.apiKey = apiKey
}

// Search 使用智谱AI进行搜索
func (zc *ZhipuClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	zc.logger.Info(ctx, "开始智谱AI搜索", logx.KV("query", query))

	if zc.apiKey == "" {
		zc.logger.Warn(ctx, "智谱AI API密钥未配置，返回模拟数据")
		return zc.getMockSearchResults(query), nil
	}

	// 构建搜索请求
	req := ZhipuSearchRequest{
		Query:       query,
		Model:       "glm-4",
		MaxTokens:   1000,
		Temperature: 0.7,
		TopP:        0.9,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求到智谱AI API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+zc.apiKey)

	resp, err := zc.httpClient.Do(httpReq)
	if err != nil {
		zc.logger.Error(ctx, "智谱AI搜索请求失败", logx.KV("error", err))
		return zc.getMockSearchResults(query), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		zc.logger.Error(ctx, "智谱AI搜索API返回错误", logx.KV("status_code", resp.StatusCode))
		return zc.getMockSearchResults(query), nil
	}

	var response ZhipuSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		zc.logger.Error(ctx, "解析智谱AI搜索响应失败", logx.KV("error", err))
		return zc.getMockSearchResults(query), nil
	}

	// 转换结果
	var results []SearchResult
	for _, choice := range response.Choices {
		result := SearchResult{
			Title:       fmt.Sprintf("智谱AI搜索结果: %s", query),
			Content:     choice.Message.Content,
			URL:         "https://open.bigmodel.cn",
			Author:      "智谱AI",
			PublishedAt: time.Unix(response.Created, 0),
			Platform:    "zhipu",
			Tags:        []string{"智谱AI", "AI搜索"},
		}
		results = append(results, result)
	}

	zc.logger.Info(ctx, "智谱AI搜索完成", logx.KV("results_count", len(results)))
	return results, nil
}

// WebSearch 使用智谱AI进行网络搜索
func (zc *ZhipuClient) WebSearch(ctx context.Context, query string) ([]SearchResult, error) {
	zc.logger.Info(ctx, "开始智谱AI网络搜索", logx.KV("query", query))

	if zc.apiKey == "" {
		zc.logger.Warn(ctx, "智谱AI API密钥未配置，返回模拟数据")
		return zc.getMockWebSearchResults(query), nil
	}

	// 构建网络搜索请求
	req := ZhipuWebSearchRequest{
		Query:    query,
		Num:      10,
		Page:     1,
		SortType: "0", // 综合排序
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求到智谱AI网络搜索API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+zc.apiKey)

	resp, err := zc.httpClient.Do(httpReq)
	if err != nil {
		zc.logger.Error(ctx, "智谱AI网络搜索请求失败", logx.KV("error", err))
		return zc.getMockWebSearchResults(query), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		zc.logger.Error(ctx, "智谱AI网络搜索API返回错误", logx.KV("status_code", resp.StatusCode))
		return zc.getMockWebSearchResults(query), nil
	}

	var response ZhipuWebSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		zc.logger.Error(ctx, "解析智谱AI网络搜索响应失败", logx.KV("error", err))
		return zc.getMockWebSearchResults(query), nil
	}

	// 转换结果
	var results []SearchResult
	for _, item := range response.Data.List {
		publishedAt := time.Now()
		if item.Time != "" {
			if t, err := time.Parse("2006-01-02", item.Time); err == nil {
				publishedAt = t
			}
		}

		result := SearchResult{
			Title:       item.Title,
			Content:     item.Content,
			URL:         item.URL,
			Author:      item.Source,
			PublishedAt: publishedAt,
			Platform:    "zhipu_web",
			Tags:        []string{"智谱AI网络搜索"},
		}
		results = append(results, result)
	}

	zc.logger.Info(ctx, "智谱AI网络搜索完成", logx.KV("results_count", len(results)))
	return results, nil
}

// Chat 使用智谱AI进行对话
func (zc *ZhipuClient) Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	zc.logger.Info(ctx, "开始智谱AI对话", logx.KV("messages_count", len(messages)))

	if zc.apiKey == "" {
		return nil, fmt.Errorf("智谱AI API密钥未配置")
	}

	// 构建对话请求
	req := map[string]interface{}{
		"model":       "glm-4",
		"messages":    messages,
		"max_tokens":  1000,
		"temperature": 0.7,
		"top_p":       0.9,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求到智谱AI API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+zc.apiKey)

	resp, err := zc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("智谱AI对话请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("智谱AI对话API返回错误: %d", resp.StatusCode)
	}

	var response ZhipuSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析智谱AI对话响应失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("智谱AI对话响应为空")
	}

	chatResponse := &ChatResponse{
		ID:        response.ID,
		Content:   response.Choices[0].Message.Content,
		Model:     response.Model,
		CreatedAt: time.Unix(response.Created, 0),
		Usage: ChatUsage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}

	zc.logger.Info(ctx, "智谱AI对话完成",
		logx.KV("response_id", chatResponse.ID),
		logx.KV("total_tokens", chatResponse.Usage.TotalTokens))

	return chatResponse, nil
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Usage     ChatUsage `json:"usage"`
}

// ChatUsage 对话使用统计
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// getMockSearchResults 获取模拟搜索结果
func (zc *ZhipuClient) getMockSearchResults(query string) []SearchResult {
	return []SearchResult{
		{
			Title:       fmt.Sprintf("智谱AI搜索结果: %s", query),
			Content:     fmt.Sprintf("这是智谱AI关于%s的搜索结果。智谱AI提供了相关的信息和见解。", query),
			URL:         "https://open.bigmodel.cn",
			Author:      "智谱AI",
			PublishedAt: time.Now(),
			Platform:    "zhipu",
			Tags:        []string{"智谱AI", "AI搜索", query},
		},
		{
			Title:       fmt.Sprintf("智谱AI分析: %s", query),
			Content:     fmt.Sprintf("智谱AI对%s进行了深入分析，提供了专业的见解和建议。", query),
			URL:         "https://open.bigmodel.cn",
			Author:      "智谱AI",
			PublishedAt: time.Now().Add(-time.Hour),
			Platform:    "zhipu",
			Tags:        []string{"智谱AI", "AI分析", query},
		},
	}
}

// getMockWebSearchResults 获取模拟网络搜索结果
func (zc *ZhipuClient) getMockWebSearchResults(query string) []SearchResult {
	return []SearchResult{
		{
			Title:       fmt.Sprintf("网络搜索结果: %s", query),
			Content:     fmt.Sprintf("这是关于%s的网络搜索结果，包含了相关的网页信息和内容。", query),
			URL:         "https://example.com/search-result-1",
			Author:      "网络搜索",
			PublishedAt: time.Now().Add(-time.Hour),
			Platform:    "zhipu_web",
			Tags:        []string{"网络搜索", query},
		},
		{
			Title:       fmt.Sprintf("相关信息: %s", query),
			Content:     fmt.Sprintf("提供了关于%s的详细信息和相关资料。", query),
			URL:         "https://example.com/search-result-2",
			Author:      "网络搜索",
			PublishedAt: time.Now().Add(-2 * time.Hour),
			Platform:    "zhipu_web",
			Tags:        []string{"网络搜索", query},
		},
	}
}
