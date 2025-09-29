package multimodal

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// Processor 多模态处理器接口
type Processor interface {
	// 检测是否需要多模态处理
	ShouldUseMultimodalProcessing(ctx context.Context, userID, sessionID, instruction string) (bool, error)

	// 处理多模态文件
	ProcessMultimodalFiles(ctx context.Context, req ProcessMultimodalRequest) (*ProcessMultimodalResponse, error)

	// 分析图片
	AnalyzeImage(ctx context.Context, req AnalyzeImageRequest) (*AnalyzeImageResponse, error)

	// 处理文件上传
	ProcessFileUpload(ctx context.Context, file *multipart.FileHeader, userID, sessionID string) (*FileProcessResponse, error)

	// 清理临时文件
	CleanupTempFiles(ctx context.Context, filePaths []string) error
}

// MultimodalProcessor 多模态处理器实现
type MultimodalProcessor struct {
	config       *config.GeminiConfig
	logger       *logx.Logger
	ossClient    OSSClient
	geminiClient GeminiClient
	tempDir      string
}

// OSSClient OSS客户端接口
type OSSClient interface {
	UploadFile(ctx context.Context, filePath, key string) (string, error)
	GetFileURL(ctx context.Context, key string) (string, error)
	DeleteFile(ctx context.Context, key string) error
}

// GeminiClient Gemini客户端接口
type GeminiClient interface {
	AnalyzeImage(ctx context.Context, imageData []byte, prompt string) (*GeminiImageAnalysis, error)
	ProcessMultimodal(ctx context.Context, req GeminiMultimodalRequest) (*GeminiMultimodalResponse, error)
}

// ProcessMultimodalRequest 多模态处理请求
type ProcessMultimodalRequest struct {
	UserID      string `json:"user_id"`
	SessionID   string `json:"session_id"`
	Instruction string `json:"instruction"`
	AgentName   string `json:"agent_name"`
}

// ProcessMultimodalResponse 多模态处理响应
type ProcessMultimodalResponse struct {
	AnalysisResults []FileAnalysisResult `json:"analysis_results"`
	ProcessedFiles  []ProcessedFile      `json:"processed_files"`
	TempFiles       []string             `json:"temp_files"`
}

// FileAnalysisResult 文件分析结果
type FileAnalysisResult struct {
	FileName    string                 `json:"file_name"`
	FileType    string                 `json:"file_type"`
	FileSize    int64                  `json:"file_size"`
	Analysis    map[string]interface{} `json:"analysis"`
	OSSKey      string                 `json:"oss_key"`
	OSSURL      string                 `json:"oss_url"`
	ProcessedAt time.Time              `json:"processed_at"`
}

// ProcessedFile 已处理的文件
type ProcessedFile struct {
	OriginalPath string `json:"original_path"`
	TempPath     string `json:"temp_path"`
	OSSKey       string `json:"oss_key"`
	OSSURL       string `json:"oss_url"`
	FileSize     int64  `json:"file_size"`
	FileType     string `json:"file_type"`
}

// AnalyzeImageRequest 图片分析请求
type AnalyzeImageRequest struct {
	ImageData     []byte   `json:"image_data"`
	Prompt        string   `json:"prompt"`
	AnalysisTypes []string `json:"analysis_types"`
}

// AnalyzeImageResponse 图片分析响应
type AnalyzeImageResponse struct {
	Description   string                 `json:"description"`
	OCRText       string                 `json:"ocr_text"`
	Objects       []DetectedObject       `json:"objects"`
	Faces         []DetectedFace         `json:"faces"`
	ImageInfo     ImageInfo              `json:"image_info"`
	AnalysisTypes []string               `json:"analysis_types"`
	RawAnalysis   map[string]interface{} `json:"raw_analysis"`
	ProcessedAt   time.Time              `json:"processed_at"`
}

// DetectedObject 检测到的物体
type DetectedObject struct {
	Name        string      `json:"name"`
	Confidence  float64     `json:"confidence"`
	BoundingBox BoundingBox `json:"bounding_box"`
}

// DetectedFace 检测到的人脸
type DetectedFace struct {
	Confidence  float64     `json:"confidence"`
	Emotion     string      `json:"emotion"`
	Age         int         `json:"age"`
	Gender      string      `json:"gender"`
	BoundingBox BoundingBox `json:"bounding_box"`
}

// BoundingBox 边界框
type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ImageInfo 图片信息
type ImageInfo struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Format     string `json:"format"`
	Size       int64  `json:"size"`
	ColorSpace string `json:"color_space"`
}

// FileProcessResponse 文件处理响应
type FileProcessResponse struct {
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	FileType    string    `json:"file_type"`
	TempPath    string    `json:"temp_path"`
	OSSKey      string    `json:"oss_key"`
	OSSURL      string    `json:"oss_url"`
	ProcessedAt time.Time `json:"processed_at"`
}

