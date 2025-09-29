package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// ImageRecognitionTool 图像识别工具
type ImageRecognitionTool struct {
	logger       log.Logger
	config       *config.Config
	openaiClient *OpenAIVisionClient
	baiduClient  *BaiduAIClient
	httpClient   *http.Client
}

// NewImageRecognitionTool 创建图像识别工具
func NewImageRecognitionTool(logger log.Logger, config *config.Config) *ImageRecognitionTool {
	return &ImageRecognitionTool{
		logger:     logger,
		config:     config,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Initialize 初始化图像识别工具
func (irt *ImageRecognitionTool) Initialize(ctx context.Context) error {
	irt.logger.Info(ctx, "初始化图像识别工具")

	// 初始化OpenAI Vision客户端
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey != "" {
		irt.openaiClient = NewOpenAIVisionClient(irt.logger, openaiAPIKey)
		irt.logger.Info(ctx, "OpenAI Vision客户端初始化成功")
	} else {
		irt.logger.Warn(ctx, "OpenAI API Key未设置，跳过OpenAI Vision初始化")
	}

	// 初始化百度AI客户端
	baiduAPIKey := os.Getenv("BAIDU_API_KEY")
	baiduSecretKey := os.Getenv("BAIDU_SECRET_KEY")
	if baiduAPIKey != "" && baiduSecretKey != "" {
		irt.baiduClient = NewBaiduAIClient(irt.logger, baiduAPIKey, baiduSecretKey)
		irt.logger.Info(ctx, "百度AI客户端初始化成功")
	} else {
		irt.logger.Warn(ctx, "百度AI Key未设置，跳过百度AI初始化")
	}

	irt.logger.Info(ctx, "图像识别工具初始化完成")
	return nil
}

// AnalyzeImageComprehensive 综合图片分析
func (irt *ImageRecognitionTool) AnalyzeImageComprehensive(ctx context.Context, imageData []byte, prompt string) (*ImageAnalysisResult, error) {
	irt.logger.Info(ctx, "开始综合图片分析", "image_size", len(imageData), "prompt", prompt)

	// 优先使用OpenAI Vision API
	if irt.openaiClient != nil {
		result, err := irt.openaiClient.AnalyzeImage(ctx, imageData, prompt)
		if err == nil {
			irt.logger.Info(ctx, "OpenAI Vision分析成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "OpenAI Vision分析失败，尝试百度AI", "error", err)
	}

	// 备用百度AI
	if irt.baiduClient != nil {
		result, err := irt.baiduClient.AnalyzeImage(ctx, imageData, prompt)
		if err == nil {
			irt.logger.Info(ctx, "百度AI分析成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "百度AI分析失败", "error", err)
	}

	return nil, fmt.Errorf("所有图像识别服务都不可用")
}

// PerformOCR 执行OCR识别
func (irt *ImageRecognitionTool) PerformOCR(ctx context.Context, imageData []byte) (*OCRResult, error) {
	irt.logger.Info(ctx, "开始OCR识别", "image_size", len(imageData))

	// 优先使用百度AI OCR
	if irt.baiduClient != nil {
		result, err := irt.baiduClient.PerformOCR(ctx, imageData)
		if err == nil {
			irt.logger.Info(ctx, "百度AI OCR识别成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "百度AI OCR识别失败", "error", err)
	}

	// 备用OpenAI Vision OCR
	if irt.openaiClient != nil {
		result, err := irt.openaiClient.PerformOCR(ctx, imageData)
		if err == nil {
			irt.logger.Info(ctx, "OpenAI Vision OCR识别成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "OpenAI Vision OCR识别失败", "error", err)
	}

	return nil, fmt.Errorf("所有OCR服务都不可用")
}

// DetectObjects 检测物体
func (irt *ImageRecognitionTool) DetectObjects(ctx context.Context, imageData []byte) (*ObjectDetectionResult, error) {
	irt.logger.Info(ctx, "开始物体检测", "image_size", len(imageData))

	// 优先使用OpenAI Vision API
	if irt.openaiClient != nil {
		result, err := irt.openaiClient.DetectObjects(ctx, imageData)
		if err == nil {
			irt.logger.Info(ctx, "OpenAI Vision物体检测成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "OpenAI Vision物体检测失败", "error", err)
	}

	// 备用百度AI
	if irt.baiduClient != nil {
		result, err := irt.baiduClient.DetectObjects(ctx, imageData)
		if err == nil {
			irt.logger.Info(ctx, "百度AI物体检测成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "百度AI物体检测失败", "error", err)
	}

	return nil, fmt.Errorf("所有物体检测服务都不可用")
}

// AnalyzeFaces 分析人脸
func (irt *ImageRecognitionTool) AnalyzeFaces(ctx context.Context, imageData []byte) (*FaceAnalysisResult, error) {
	irt.logger.Info(ctx, "开始人脸分析", "image_size", len(imageData))

	// 优先使用百度AI
	if irt.baiduClient != nil {
		result, err := irt.baiduClient.AnalyzeFaces(ctx, imageData)
		if err == nil {
			irt.logger.Info(ctx, "百度AI人脸分析成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "百度AI人脸分析失败", "error", err)
	}

	// 备用OpenAI Vision
	if irt.openaiClient != nil {
		result, err := irt.openaiClient.AnalyzeFaces(ctx, imageData)
		if err == nil {
			irt.logger.Info(ctx, "OpenAI Vision人脸分析成功")
			return result, nil
		}
		irt.logger.Warn(ctx, "OpenAI Vision人脸分析失败", "error", err)
	}

	return nil, fmt.Errorf("所有人脸分析服务都不可用")
}

// GetAvailableServices 获取可用服务
func (irt *ImageRecognitionTool) GetAvailableServices(ctx context.Context) []string {
	var services []string

	if irt.openaiClient != nil {
		services = append(services, "openai_vision")
	}

	if irt.baiduClient != nil {
		services = append(services, "baidu_ai")
	}

	irt.logger.Info(ctx, "可用图像识别服务", "services", services)
	return services
}

// OpenAI Vision 客户端

// OpenAIVisionClient OpenAI Vision客户端
type OpenAIVisionClient struct {
	logger   log.Logger
	apiKey   string
	baseURL  string
	model    string
	httpClient *http.Client
}

// NewOpenAIVisionClient 创建OpenAI Vision客户端
func NewOpenAIVisionClient(logger log.Logger, apiKey string) *OpenAIVisionClient {
	return &OpenAIVisionClient{
		logger:     logger,
		apiKey:     apiKey,
		baseURL:    "https://api.openai.com/v1",
		model:      "gpt-4-vision-preview",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// AnalyzeImage 分析图片
func (ovc *OpenAIVisionClient) AnalyzeImage(ctx context.Context, imageData []byte, prompt string) (*ImageAnalysisResult, error) {
	ovc.logger.Info(ctx, "OpenAI Vision分析图片", "prompt", prompt)

	// 构建请求
	request := map[string]interface{}{
		"model": ovc.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(imageData)),
						},
					},
				},
			},
		},
		"max_tokens": 1000,
	}

	// 发送请求
	response, err := ovc.sendRequest(ctx, "chat/completions", request)
	if err != nil {
		return nil, fmt.Errorf("发送OpenAI Vision请求失败: %w", err)
	}

	// 解析响应
	result := &ImageAnalysisResult{
		Provider: "openai_vision",
		Analysis: ovc.extractContent(response),
		Timestamp: time.Now(),
	}

	return result, nil
}

// PerformOCR 执行OCR
func (ovc *OpenAIVisionClient) PerformOCR(ctx context.Context, imageData []byte) (*OCRResult, error) {
	prompt := "请识别图片中的所有文字内容，包括中文和英文，按顺序输出。"
	
	analysis, err := ovc.AnalyzeImage(ctx, imageData, prompt)
	if err != nil {
		return nil, err
	}

	return &OCRResult{
		Provider: "openai_vision",
		Text:     analysis.Analysis,
		Timestamp: time.Now(),
	}, nil
}

// DetectObjects 检测物体
func (ovc *OpenAIVisionClient) DetectObjects(ctx context.Context, imageData []byte) (*ObjectDetectionResult, error) {
	prompt := "请识别图片中的所有物体，包括人物、动物、物品等，并描述它们的位置和特征。"
	
	analysis, err := ovc.AnalyzeImage(ctx, imageData, prompt)
	if err != nil {
		return nil, err
	}

	return &ObjectDetectionResult{
		Provider: "openai_vision",
		Objects:  []Object{{Name: "检测到的物体", Description: analysis.Analysis}},
		Timestamp: time.Now(),
	}, nil
}

// AnalyzeFaces 分析人脸
func (ovc *OpenAIVisionClient) AnalyzeFaces(ctx context.Context, imageData []byte) (*FaceAnalysisResult, error) {
	prompt := "请分析图片中的人脸，包括人数、年龄、性别、表情等信息。"
	
	analysis, err := ovc.AnalyzeImage(ctx, imageData, prompt)
	if err != nil {
		return nil, err
	}

	return &FaceAnalysisResult{
		Provider: "openai_vision",
		Faces:    []Face{{Analysis: analysis.Analysis}},
		Timestamp: time.Now(),
	}, nil
}

// sendRequest 发送请求
func (ovc *OpenAIVisionClient) sendRequest(ctx context.Context, endpoint string, request map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s", ovc.baseURL, endpoint)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ovc.apiKey)

	resp, err := ovc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return response, nil
}

// extractContent 提取内容
func (ovc *OpenAIVisionClient) extractContent(response map[string]interface{}) string {
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}
	return ""
}

// 百度AI 客户端

// BaiduAIClient 百度AI客户端
type BaiduAIClient struct {
	logger      log.Logger
	apiKey      string
	secretKey   string
	accessToken string
	baseURL     string
	httpClient  *http.Client
}

// NewBaiduAIClient 创建百度AI客户端
func NewBaiduAIClient(logger log.Logger, apiKey, secretKey string) *BaiduAIClient {
	return &BaiduAIClient{
		logger:     logger,
		apiKey:     apiKey,
		secretKey:  secretKey,
		baseURL:    "https://aip.baidubce.com",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// getAccessToken 获取访问令牌
func (bac *BaiduAIClient) getAccessToken(ctx context.Context) error {
	if bac.accessToken != "" {
		return nil
	}

	url := fmt.Sprintf("https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s", 
		bac.apiKey, bac.secretKey)

	resp, err := bac.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("获取访问令牌失败: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("解析访问令牌响应失败: %w", err)
	}

	if token, ok := result["access_token"].(string); ok {
		bac.accessToken = token
		return nil
	}

	return fmt.Errorf("获取访问令牌失败")
}

// AnalyzeImage 分析图片
func (bac *BaiduAIClient) AnalyzeImage(ctx context.Context, imageData []byte, prompt string) (*ImageAnalysisResult, error) {
	bac.logger.Info(ctx, "百度AI分析图片", "prompt", prompt)

	// 获取访问令牌
	err := bac.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取访问令牌失败: %w", err)
	}

	// 构建请求
	request := map[string]interface{}{
		"image": base64.StdEncoding.EncodeToString(imageData),
	}

	// 发送请求
	response, err := bac.sendRequest(ctx, "rest/2.0/vision/v1/general_basic", request)
	if err != nil {
		return nil, fmt.Errorf("发送百度AI请求失败: %w", err)
	}

	// 解析响应
	result := &ImageAnalysisResult{
		Provider: "baidu_ai",
		Analysis: bac.extractAnalysis(response),
		Timestamp: time.Now(),
	}

	return result, nil
}

// PerformOCR 执行OCR
func (bac *BaiduAIClient) PerformOCR(ctx context.Context, imageData []byte) (*OCRResult, error) {
	bac.logger.Info(ctx, "百度AI OCR识别")

	// 获取访问令牌
	err := bac.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取访问令牌失败: %w", err)
	}

	// 构建请求
	request := map[string]interface{}{
		"image": base64.StdEncoding.EncodeToString(imageData),
	}

	// 发送请求
	response, err := bac.sendRequest(ctx, "rest/2.0/vision/v1/general_basic", request)
	if err != nil {
		return nil, fmt.Errorf("发送OCR请求失败: %w", err)
	}

	// 解析响应
	text := bac.extractOCRText(response)
	result := &OCRResult{
		Provider: "baidu_ai",
		Text:     text,
		Timestamp: time.Now(),
	}

	return result, nil
}

// DetectObjects 检测物体
func (bac *BaiduAIClient) DetectObjects(ctx context.Context, imageData []byte) (*ObjectDetectionResult, error) {
	bac.logger.Info(ctx, "百度AI物体检测")

	// 获取访问令牌
	err := bac.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取访问令牌失败: %w", err)
	}

	// 构建请求
	request := map[string]interface{}{
		"image": base64.StdEncoding.EncodeToString(imageData),
	}

	// 发送请求
	response, err := bac.sendRequest(ctx, "rest/2.0/vision/v1/object_detect", request)
	if err != nil {
		return nil, fmt.Errorf("发送物体检测请求失败: %w", err)
	}

	// 解析响应
	objects := bac.extractObjects(response)
	result := &ObjectDetectionResult{
		Provider: "baidu_ai",
		Objects:  objects,
		Timestamp: time.Now(),
	}

	return result, nil
}

// AnalyzeFaces 分析人脸
func (bac *BaiduAIClient) AnalyzeFaces(ctx context.Context, imageData []byte) (*FaceAnalysisResult, error) {
	bac.logger.Info(ctx, "百度AI人脸分析")

	// 获取访问令牌
	err := bac.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取访问令牌失败: %w", err)
	}

	// 构建请求
	request := map[string]interface{}{
		"image": base64.StdEncoding.EncodeToString(imageData),
	}

	// 发送请求
	response, err := bac.sendRequest(ctx, "rest/2.0/vision/v1/face", request)
	if err != nil {
		return nil, fmt.Errorf("发送人脸分析请求失败: %w", err)
	}

	// 解析响应
	faces := bac.extractFaces(response)
	result := &FaceAnalysisResult{
		Provider: "baidu_ai",
		Faces:    faces,
		Timestamp: time.Now(),
	}

	return result, nil
}

// sendRequest 发送请求
func (bac *BaiduAIClient) sendRequest(ctx context.Context, endpoint string, request map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s?access_token=%s", bac.baseURL, endpoint, bac.accessToken)
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := bac.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return response, nil
}

// extractAnalysis 提取分析结果
func (bac *BaiduAIClient) extractAnalysis(response map[string]interface{}) string {
	// 简化的提取逻辑
	if result, ok := response["result"].(map[string]interface{}); ok {
		if analysis, ok := result["analysis"].(string); ok {
			return analysis
		}
	}
	return "分析结果提取失败"
}

// extractOCRText 提取OCR文本
func (bac *BaiduAIClient) extractOCRText(response map[string]interface{}) string {
	if wordsResult, ok := response["words_result"].([]interface{}); ok {
		var text string
		for _, word := range wordsResult {
			if wordMap, ok := word.(map[string]interface{}); ok {
				if wordStr, ok := wordMap["words"].(string); ok {
					text += wordStr + "\n"
				}
			}
		}
		return text
	}
	return ""
}

// extractObjects 提取物体信息
func (bac *BaiduAIClient) extractObjects(response map[string]interface{}) []Object {
	var objects []Object
	
	if result, ok := response["result"].([]interface{}); ok {
		for _, item := range result {
			if itemMap, ok := item.(map[string]interface{}); ok {
				object := Object{
					Name:        getString(itemMap, "name"),
					Description: getString(itemMap, "description"),
					Confidence:  getFloat64(itemMap, "confidence"),
				}
				objects = append(objects, object)
			}
		}
	}
	
	return objects
}

// extractFaces 提取人脸信息
func (bac *BaiduAIClient) extractFaces(response map[string]interface{}) []Face {
	var faces []Face
	
	if result, ok := response["result"].([]interface{}); ok {
		for _, item := range result {
			if itemMap, ok := item.(map[string]interface{}); ok {
				face := Face{
					Analysis: getString(itemMap, "analysis"),
					Age:      getInt(itemMap, "age"),
					Gender:   getString(itemMap, "gender"),
				}
				faces = append(faces, face)
			}
		}
	}
	
	return faces
}

// 数据结构

// ImageAnalysisResult 图像分析结果
type ImageAnalysisResult struct {
	Provider  string    `json:"provider"`
	Analysis  string    `json:"analysis"`
	Timestamp time.Time `json:"timestamp"`
}

// OCRResult OCR识别结果
type OCRResult struct {
	Provider  string    `json:"provider"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// ObjectDetectionResult 物体检测结果
type ObjectDetectionResult struct {
	Provider  string    `json:"provider"`
	Objects   []Object  `json:"objects"`
	Timestamp time.Time `json:"timestamp"`
}

// Object 物体信息
type Object struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
}

// FaceAnalysisResult 人脸分析结果
type FaceAnalysisResult struct {
	Provider  string    `json:"provider"`
	Faces     []Face    `json:"faces"`
	Timestamp time.Time `json:"timestamp"`
}

// Face 人脸信息
type Face struct {
	Analysis string `json:"analysis"`
	Age      int    `json:"age"`
	Gender   string `json:"gender"`
}

// 辅助函数

func getString(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if value, ok := data[key].(float64); ok {
		return int(value)
	}
	return 0
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if value, ok := data[key].(float64); ok {
		return value
	}
	return 0.0
}
