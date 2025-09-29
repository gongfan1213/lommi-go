package multimodal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"strings"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// ImageRecognitionTool 图像识别工具
type ImageRecognitionTool struct {
	logger      *logx.Logger
	openaiKey   string
	baiduKey    string
	baiduSecret string
	httpClient  *http.Client
}

// NewImageRecognitionTool 创建图像识别工具
func NewImageRecognitionTool(logger *logx.Logger) *ImageRecognitionTool {
	return &ImageRecognitionTool{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// AnalyzeImageComprehensive 综合图片分析
func (irt *ImageRecognitionTool) AnalyzeImageComprehensive(ctx context.Context, req ComprehensiveAnalysisRequest) (*ComprehensiveAnalysisResponse, error) {
	irt.logger.Info(ctx, "开始综合图片分析",
		logx.KV("analysis_types", req.AnalysisTypes),
		logx.KV("image_size", len(req.ImageData)))

	// 预处理图片
	processedImage, err := irt.preprocessImage(req.ImageData)
	if err != nil {
		return nil, fmt.Errorf("预处理图片失败: %w", err)
	}

	response := &ComprehensiveAnalysisResponse{
		ImageInfo:     irt.getImageInfo(processedImage),
		Timestamp:     time.Now(),
		AnalysisTypes: req.AnalysisTypes,
		Results:       make(map[string]interface{}),
	}

	// 并行执行各种分析
	analysisTasks := make(map[string]func() (interface{}, error))

	if irt.containsAnalysisType(req.AnalysisTypes, "description") {
		analysisTasks["description"] = func() (interface{}, error) {
			return irt.analyzeWithVisionAPI(ctx, processedImage, req.DescriptionPrompt)
		}
	}

	if irt.containsAnalysisType(req.AnalysisTypes, "ocr") {
		analysisTasks["ocr"] = func() (interface{}, error) {
			return irt.extractTextOCR(ctx, processedImage)
		}
	}

	if irt.containsAnalysisType(req.AnalysisTypes, "objects") {
		analysisTasks["objects"] = func() (interface{}, error) {
			return irt.detectObjects(ctx, processedImage)
		}
	}

	if irt.containsAnalysisType(req.AnalysisTypes, "faces") {
		analysisTasks["faces"] = func() (interface{}, error) {
			return irt.analyzeFaces(ctx, processedImage)
		}
	}

	// 执行分析任务
	for analysisType, task := range analysisTasks {
		result, err := task()
		if err != nil {
			irt.logger.Error(ctx, "分析失败",
				logx.KV("analysis_type", analysisType),
				logx.KV("error", err))
			response.Results[analysisType] = map[string]interface{}{
				"error": err.Error(),
			}
		} else {
			response.Results[analysisType] = result
		}
	}

	irt.logger.Info(ctx, "综合图片分析完成",
		logx.KV("completed_analyses", len(response.Results)))

	return response, nil
}

// ComprehensiveAnalysisRequest 综合分析请求
type ComprehensiveAnalysisRequest struct {
	ImageData         []byte   `json:"image_data"`
	AnalysisTypes     []string `json:"analysis_types"`
	DescriptionPrompt string   `json:"description_prompt"`
}

// ComprehensiveAnalysisResponse 综合分析响应
type ComprehensiveAnalysisResponse struct {
	ImageInfo     ImageInfo              `json:"image_info"`
	Timestamp     time.Time              `json:"timestamp"`
	AnalysisTypes []string               `json:"analysis_types"`
	Results       map[string]interface{} `json:"results"`
}

// preprocessImage 预处理图片
func (irt *ImageRecognitionTool) preprocessImage(imageData []byte) ([]byte, error) {
	// 检查图片格式并转换
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	// 如果已经是JPEG格式，直接返回
	if format == "jpeg" {
		return imageData, nil
	}

	// 转换为JPEG格式
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, fmt.Errorf("转换图片格式失败: %w", err)
	}

	return buf.Bytes(), nil
}

// getImageInfo 获取图片信息
func (irt *ImageRecognitionTool) getImageInfo(imageData []byte) ImageInfo {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return ImageInfo{
			Size:   len(imageData),
			Format: "unknown",
		}
	}

	bounds := img.Bounds()
	return ImageInfo{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Size:   len(imageData),
		Format: "jpeg",
	}
}

// analyzeWithVisionAPI 使用Vision API分析
func (irt *ImageRecognitionTool) analyzeWithVisionAPI(ctx context.Context, imageData []byte, prompt string) (map[string]interface{}, error) {
	if irt.openaiKey == "" {
		return map[string]interface{}{
			"description": "Vision API未配置，无法进行图片描述分析",
		}, nil
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	requestBody := map[string]interface{}{
		"model": "gpt-4-vision-preview",
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
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
						},
					},
				},
			},
		},
		"max_tokens": 500,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+irt.openaiKey)

	resp, err := irt.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Vision API请求失败: %d", resp.StatusCode)
	}

	// 提取描述文本
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return map[string]interface{}{
						"description": content,
						"model":       "gpt-4-vision-preview",
					}, nil
				}
			}
		}
	}

	return map[string]interface{}{
		"description": "无法提取图片描述",
	}, nil
}