// GeminiImageAnalysis Gemini图片分析结果
type GeminiImageAnalysis struct {
	Description string                 `json:"description"`
	Objects     []DetectedObject       `json:"objects"`
	OCRText     string                 `json:"ocr_text"`
	RawResponse map[string]interface{} `json:"raw_response"`
}

// GeminiMultimodalRequest Gemini多模态请求
type GeminiMultimodalRequest struct {
	Text  string            `json:"text"`
	Files []GeminiFileInput `json:"files"`
}

// GeminiMultimodalResponse Gemini多模态响应
type GeminiMultimodalResponse struct {
	Text     string                 `json:"text"`
	Analysis map[string]interface{} `json:"analysis"`
	Files    []GeminiFileOutput     `json:"files"`
}

// GeminiFileInput Gemini文件输入
type GeminiFileInput struct {
	FileName string `json:"file_name"`
	FileData []byte `json:"file_data"`
	FileType string `json:"file_type"`
}

// GeminiFileOutput Gemini文件输出
type GeminiFileOutput struct {
	FileName string                 `json:"file_name"`
	Analysis map[string]interface{} `json:"analysis"`
	Summary  string                 `json:"summary"`
}

// NewMultimodalProcessor 创建多模态处理器
func NewMultimodalProcessor(cfg *config.GeminiConfig, logger *logx.Logger) *MultimodalProcessor {
	return &MultimodalProcessor{
		config:  cfg,
		logger:  logger,
		tempDir: "/tmp/loomi_multimodal",
	}
}

// ShouldUseMultimodalProcessing 检测是否需要多模态处理
func (mp *MultimodalProcessor) ShouldUseMultimodalProcessing(ctx context.Context, userID, sessionID, instruction string) (bool, error) {
	// 检查指令中是否包含文件引用
	fileKeywords := []string{
		"文件", "图片", "照片", "文档", "附件", "上传", "分析", "识别",
		"file", "image", "photo", "document", "attachment", "upload", "analyze", "recognize",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".pdf", ".doc", ".docx", ".txt",
	}

	instructionLower := strings.ToLower(instruction)
	for _, keyword := range fileKeywords {
		if strings.Contains(instructionLower, keyword) {
			mp.logger.Info(ctx, "检测到多模态处理需求",
				logx.KV("user_id", userID),
				logx.KV("session_id", sessionID),
				logx.KV("keyword", keyword))
			return true, nil
		}
	}

	return false, nil
}

// ProcessMultimodalFiles 处理多模态文件
func (mp *MultimodalProcessor) ProcessMultimodalFiles(ctx context.Context, req ProcessMultimodalRequest) (*ProcessMultimodalResponse, error) {
	mp.logger.Info(ctx, "开始处理多模态文件",
		logx.KV("user_id", req.UserID),
		logx.KV("session_id", req.SessionID),
		logx.KV("agent_name", req.AgentName))

	response := &ProcessMultimodalResponse{
		AnalysisResults: []FileAnalysisResult{},
		ProcessedFiles:  []ProcessedFile{},
		TempFiles:       []string{},
	}

	// 1. 提取文件引用
	fileRefs, err := mp.extractFileReferences(req.Instruction)
	if err != nil {
		return nil, fmt.Errorf("提取文件引用失败: %w", err)
	}

	if len(fileRefs) == 0 {
		mp.logger.Info(ctx, "未找到文件引用，跳过多模态处理")
		return response, nil
	}

	mp.logger.Info(ctx, "找到文件引用", logx.KV("count", len(fileRefs)))

	// 2. 处理每个文件
	for _, fileRef := range fileRefs {
		analysisResult, processedFile, err := mp.processFile(ctx, fileRef, req.UserID, req.SessionID)
		if err != nil {
			mp.logger.Error(ctx, "处理文件失败",
				logx.KV("file_ref", fileRef),
				logx.KV("error", err))
			continue
		}

		if analysisResult != nil {
			response.AnalysisResults = append(response.AnalysisResults, *analysisResult)
		}

		if processedFile != nil {
			response.ProcessedFiles = append(response.ProcessedFiles, *processedFile)
			response.TempFiles = append(response.TempFiles, processedFile.TempPath)
		}
	}

	mp.logger.Info(ctx, "多模态文件处理完成",
		logx.KV("processed_files", len(response.ProcessedFiles)),
		logx.KV("analysis_results", len(response.AnalysisResults)))

	return response, nil
}

