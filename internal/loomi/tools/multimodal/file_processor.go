package multimodal

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	"github.com/blueplan/loomi-go/internal/loomi/database"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// FileProcessor 文件处理器
type FileProcessor struct {
	config       *config.Config
	logger       *logx.Logger
	ossClient    OSSClient
	geminiClient GeminiClient
	fileStorage  database.FileStorage
	tempDir      string
}

// FileInfo 文件信息
type FileInfo struct {
	ID           int64     `json:"id"`
	UserID       string    `json:"user_id"`
	SessionID    string    `json:"session_id"`
	FileName     string    `json:"file_name"`
	OriginalName string    `json:"original_name"`
	FileType     string    `json:"file_type"`
	FileSize     int64     `json:"file_size"`
	MD5Hash      string    `json:"md5_hash"`
	OSSKey       string    `json:"oss_key"`
	OSSURL       string    `json:"oss_url"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProcessFileRequest 处理文件请求
type ProcessFileRequest struct {
	File        *multipart.FileHeader `json:"file"`
	UserID      string                `json:"user_id"`
	SessionID   string                `json:"session_id"`
	Description string                `json:"description"`
	ProcessType string                `json:"process_type"` // "upload", "analyze", "both"
}

// ProcessFileResponse 处理文件响应
type ProcessFileResponse struct {
	FileInfo    FileInfo               `json:"file_info"`
	Analysis    map[string]interface{} `json:"analysis,omitempty"`
	TempPath    string                 `json:"temp_path,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
}

// NewFileProcessor 创建文件处理器
func NewFileProcessor(cfg *config.Config, logger *logx.Logger, fileStorage database.FileStorage) *FileProcessor {
	return &FileProcessor{
		config:      cfg,
		logger:      logger,
		fileStorage: fileStorage,
		tempDir:     "/tmp/loomi_files",
	}
}

// ProcessFile 处理文件
func (fp *FileProcessor) ProcessFile(ctx context.Context, req ProcessFileRequest) (*ProcessFileResponse, error) {
	fp.logger.Info(ctx, "开始处理文件",
		logx.KV("file_name", req.File.Filename),
		logx.KV("file_size", req.File.Size),
		logx.KV("user_id", req.UserID),
		logx.KV("process_type", req.ProcessType))

	// 1. 保存到临时目录
	tempPath, err := fp.saveToTemp(req.File, req.UserID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("保存临时文件失败: %w", err)
	}

	// 2. 计算文件哈希
	md5Hash, err := fp.calculateMD5(tempPath)
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("计算文件哈希失败: %w", err)
	}

	// 3. 检查文件是否已存在
	existingFile, err := fp.findExistingFile(ctx, req.UserID, md5Hash)
	if err == nil && existingFile != nil {
		// 文件已存在，返回现有文件信息
		os.Remove(tempPath)
		fp.logger.Info(ctx, "文件已存在，返回现有文件信息",
			logx.KV("file_id", existingFile.ID),
			logx.KV("md5_hash", md5Hash))

		return &ProcessFileResponse{
			FileInfo: FileInfo{
				ID:          existingFile.ID,
				UserID:      existingFile.UserID,
				SessionID:   existingFile.SessionID,
				FileName:    existingFile.FileName,
				FileType:    existingFile.FileType,
				FileSize:    existingFile.FileSize,
				OSSKey:      existingFile.OSSKey,
				Description: existingFile.Description,
				CreatedAt:   existingFile.CreatedAt,
				UpdatedAt:   existingFile.UpdatedAt,
			},
			ProcessedAt: time.Now(),
		}, nil
	}

	// 4. 上传到OSS
	ossKey := fp.generateOSSKey(req.UserID, req.SessionID, req.File.Filename)
	ossURL, err := fp.uploadToOSS(ctx, tempPath, ossKey)
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("上传到OSS失败: %w", err)
	}

	// 5. 保存文件信息到数据库
	fileInfo := FileInfo{
		UserID:       req.UserID,
		SessionID:    req.SessionID,
		FileName:     filepath.Base(tempPath),
		OriginalName: req.File.Filename,
		FileType:     fp.getFileType(req.File.Filename),
		FileSize:     req.File.Size,
		MD5Hash:      md5Hash,
		OSSKey:       ossKey,
		OSSURL:       ossURL,
		Description:  req.Description,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	savedFile, err := fp.saveFileInfo(ctx, fileInfo)
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("保存文件信息失败: %w", err)
	}

	response := &ProcessFileResponse{
		FileInfo:    *savedFile,
		TempPath:    tempPath,
		ProcessedAt: time.Now(),
	}

	// 6. 根据处理类型进行文件分析
	if req.ProcessType == "analyze" || req.ProcessType == "both" {
		analysis, err := fp.analyzeFile(ctx, tempPath, fileInfo)
		if err != nil {
			fp.logger.Error(ctx, "文件分析失败",
				logx.KV("file_name", req.File.Filename),
				logx.KV("error", err))
		} else {
			response.Analysis = analysis
		}
	}

	// 7. 清理临时文件（如果不需要保留）
	if req.ProcessType != "analyze" {
		os.Remove(tempPath)
		response.TempPath = ""
	}

	fp.logger.Info(ctx, "文件处理完成",
		logx.KV("file_id", savedFile.ID),
		logx.KV("oss_key", ossKey),
		logx.KV("oss_url", ossURL))

	return response, nil
}