// extractTextOCR OCR文字提取
func (irt *ImageRecognitionTool) extractTextOCR(ctx context.Context, imageData []byte) (map[string]interface{}, error) {
	if irt.baiduKey == "" || irt.baiduSecret == "" {
		return map[string]interface{}{
			"ocr_text": "百度OCR未配置，无法进行文字识别",
		}, nil
	}

	// 获取百度访问令牌
	accessToken, err := irt.getBaiduAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// 调用百度OCR API
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	requestBody := map[string]string{
		"image":        base64Image,
		"access_token": accessToken,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://aip.baidubce.com/rest/2.0/ocr/v1/general_basic", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := irt.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	// 提取OCR文本
	var ocrText strings.Builder
	if wordsResult, ok := response["words_result"].([]interface{}); ok {
		for _, word := range wordsResult {
			if wordMap, ok := word.(map[string]interface{}); ok {
				if words, ok := wordMap["words"].(string); ok {
					ocrText.WriteString(words + "\n")
				}
			}
		}
	}

	return map[string]interface{}{
		"ocr_text": strings.TrimSpace(ocrText.String()),
		"provider": "baidu_ocr",
	}, nil
}

// detectObjects 物体检测
func (irt *ImageRecognitionTool) detectObjects(ctx context.Context, imageData []byte) (map[string]interface{}, error) {
	if irt.baiduKey == "" || irt.baiduSecret == "" {
		return map[string]interface{}{
			"objects": []DetectedObject{},
			"note":    "百度物体检测未配置",
		}, nil
	}

	// 获取百度访问令牌
	accessToken, err := irt.getBaiduAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// 调用百度物体检测API
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	requestBody := map[string]string{
		"image":        base64Image,
		"access_token": accessToken,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://aip.baidubce.com/rest/2.0/image-classify/v1/object_detect", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := irt.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	// 提取物体信息
	var objects []DetectedObject
	if result, ok := response["result"].([]interface{}); ok {
		for _, item := range result {
			if itemMap, ok := item.(map[string]interface{}); ok {
				obj := DetectedObject{
					Name:       irt.getString(itemMap, "name"),
					Confidence: irt.getFloat64(itemMap, "score"),
				}

				// 解析边界框
				if location, ok := itemMap["location"].(map[string]interface{}); ok {
					obj.BoundingBox = BoundingBox{
						X:      irt.getInt(location, "left"),
						Y:      irt.getInt(location, "top"),
						Width:  irt.getInt(location, "width"),
						Height: irt.getInt(location, "height"),
					}
				}

				objects = append(objects, obj)
			}
		}
	}

	return map[string]interface{}{
		"objects":  objects,
		"provider": "baidu_object_detect",
	}, nil
}

// analyzeFaces 人脸分析
func (irt *ImageRecognitionTool) analyzeFaces(ctx context.Context, imageData []byte) (map[string]interface{}, error) {
	if irt.baiduKey == "" || irt.baiduSecret == "" {
		return map[string]interface{}{
			"faces": []DetectedFace{},
			"note":  "百度人脸检测未配置",
		}, nil
	}

	// 获取百度访问令牌
	accessToken, err := irt.getBaiduAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	// 调用百度人脸检测API
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	requestBody := map[string]string{
		"image":        base64Image,
		"access_token": accessToken,
		"face_field":   "age,beauty,expression,gender",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://aip.baidubce.com/rest/2.0/face/v3/detect", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := irt.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	// 提取人脸信息
	var faces []DetectedFace
	if result, ok := response["result"].(map[string]interface{}); ok {
		if faceList, ok := result["face_list"].([]interface{}); ok {
			for _, face := range faceList {
				if faceMap, ok := face.(map[string]interface{}); ok {
					detectedFace := DetectedFace{
						Confidence: irt.getFloat64(faceMap, "face_probability"),
						Age:        irt.getInt(faceMap, "age"),
						Gender:     irt.getString(faceMap, "gender"),
						Emotion:    irt.getString(faceMap, "expression"),
					}

					// 解析位置信息
					if location, ok := faceMap["location"].(map[string]interface{}); ok {
						detectedFace.BoundingBox = BoundingBox{
							X:      irt.getInt(location, "left"),
							Y:      irt.getInt(location, "top"),
							Width:  irt.getInt(location, "width"),
							Height: irt.getInt(location, "height"),
						}
					}

					faces = append(faces, detectedFace)
				}
			}
		}
	}

	return map[string]interface{}{
		"faces":    faces,
		"provider": "baidu_face_detect",
	}, nil
}

// getBaiduAccessToken 获取百度访问令牌
func (irt *ImageRecognitionTool) getBaiduAccessToken(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s",
		irt.baiduKey, irt.baiduSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := irt.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if accessToken, ok := response["access_token"].(string); ok {
		return accessToken, nil
	}

	return "", fmt.Errorf("无法获取百度访问令牌")
}

// containsAnalysisType 检查是否包含分析类型
func (irt *ImageRecognitionTool) containsAnalysisType(types []string, targetType string) bool {
	for _, t := range types {
		if t == targetType {
			return true
		}
	}
	return false
}

// 辅助函数
func (irt *ImageRecognitionTool) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func (irt *ImageRecognitionTool) getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0.0
}

func (irt *ImageRecognitionTool) getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}