// AnalyzeImage 分析图片
func (mp *MultimodalProcessor) AnalyzeImage(ctx context.Context, req AnalyzeImageRequest) (*AnalyzeImageResponse, error) {
	mp.logger.Info(ctx, "开始分析图片",
		logx.KV("image_size", len(req.ImageData)),
		logx.KV("analysis_types", req.AnalysisTypes))

	if mp.geminiClient == nil {
		return nil, fmt.Errorf("Gemini客户端未初始化")
	}

	// 使用Gemini分析图片
	analysis, err := mp.geminiClient.AnalyzeImage(ctx, req.ImageData, req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("Gemini图片分析失败: %w", err)
	}

	response := &AnalyzeImageResponse{
		Description:   analysis.Description,
		OCRText:       analysis.OCRText,
		Objects:       analysis.Objects,
		AnalysisTypes: req.AnalysisTypes,
		RawAnalysis:   analysis.RawResponse,
		ProcessedAt:   time.Now(),
	}

	mp.logger.Info(ctx, "图片分析完成",
		logx.KV("objects_count", len(response.Objects)),
		logx.KV("ocr_length", len(response.OCRText)))

	return response, nil
}

// ProcessFileUpload 处理文件上传
func (mp *MultimodalProcessor) ProcessFileUpload(ctx context.Context, file *multipart.FileHeader, userID, sessionID string) (*FileProcessResponse, error) {
	mp.logger.Info(ctx, "开始处理文件上传",
		logx.KV("file_name", file.Filename),
		logx.KV("file_size", file.Size),
		logx.KV("user_id", userID))

	// 1. 保存到临时目录
	tempPath, err := mp.saveToTemp(file, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("保存临时文件失败: %w", err)
	}

	// 2. 上传到OSS
	ossKey := mp.generateOSSKey(userID, sessionID, file.Filename)
	ossURL, err := mp.uploadToOSS(ctx, tempPath, ossKey)
	if err != nil {
		// 清理临时文件
		os.Remove(tempPath)
		return nil, fmt.Errorf("上传到OSS失败: %w", err)
	}

	response := &FileProcessResponse{
		FileName:    file.Filename,
		FileSize:    file.Size,
		FileType:    mp.getFileType(file.Filename),
		TempPath:    tempPath,
		OSSKey:      ossKey,
		OSSURL:      ossURL,
		ProcessedAt: time.Now(),
	}

	mp.logger.Info(ctx, "文件上传处理完成",
		logx.KV("oss_key", ossKey),
		logx.KV("oss_url", ossURL))

	return response, nil
}

// CleanupTempFiles 清理临时文件
func (mp *MultimodalProcessor) CleanupTempFiles(ctx context.Context, filePaths []string) error {
	mp.logger.Info(ctx, "开始清理临时文件", logx.KV("count", len(filePaths)))

	for _, filePath := range filePaths {
		if err := os.Remove(filePath); err != nil {
			mp.logger.Error(ctx, "删除临时文件失败",
				logx.KV("file_path", filePath),
				logx.KV("error", err))
		}
	}

	mp.logger.Info(ctx, "临时文件清理完成")
	return nil
}

// extractFileReferences 提取文件引用
func (mp *MultimodalProcessor) extractFileReferences(instruction string) ([]string, error) {
	// 简单的文件引用提取逻辑
	// 实际实现中应该使用更复杂的解析器
	fileRefs := []string{}

	// 查找常见的文件引用模式
	patterns := []string{
		"文件:", "图片:", "文档:", "附件:",
		"file:", "image:", "document:", "attachment:",
	}

	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(instruction), pattern) {
			// 提取文件名
			parts := strings.Split(instruction, pattern)
			if len(parts) > 1 {
				filename := strings.TrimSpace(parts[1])
				if filename != "" {
					fileRefs = append(fileRefs, filename)
				}
			}
		}
	}

	return fileRefs, nil
}

