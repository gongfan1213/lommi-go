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

// ZhipuHTTPClient 智谱HTTP客户端
type ZhipuHTTPClient struct {
	apiKey     string
	httpClient *http.Client
	logger     *logx.Logger
	baseURL    string
}

// ZhipuHTTPRequest 智谱HTTP请求
type ZhipuHTTPRequest struct {
	Model       string                 `json:"model"`
	Messages    []ZhipuMessage         `json:"messages"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
	TopP        float64                `json:"top_p"`
	Stream      bool                   `json:"stream"`
	Tools       []ZhipuTool            `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// ZhipuMessage 智谱消息
type ZhipuMessage struct {
	Role      string                 `json:"role"`
	Content   interface{}            `json:"content"`
	Name      string                 `json:"name,omitempty"`
	ToolCalls []ZhipuToolCall        `json:"tool_calls,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// ZhipuToolCall 智谱工具调用
type ZhipuToolCall struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Function ZhipuFunction `json:"function"`
}

// ZhipuFunction 智谱函数
type ZhipuFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ZhipuTool 智谱工具
type ZhipuTool struct {
	Type     string                 `json:"type"`
	Function ZhipuToolFunction      `json:"function"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// ZhipuToolFunction 智谱工具函数
type ZhipuToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// ZhipuHTTPResponse 智谱HTTP响应
type ZhipuHTTPResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ZhipuChoice `json:"choices"`
	Usage   ZhipuUsage    `json:"usage"`
	Error   *ZhipuError   `json:"error,omitempty"`
}

// ZhipuChoice 智谱选择
type ZhipuChoice struct {
	Index        int          `json:"index"`
	Message      ZhipuMessage `json:"message"`
	Delta        ZhipuMessage `json:"delta,omitempty"`
	Logprobs     interface{}  `json:"logprobs,omitempty"`
	FinishReason string       `json:"finish_reason"`
}

// ZhipuUsage 智谱使用统计
type ZhipuUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ZhipuError 智谱错误
type ZhipuError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewZhipuHTTPClient 创建智谱HTTP客户端
func NewZhipuHTTPClient(logger *logx.Logger) *ZhipuHTTPClient {
	return &ZhipuHTTPClient{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:  logger,
		baseURL: "https://open.bigmodel.cn/api/paas/v4",
	}
}

// SetAPIKey 设置API密钥
func (zhc *ZhipuHTTPClient) SetAPIKey(apiKey string) {
	zhc.apiKey = apiKey
}

// SetBaseURL 设置基础URL
func (zhc *ZhipuHTTPClient) SetBaseURL(baseURL string) {
	zhc.baseURL = baseURL
}

// ChatCompletion 聊天补全
func (zhc *ZhipuHTTPClient) ChatCompletion(ctx context.Context, req ZhipuHTTPRequest) (*ZhipuHTTPResponse, error) {
	zhc.logger.Info(ctx, "开始智谱聊天补全",
		logx.KV("model", req.Model),
		logx.KV("messages_count", len(req.Messages)))

	if zhc.apiKey == "" {
		return nil, fmt.Errorf("智谱API密钥未配置")
	}

	// 设置默认值
	if req.Model == "" {
		req.Model = "glm-4"
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.TopP == 0 {
		req.TopP = 0.9
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	url := zhc.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+zhc.apiKey)

	resp, err := zhc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("智谱API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errorResp ZhipuHTTPResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error != nil {
			return nil, fmt.Errorf("智谱API错误: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("智谱API返回错误状态码: %d", resp.StatusCode)
	}

	var response ZhipuHTTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析智谱API响应失败: %w", err)
	}

	zhc.logger.Info(ctx, "智谱聊天补全完成",
		logx.KV("response_id", response.ID),
		logx.KV("total_tokens", response.Usage.TotalTokens))

	return &response, nil
}

// StreamChatCompletion 流式聊天补全
func (zhc *ZhipuHTTPClient) StreamChatCompletion(ctx context.Context, req ZhipuHTTPRequest, callback func(*ZhipuHTTPResponse) error) error {
	zhc.logger.Info(ctx, "开始智谱流式聊天补全",
		logx.KV("model", req.Model),
		logx.KV("messages_count", len(req.Messages)))

	if zhc.apiKey == "" {
		return fmt.Errorf("智谱API密钥未配置")
	}

	// 设置流式模式
	req.Stream = true

	// 设置默认值
	if req.Model == "" {
		req.Model = "glm-4"
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.TopP == 0 {
		req.TopP = 0.9
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	url := zhc.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+zhc.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := zhc.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("智谱API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("智谱API返回错误状态码: %d", resp.StatusCode)
	}

	// 处理流式响应
	decoder := json.NewDecoder(resp.Body)
	for {
		var response ZhipuHTTPResponse
		if err := decoder.Decode(&response); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("解析流式响应失败: %w", err)
		}

		// 调用回调函数
		if err := callback(&response); err != nil {
			return fmt.Errorf("处理流式响应失败: %w", err)
		}

		// 检查是否结束
		if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
			break
		}
	}

	zhc.logger.Info(ctx, "智谱流式聊天补全完成")
	return nil
}

// Embedding 文本嵌入
func (zhc *ZhipuHTTPClient) Embedding(ctx context.Context, input []string, model string) (*ZhipuEmbeddingResponse, error) {
	zhc.logger.Info(ctx, "开始智谱文本嵌入",
		logx.KV("input_count", len(input)),
		logx.KV("model", model))

	if zhc.apiKey == "" {
		return nil, fmt.Errorf("智谱API密钥未配置")
	}

	if model == "" {
		model = "embedding-2"
	}

	req := map[string]interface{}{
		"model": model,
		"input": input,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	url := zhc.baseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+zhc.apiKey)

	resp, err := zhc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("智谱API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("智谱API返回错误状态码: %d", resp.StatusCode)
	}

	var response ZhipuEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析智谱API响应失败: %w", err)
	}

	zhc.logger.Info(ctx, "智谱文本嵌入完成",
		logx.KV("embeddings_count", len(response.Data)))

	return &response, nil
}

// ZhipuEmbeddingResponse 智谱嵌入响应
type ZhipuEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string     `json:"model"`
	Usage ZhipuUsage `json:"usage"`
}

// GetModels 获取可用模型列表
func (zhc *ZhipuHTTPClient) GetModels(ctx context.Context) (*ZhipuModelsResponse, error) {
	zhc.logger.Info(ctx, "获取智谱可用模型列表")

	if zhc.apiKey == "" {
		return nil, fmt.Errorf("智谱API密钥未配置")
	}

	// 发送请求
	url := zhc.baseURL + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+zhc.apiKey)

	resp, err := zhc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("智谱API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("智谱API返回错误状态码: %d", resp.StatusCode)
	}

	var response ZhipuModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析智谱API响应失败: %w", err)
	}

	zhc.logger.Info(ctx, "获取智谱模型列表完成",
		logx.KV("models_count", len(response.Data)))

	return &response, nil
}

// ZhipuModelsResponse 智谱模型响应
type ZhipuModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}