// saveToTemp 保存到临时目录
func (fp *FileProcessor) saveToTemp(file *multipart.FileHeader, userID, sessionID string) (string, error) {
	// 创建临时目录
	if err := os.MkdirAll(fp.tempDir, 0755); err != nil {
		return "", err
	}

	// 生成临时文件路径
	tempFileName := fmt.Sprintf("%s_%s_%s_%d_%s",
		userID, sessionID, time.Now().Format("20060102_150405"),
		file.Size, file.Filename)
	tempPath := filepath.Join(fp.tempDir, tempFileName)

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

// calculateMD5 计算文件MD5哈希
func (fp *FileProcessor) calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// findExistingFile 查找已存在的文件
func (fp *FileProcessor) findExistingFile(ctx context.Context, userID, md5Hash string) (*database.FileRecord, error) {
	// 这里应该实现根据MD5哈希查找已存在文件的逻辑
	// 由于数据库接口限制，暂时返回nil
	return nil, nil
}

// uploadToOSS 上传到OSS
func (fp *FileProcessor) uploadToOSS(ctx context.Context, filePath, key string) (string, error) {
	if fp.ossClient == nil {
		return "", fmt.Errorf("OSS客户端未初始化")
	}

	url, err := fp.ossClient.UploadFile(ctx, filePath, key)
	if err != nil {
		return "", err
	}

	return url, nil
}

// generateOSSKey 生成OSS键
func (fp *FileProcessor) generateOSSKey(userID, sessionID, filename string) string {
	timestamp := time.Now().Format("20060102_150405")
	safeFilename := strings.ReplaceAll(filename, "/", "_")
	return fmt.Sprintf("files/%s/%s/%s_%s", userID, sessionID, timestamp, safeFilename)
}

// getFileType 获取文件类型
func (fp *FileProcessor) getFileType(filename string) string {
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
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/x-rar-compressed"
	default:
		return "application/octet-stream"
	}
}

// saveFileInfo 保存文件信息
func (fp *FileProcessor) saveFileInfo(ctx context.Context, fileInfo FileInfo) (*FileInfo, error) {
	if fp.fileStorage == nil {
		return &fileInfo, nil
	}

	req := database.SaveFileRequest{
		UserID:      fileInfo.UserID,
		SessionID:   fileInfo.SessionID,
		FileName:    fileInfo.FileName,
		FileType:    fileInfo.FileType,
		FileSize:    fileInfo.FileSize,
		OSSKey:      fileInfo.OSSKey,
		Description: fileInfo.Description,
	}

	resp, err := fp.fileStorage.SaveFile(ctx, req)
	if err != nil {
		return nil, err
	}

	fileInfo.ID = resp.ID
	return &fileInfo, nil
}

// analyzeFile 分析文件
func (fp *FileProcessor) analyzeFile(ctx context.Context, filePath string, fileInfo FileInfo) (map[string]interface{}, error) {
	fileType := fileInfo.FileType

	// 根据文件类型进行分析
	if strings.HasPrefix(fileType, "image/") {
		return fp.analyzeImageFile(ctx, filePath)
	} else if fileType == "application/pdf" {
		return fp.analyzePDFFile(ctx, filePath)
	} else if strings.HasPrefix(fileType, "text/") {
		return fp.analyzeTextFile(ctx, filePath)
	}

	return map[string]interface{}{
		"file_type": fileType,
		"size":      fileInfo.FileSize,
		"analysis":  "basic_file_info",
		"note":      "不支持的文件类型分析",
	}, nil
}