// processFile 处理单个文件
func (mp *MultimodalProcessor) processFile(ctx context.Context, fileRef, userID, sessionID string) (*FileAnalysisResult, *ProcessedFile, error) {
	// 1. 检查文件是否存在
	if !mp.fileExists(fileRef) {
		return nil, nil, fmt.Errorf("文件不存在: %s", fileRef)
	}

	// 2. 读取文件
	fileData, err := os.ReadFile(fileRef)
	if err != nil {
		return nil, nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 3. 上传到OSS
	ossKey := mp.generateOSSKey(userID, sessionID, filepath.Base(fileRef))
	ossURL, err := mp.uploadToOSS(ctx, fileRef, ossKey)
	if err != nil {
		return nil, nil, fmt.Errorf("上传到OSS失败: %w", err)
	}

	// 4. 分析文件
	analysis, err := mp.analyzeFile(ctx, fileData, fileRef)
	if err != nil {
		mp.logger.Error(ctx, "分析文件失败",
			logx.KV("file_ref", fileRef),
			logx.KV("error", err))
	}

	processedFile := &ProcessedFile{
		OriginalPath: fileRef,
		TempPath:     fileRef,
		OSSKey:       ossKey,
		OSSURL:       ossURL,
		FileSize:     int64(len(fileData)),
		FileType:     mp.getFileType(fileRef),
	}

	var analysisResult *FileAnalysisResult
	if analysis != nil {
		analysisResult = &FileAnalysisResult{
			FileName:    filepath.Base(fileRef),
			FileType:    mp.getFileType(fileRef),
			FileSize:    int64(len(fileData)),
			Analysis:    *analysis,
			OSSKey:      ossKey,
			OSSURL:      ossURL,
			ProcessedAt: time.Now(),
		}
	}

	return analysisResult, processedFile, nil
}

// saveToTemp 保存到临时目录
func (mp *MultimodalProcessor) saveToTemp(file *multipart.FileHeader, userID, sessionID string) (string, error) {
	// 创建临时目录
	if err := os.MkdirAll(mp.tempDir, 0755); err != nil {
		return "", err
	}

	// 生成临时文件路径
	tempPath := filepath.Join(mp.tempDir, fmt.Sprintf("%s_%s_%s", userID, sessionID, file.Filename))

	// 保存文件
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(tempPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return tempPath, err
}

// uploadToOSS 上传到OSS
func (mp *MultimodalProcessor) uploadToOSS(ctx context.Context, filePath, key string) (string, error) {
	if mp.ossClient == nil {
		return "", fmt.Errorf("OSS客户端未初始化")
	}

	url, err := mp.ossClient.UploadFile(ctx, filePath, key)
	if err != nil {
		return "", err
	}

	return url, nil
}

// generateOSSKey 生成OSS键
func (mp *MultimodalProcessor) generateOSSKey(userID, sessionID, filename string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("multimodal/%s/%s/%s_%s", userID, sessionID, timestamp, filename)
}

// getFileType 获取文件类型
func (mp *MultimodalProcessor) getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// fileExists 检查文件是否存在
func (mp *MultimodalProcessor) fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// analyzeFile 分析文件
func (mp *MultimodalProcessor) analyzeFile(ctx context.Context, fileData []byte, filePath string) (*map[string]interface{}, error) {
	fileType := mp.getFileType(filePath)

	// 根据文件类型进行分析
	if strings.HasPrefix(fileType, "image/") {
		return mp.analyzeImageFile(ctx, fileData)
	} else if fileType == "application/pdf" {
		return mp.analyzePDFFile(ctx, fileData)
	} else if strings.HasPrefix(fileType, "text/") {
		return mp.analyzeTextFile(ctx, fileData)
	}

	return &map[string]interface{}{
		"file_type": fileType,
		"size":      len(fileData),
		"analysis":  "basic_file_info",
	}, nil
}

// analyzeImageFile 分析图片文件
func (mp *MultimodalProcessor) analyzeImageFile(ctx context.Context, imageData []byte) (*map[string]interface{}, error) {
	if mp.geminiClient == nil {
		return &map[string]interface{}{
			"file_type": "image",
			"size":      len(imageData),
			"analysis":  "image_file_detected",
		}, nil
	}

	analysis, err := mp.geminiClient.AnalyzeImage(ctx, imageData, "请详细描述这张图片的内容")
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"file_type":   "image",
		"size":        len(imageData),
		"description": analysis.Description,
		"ocr_text":    analysis.OCRText,
		"objects":     analysis.Objects,
		"analysis":    "gemini_image_analysis",
	}

	return &result, nil
}

// analyzePDFFile 分析PDF文件
func (mp *MultimodalProcessor) analyzePDFFile(ctx context.Context, pdfData []byte) (*map[string]interface{}, error) {
	// 简单的PDF分析，实际实现中应该使用PDF解析库
	result := map[string]interface{}{
		"file_type": "application/pdf",
		"size":      len(pdfData),
		"analysis":  "pdf_file_detected",
		"note":      "PDF内容提取需要专门的PDF解析库",
	}

	return &result, nil
}

// analyzeTextFile 分析文本文件
func (mp *MultimodalProcessor) analyzeTextFile(ctx context.Context, textData []byte) (*map[string]interface{}, error) {
	text := string(textData)
	lines := strings.Split(text, "\n")
	words := strings.Fields(text)

	result := map[string]interface{}{
		"file_type":  "text/plain",
		"size":       len(textData),
		"line_count": len(lines),
		"word_count": len(words),
		"char_count": len(text),
		"analysis":   "text_file_analysis",
		"preview":    mp.truncateString(text, 200),
	}

	return &result, nil
}

// truncateString 截断字符串
func (mp *MultimodalProcessor) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
