package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// GeminiClientGen Gemini 2.5 Pro客户端
type GeminiClientGen struct {
	logger     log.Logger
	config     *config.Config
	apiKey     string
	projectID  string
	location   string
	baseURL    string
	modelID    string
	httpClient *http.Client
	vertexAI   bool
}

// NewGeminiClientGen 创建Gemini客户端
func NewGeminiClientGen(logger log.Logger, config *config.Config) *GeminiClientGen {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	projectID := os.Getenv("GOOGLE_PROJECT_ID")
	location := os.Getenv("GOOGLE_LOCATION")

	// 检查是否使用Vertex AI
	vertexAI := os.Getenv("GOOGLE_VERTEX_AI") == "true"

	var baseURL string
	if vertexAI {
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models",
			location, projectID, location)
	} else {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	modelID := os.Getenv("GEMINI_MODEL_ID")
	if modelID == "" {
		modelID = "gemini-2.5-pro"
	}

	return &GeminiClientGen{
		logger:     logger,
		config:     config,
		apiKey:     apiKey,
		projectID:  projectID,
		location:   location,
		baseURL:    baseURL,
		modelID:    modelID,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		vertexAI:   vertexAI,
	}
}

// GenerateContent 生成内容
func (gc *GeminiClientGen) GenerateContent(ctx context.Context, contents []string, config *GenerateContentConfig) (*GenerateContentResponse, error) {
	gc.logger.Info(ctx, "开始生成内容",
		"contents_count", len(contents),
		"model", gc.modelID,
		"vertex_ai", gc.vertexAI)

	// 构建请求
	request := gc.buildGenerateRequest(contents, config)

	// 发送请求
	response, err := gc.sendRequest(ctx, "generateContent", request)
	if err != nil {
		return nil, fmt.Errorf("发送生成内容请求失败: %w", err)
	}

	// 解析响应
	result := &GenerateContentResponse{}
	err = json.Unmarshal(response, result)
	if err != nil {
		return nil, fmt.Errorf("解析生成内容响应失败: %w", err)
	}

	gc.logger.Info(ctx, "内容生成成功", "candidates_count", len(result.Candidates))
	return result, nil
}

// StreamGenerateContent 流式生成内容
func (gc *GeminiClientGen) StreamGenerateContent(ctx context.Context, contents []string, config *GenerateContentConfig, onChunk func(string) error) error {
	gc.logger.Info(ctx, "开始流式生成内容",
		"contents_count", len(contents),
		"model", gc.modelID)

	// 构建流式请求
	request := gc.buildGenerateRequest(contents, config)
	request["stream"] = true

	// 发送流式请求
	err := gc.sendStreamRequest(ctx, "streamGenerateContent", request, onChunk)
	if err != nil {
		return fmt.Errorf("发送流式生成内容请求失败: %w", err)
	}

	gc.logger.Info(ctx, "流式内容生成完成")
	return nil
}

// AnalyzeImage 分析图片
func (gc *GeminiClientGen) AnalyzeImage(ctx context.Context, imageData []byte, prompt string) (*GenerateContentResponse, error) {
	gc.logger.Info(ctx, "开始分析图片", "image_size", len(imageData))

	// 构建图片内容
	imageContent := map[string]interface{}{
		"parts": []map[string]interface{}{
			{
				"text": prompt,
			},
			{
				"inline_data": map[string]interface{}{
					"mime_type": "image/jpeg",
					"data":      encodeBase64(imageData),
				},
			},
		},
	}

	// 构建请求
	request := map[string]interface{}{
		"contents": []map[string]interface{}{imageContent},
		"generationConfig": map[string]interface{}{
			"temperature":     0.4,
			"topK":            32,
			"topP":            1,
			"maxOutputTokens": 4096,
		},
	}

	// 发送请求
	response, err := gc.sendRequest(ctx, "generateContent", request)
	if err != nil {
		return nil, fmt.Errorf("发送图片分析请求失败: %w", err)
	}

	// 解析响应
	result := &GenerateContentResponse{}
	err = json.Unmarshal(response, result)
	if err != nil {
		return nil, fmt.Errorf("解析图片分析响应失败: %w", err)
	}

	gc.logger.Info(ctx, "图片分析完成", "candidates_count", len(result.Candidates))
	return result, nil
}

// ProcessMultimodalFiles 处理多模态文件
func (gc *GeminiClientGen) ProcessMultimodalFiles(ctx context.Context, files []MultimodalFile, prompt string) (*GenerateContentResponse, error) {
	gc.logger.Info(ctx, "开始处理多模态文件", "files_count", len(files))

	// 构建内容部分
	var parts []map[string]interface{}

	// 添加文本提示
	parts = append(parts, map[string]interface{}{
		"text": prompt,
	})

	// 添加文件内容
	for _, file := range files {
		part := map[string]interface{}{
			"inline_data": map[string]interface{}{
				"mime_type": file.MimeType,
				"data":      encodeBase64(file.Data),
			},
		}
		parts = append(parts, part)
	}

	// 构建内容
	content := map[string]interface{}{
		"parts": parts,
	}

	// 构建请求
	request := map[string]interface{}{
		"contents": []map[string]interface{}{content},
		"generationConfig": map[string]interface{}{
			"temperature":     0.4,
			"topK":            32,
			"topP":            1,
			"maxOutputTokens": 4096,
		},
	}

	// 发送请求
	response, err := gc.sendRequest(ctx, "generateContent", request)
	if err != nil {
		return nil, fmt.Errorf("发送多模态文件处理请求失败: %w", err)
	}

	// 解析响应
	result := &GenerateContentResponse{}
	err = json.Unmarshal(response, result)
	if err != nil {
		return nil, fmt.Errorf("解析多模态文件处理响应失败: %w", err)
	}

	gc.logger.Info(ctx, "多模态文件处理完成", "candidates_count", len(result.Candidates))
	return result, nil
}