// analyzeImageFile 分析图片文件
func (fp *FileProcessor) analyzeImageFile(ctx context.Context, filePath string) (map[string]interface{}, error) {
	if fp.geminiClient == nil {
		return map[string]interface{}{
			"file_type": "image",
			"analysis":  "image_file_detected",
			"note":      "Gemini客户端未配置，无法进行图片分析",
		}, nil
	}

	// 读取图片数据
	imageData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取图片文件失败: %w", err)
	}

	// 使用Gemini分析图片
	analysis, err := fp.geminiClient.AnalyzeImage(ctx, imageData, "请详细描述这张图片的内容")
	if err != nil {
		return nil, fmt.Errorf("Gemini图片分析失败: %w", err)
	}

	result := map[string]interface{}{
		"file_type":   "image",
		"description": analysis.Description,
		"ocr_text":    analysis.OCRText,
		"objects":     analysis.Objects,
		"analysis":    "gemini_image_analysis",
	}

	return result, nil
}

// analyzePDFFile 分析PDF文件
func (fp *FileProcessor) analyzePDFFile(ctx context.Context, filePath string) (map[string]interface{}, error) {
	// 简单的PDF分析，实际实现中应该使用PDF解析库
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"file_type": "application/pdf",
		"size":      fileInfo.Size(),
		"analysis":  "pdf_file_detected",
		"note":      "PDF内容提取需要专门的PDF解析库",
	}

	return result, nil
}

// analyzeTextFile 分析文本文件
func (fp *FileProcessor) analyzeTextFile(ctx context.Context, filePath string) (map[string]interface{}, error) {
	textData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文本文件失败: %w", err)
	}

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
		"preview":    fp.truncateString(text, 200),
	}

	return result, nil
}

// truncateString 截断字符串
func (fp *FileProcessor) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetFileInfo 获取文件信息
func (fp *FileProcessor) GetFileInfo(ctx context.Context, fileID int64) (*FileInfo, error) {
	if fp.fileStorage == nil {
		return nil, fmt.Errorf("文件存储未初始化")
	}

	req := database.GetFileRequest{ID: fileID}
	record, err := fp.fileStorage.GetFile(ctx, req)
	if err != nil {
		return nil, err
	}

	if record == nil {
		return nil, fmt.Errorf("文件不存在")
	}

	return &FileInfo{
		ID:          record.ID,
		UserID:      record.UserID,
		SessionID:   record.SessionID,
		FileName:    record.FileName,
		FileType:    record.FileType,
		FileSize:    record.FileSize,
		OSSKey:      record.OSSKey,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}, nil
}

// DeleteFile 删除文件
func (fp *FileProcessor) DeleteFile(ctx context.Context, fileID int64) error {
	if fp.fileStorage == nil {
		return fmt.Errorf("文件存储未初始化")
	}

	// 先获取文件信息
	fileInfo, err := fp.GetFileInfo(ctx, fileID)
	if err != nil {
		return err
	}

	// 从OSS删除文件
	if fp.ossClient != nil && fileInfo.OSSKey != "" {
		if err := fp.ossClient.DeleteFile(ctx, fileInfo.OSSKey); err != nil {
			fp.logger.Error(ctx, "从OSS删除文件失败",
				logx.KV("oss_key", fileInfo.OSSKey),
				logx.KV("error", err))
		}
	}

	// 从数据库删除文件信息
	req := database.DeleteFileRequest{ID: fileID}
	return fp.fileStorage.DeleteFile(ctx, req)
}

// ListFiles 列出文件
func (fp *FileProcessor) ListFiles(ctx context.Context, userID, sessionID string, limit, offset int) ([]FileInfo, error) {
	if fp.fileStorage == nil {
		return nil, fmt.Errorf("文件存储未初始化")
	}

	req := database.ListFilesRequest{
		UserID:    userID,
		SessionID: sessionID,
		Limit:     limit,
		Offset:    offset,
	}

	resp, err := fp.fileStorage.ListFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	var fileInfos []FileInfo
	for _, record := range resp.Files {
		fileInfos = append(fileInfos, FileInfo{
			ID:          record.ID,
			UserID:      record.UserID,
			SessionID:   record.SessionID,
			FileName:    record.FileName,
			FileType:    record.FileType,
			FileSize:    record.FileSize,
			OSSKey:      record.OSSKey,
			Description: record.Description,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
		})
	}

	return fileInfos, nil
}