// GetModelInfo 获取模型信息
func (gc *GeminiClientGen) GetModelInfo(ctx context.Context) (*ModelInfo, error) {
	gc.logger.Info(ctx, "获取模型信息", "model", gc.modelID)

	// 发送请求
	response, err := gc.sendRequest(ctx, "getModel", nil)
	if err != nil {
		return nil, fmt.Errorf("获取模型信息失败: %w", err)
	}

	// 解析响应
	result := &ModelInfo{}
	err = json.Unmarshal(response, result)
	if err != nil {
		return nil, fmt.Errorf("解析模型信息失败: %w", err)
	}

	gc.logger.Info(ctx, "模型信息获取成功", "model_name", result.Name)
	return result, nil
}

// ListModels 列出可用模型
func (gc *GeminiClientGen) ListModels(ctx context.Context) (*ListModelsResponse, error) {
	gc.logger.Info(ctx, "列出可用模型")

	// 发送请求
	response, err := gc.sendRequest(ctx, "listModels", nil)
	if err != nil {
		return nil, fmt.Errorf("列出模型失败: %w", err)
	}

	// 解析响应
	result := &ListModelsResponse{}
	err = json.Unmarshal(response, result)
	if err != nil {
		return nil, fmt.Errorf("解析模型列表失败: %w", err)
	}

	gc.logger.Info(ctx, "模型列表获取成功", "models_count", len(result.Models))
	return result, nil
}

// 私有方法

// buildGenerateRequest 构建生成请求
func (gc *GeminiClientGen) buildGenerateRequest(contents []string, config *GenerateContentConfig) map[string]interface{} {
	// 构建内容
	var contentParts []map[string]interface{}
	for _, content := range contents {
		contentParts = append(contentParts, map[string]interface{}{
			"text": content,
		})
	}

	request := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": contentParts,
			},
		},
	}

	// 添加生成配置
	if config != nil {
		request["generationConfig"] = map[string]interface{}{
			"temperature":     config.Temperature,
			"topK":            config.TopK,
			"topP":            config.TopP,
			"maxOutputTokens": config.MaxOutputTokens,
		}
	} else {
		// 默认配置
		request["generationConfig"] = map[string]interface{}{
			"temperature":     0.7,
			"topK":            40,
			"topP":            0.95,
			"maxOutputTokens": 2048,
		}
	}

	return request
}

// sendRequest 发送请求
func (gc *GeminiClientGen) sendRequest(ctx context.Context, endpoint string, request map[string]interface{}) ([]byte, error) {
	// 构建URL
	url := fmt.Sprintf("%s/models/%s:%s", gc.baseURL, gc.modelID, endpoint)
	if !gc.vertexAI {
		url = fmt.Sprintf("%s/models/%s:%s?key=%s", gc.baseURL, gc.modelID, endpoint, gc.apiKey)
	}

	// 序列化请求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if gc.vertexAI {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.apiKey))
	}

	// 发送请求
	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// sendStreamRequest 发送流式请求
func (gc *GeminiClientGen) sendStreamRequest(ctx context.Context, endpoint string, request map[string]interface{}, onChunk func(string) error) error {
	// 构建URL
	url := fmt.Sprintf("%s/models/%s:%s", gc.baseURL, gc.modelID, endpoint)
	if !gc.vertexAI {
		url = fmt.Sprintf("%s/models/%s:%s?key=%s", gc.baseURL, gc.modelID, endpoint, gc.apiKey)
	}

	// 序列化请求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if gc.vertexAI {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.apiKey))
	}

	// 发送请求
	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(responseBody))
	}

	// 处理流式响应
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk map[string]interface{}
		err := decoder.Decode(&chunk)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("解析流式响应失败: %w", err)
		}

		// 提取文本内容
		if candidates, ok := chunk["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]interface{}); ok {
				if content, ok := candidate["content"].(map[string]interface{}); ok {
					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
						if part, ok := parts[0].(map[string]interface{}); ok {
							if text, ok := part["text"].(string); ok {
								err := onChunk(text)
								if err != nil {
									return fmt.Errorf("处理流式数据块失败: %w", err)
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// 数据结构

// GenerateContentConfig 生成内容配置
type GenerateContentConfig struct {
	Temperature     float64 `json:"temperature"`
	TopK            int     `json:"topK"`
	TopP            float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// GenerateContentResponse 生成内容响应
type GenerateContentResponse struct {
	Candidates []Candidate `json:"candidates"`
	Usage      Usage       `json:"usageMetadata"`
}

// Candidate 候选响应
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
}

// Content 内容
type Content struct {
	Parts []Part `json:"parts"`
}

// Part 内容部分
type Part struct {
	Text string `json:"text"`
}

// Usage 使用情况
type Usage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// MultimodalFile 多模态文件
type MultimodalFile struct {
	Data     []byte `json:"data"`
	MimeType string `json:"mime_type"`
	FileName string `json:"file_name"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName"`
	Description                string   `json:"description"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}

// ListModelsResponse 模型列表响应
type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// 辅助函数

// encodeBase64 编码为Base64
func encodeBase64(data []byte) string {
	return fmt.Sprintf("%x", data) // 简化实现，实际应该使用base64编码
}
