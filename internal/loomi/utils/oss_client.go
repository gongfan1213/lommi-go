package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// OSSClient 阿里云OSS客户端
type OSSClient struct {
	logger          log.Logger
	config          *config.Config
	endpoint        string
	bucketName      string
	accessKeyID     string
	accessKeySecret string
	httpClient      *http.Client
}

// NewOSSClient 创建OSS客户端
func NewOSSClient(logger log.Logger, config *config.Config) *OSSClient {
	endpoint := os.Getenv("OSS_ENDPOINT")
	bucketName := os.Getenv("OSS_BUCKET_NAME")
	accessKeyID := os.Getenv("OSS_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")

	return &OSSClient{
		logger:          logger,
		config:          config,
		endpoint:        endpoint,
		bucketName:      bucketName,
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// UploadFile 上传文件
func (oc *OSSClient) UploadFile(ctx context.Context, filePath, objectKey string, options *UploadOptions) error {
	oc.logger.Info(ctx, "开始上传文件",
		"file_path", filePath,
		"object_key", objectKey,
		"bucket", oc.bucketName)

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("文件不存在: %w", err)
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 设置默认选项
	if options == nil {
		options = &UploadOptions{}
	}
	if options.ContentType == "" {
		options.ContentType = oc.getContentType(filepath.Ext(filePath))
	}
	if options.Metadata == nil {
		options.Metadata = make(map[string]string)
	}

	// 构建请求URL
	requestURL := oc.buildRequestURL(objectKey)

	// 创建PUT请求
	req, err := http.NewRequestWithContext(ctx, "PUT", requestURL, file)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", options.ContentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	req.Header.Set("x-oss-storage-class", options.StorageClass)

	// 添加自定义元数据
	for key, value := range options.Metadata {
		req.Header.Set("x-oss-meta-"+key, value)
	}

	// 添加认证头
	oc.addAuthHeaders(req, "PUT", objectKey, req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上传失败，状态码: %d", resp.StatusCode)
	}

	oc.logger.Info(ctx, "文件上传成功", "object_key", objectKey)
	return nil
}

// UploadFromReader 从Reader上传文件
func (oc *OSSClient) UploadFromReader(ctx context.Context, reader io.Reader, objectKey string, size int64, options *UploadOptions) error {
	oc.logger.Info(ctx, "开始从Reader上传文件",
		"object_key", objectKey,
		"size", size,
		"bucket", oc.bucketName)

	// 设置默认选项
	if options == nil {
		options = &UploadOptions{}
	}
	if options.ContentType == "" {
		options.ContentType = "application/octet-stream"
	}
	if options.Metadata == nil {
		options.Metadata = make(map[string]string)
	}

	// 构建请求URL
	requestURL := oc.buildRequestURL(objectKey)

	// 创建PUT请求
	req, err := http.NewRequestWithContext(ctx, "PUT", requestURL, reader)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", options.ContentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", size))
	req.Header.Set("x-oss-storage-class", options.StorageClass)

	// 添加自定义元数据
	for key, value := range options.Metadata {
		req.Header.Set("x-oss-meta-"+key, value)
	}

	// 添加认证头
	oc.addAuthHeaders(req, "PUT", objectKey, req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上传失败，状态码: %d", resp.StatusCode)
	}

	oc.logger.Info(ctx, "从Reader上传文件成功", "object_key", objectKey)
	return nil
}

// DownloadFile 下载文件
func (oc *OSSClient) DownloadFile(ctx context.Context, objectKey, localPath string) error {
	oc.logger.Info(ctx, "开始下载文件",
		"object_key", objectKey,
		"local_path", localPath,
		"bucket", oc.bucketName)

	// 构建请求URL
	requestURL := oc.buildRequestURL(objectKey)

	// 创建GET请求
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 添加认证头
	oc.addAuthHeaders(req, "GET", objectKey, req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 创建本地文件
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		return fmt.Errorf("创建本地目录失败: %w", err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer file.Close()

	// 复制数据
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("复制文件数据失败: %w", err)
	}

	oc.logger.Info(ctx, "文件下载成功", "object_key", objectKey, "local_path", localPath)
	return nil
}

// DownloadToWriter 下载文件到Writer
func (oc *OSSClient) DownloadToWriter(ctx context.Context, objectKey string, writer io.Writer) error {
	oc.logger.Info(ctx, "开始下载文件到Writer",
		"object_key", objectKey,
		"bucket", oc.bucketName)

	// 构建请求URL
	requestURL := oc.buildRequestURL(objectKey)

	// 创建GET请求
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 添加认证头
	oc.addAuthHeaders(req, "GET", objectKey, req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 复制数据到Writer
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("复制文件数据失败: %w", err)
	}

	oc.logger.Info(ctx, "下载文件到Writer成功", "object_key", objectKey)
	return nil
}

// DeleteFile 删除文件
func (oc *OSSClient) DeleteFile(ctx context.Context, objectKey string) error {
	oc.logger.Info(ctx, "开始删除文件",
		"object_key", objectKey,
		"bucket", oc.bucketName)

	// 构建请求URL
	requestURL := oc.buildRequestURL(objectKey)

	// 创建DELETE请求
	req, err := http.NewRequestWithContext(ctx, "DELETE", requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 添加认证头
	oc.addAuthHeaders(req, "DELETE", objectKey, req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送删除请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("删除失败，状态码: %d", resp.StatusCode)
	}

	oc.logger.Info(ctx, "文件删除成功", "object_key", objectKey)
	return nil
}

// ListFiles 列出文件
func (oc *OSSClient) ListFiles(ctx context.Context, prefix string, maxKeys int) (*ListFilesResponse, error) {
	oc.logger.Info(ctx, "开始列出文件",
		"prefix", prefix,
		"max_keys", maxKeys,
		"bucket", oc.bucketName)

	// 构建请求URL
	requestURL := oc.buildListRequestURL(prefix, maxKeys)

	// 创建GET请求
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 添加认证头
	oc.addAuthHeaders(req, "GET", "", req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送列表请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("列出文件失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应
	response, err := oc.parseListResponse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析列表响应失败: %w", err)
	}

	oc.logger.Info(ctx, "文件列表获取成功", "count", len(response.Contents))
	return response, nil
}

// GeneratePresignedURL 生成预签名URL
func (oc *OSSClient) GeneratePresignedURL(ctx context.Context, objectKey string, method string, expires time.Duration) (string, error) {
	oc.logger.Info(ctx, "开始生成预签名URL",
		"object_key", objectKey,
		"method", method,
		"expires", expires)

	// 构建请求URL
	requestURL := oc.buildRequestURL(objectKey)

	// 创建预签名URL
	presignedURL, err := oc.createPresignedURL(requestURL, method, expires)
	if err != nil {
		return "", fmt.Errorf("创建预签名URL失败: %w", err)
	}

	oc.logger.Info(ctx, "预签名URL生成成功", "object_key", objectKey)
	return presignedURL, nil
}

// GetFileInfo 获取文件信息
func (oc *OSSClient) GetFileInfo(ctx context.Context, objectKey string) (*FileInfo, error) {
	oc.logger.Info(ctx, "开始获取文件信息",
		"object_key", objectKey,
		"bucket", oc.bucketName)

	// 构建请求URL
	requestURL := oc.buildRequestURL(objectKey)

	// 创建HEAD请求
	req, err := http.NewRequestWithContext(ctx, "HEAD", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 添加认证头
	oc.addAuthHeaders(req, "HEAD", objectKey, req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取文件信息失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应头
	fileInfo := &FileInfo{
		Key:          objectKey,
		Size:         oc.parseContentLength(resp.Header.Get("Content-Length")),
		ContentType:  resp.Header.Get("Content-Type"),
		LastModified: oc.parseLastModified(resp.Header.Get("Last-Modified")),
		ETag:         resp.Header.Get("ETag"),
		StorageClass: resp.Header.Get("x-oss-storage-class"),
	}

	// 解析自定义元数据
	fileInfo.Metadata = make(map[string]string)
	for key, values := range resp.Header {
		if strings.HasPrefix(key, "x-oss-meta-") {
			if len(values) > 0 {
				fileInfo.Metadata[strings.TrimPrefix(key, "x-oss-meta-")] = values[0]
			}
		}
	}

	oc.logger.Info(ctx, "文件信息获取成功", "object_key", objectKey, "size", fileInfo.Size)
	return fileInfo, nil
}

// CopyFile 复制文件
func (oc *OSSClient) CopyFile(ctx context.Context, sourceKey, destKey string) error {
	oc.logger.Info(ctx, "开始复制文件",
		"source_key", sourceKey,
		"dest_key", destKey,
		"bucket", oc.bucketName)

	// 构建请求URL
	requestURL := oc.buildRequestURL(destKey)

	// 创建PUT请求
	req, err := http.NewRequestWithContext(ctx, "PUT", requestURL, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置复制头
	req.Header.Set("x-oss-copy-source", fmt.Sprintf("/%s/%s", oc.bucketName, sourceKey))

	// 添加认证头
	oc.addAuthHeaders(req, "PUT", destKey, req.Header)

	// 发送请求
	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送复制请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("复制失败，状态码: %d", resp.StatusCode)
	}

	oc.logger.Info(ctx, "文件复制成功", "source_key", sourceKey, "dest_key", destKey)
	return nil
}

// 私有方法

// buildRequestURL 构建请求URL
func (oc *OSSClient) buildRequestURL(objectKey string) string {
	return fmt.Sprintf("https://%s.%s/%s", oc.bucketName, oc.endpoint, url.PathEscape(objectKey))
}

// buildListRequestURL 构建列表请求URL
func (oc *OSSClient) buildListRequestURL(prefix string, maxKeys int) string {
	baseURL := fmt.Sprintf("https://%s.%s/", oc.bucketName, oc.endpoint)
	params := url.Values{}
	if prefix != "" {
		params.Set("prefix", prefix)
	}
	if maxKeys > 0 {
		params.Set("max-keys", fmt.Sprintf("%d", maxKeys))
	}

	if len(params) > 0 {
		return baseURL + "?" + params.Encode()
	}
	return baseURL
}

// addAuthHeaders 添加认证头
func (oc *OSSClient) addAuthHeaders(req *http.Request, method, objectKey string, headers http.Header) {
	// 这里应该实现阿里云OSS的签名算法
	// 为了简化，这里只是占位实现
	req.Header.Set("Authorization", "OSS "+oc.accessKeyID+":signature")
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
}

// createPresignedURL 创建预签名URL
func (oc *OSSClient) createPresignedURL(requestURL, method string, expires time.Duration) (string, error) {
	// 这里应该实现阿里云OSS的预签名URL算法
	// 为了简化，这里只是占位实现
	return requestURL + "?signature=presigned", nil
}

// parseListResponse 解析列表响应
func (oc *OSSClient) parseListResponse(body io.Reader) (*ListFilesResponse, error) {
	// 这里应该解析XML响应
	// 为了简化，这里返回空响应
	return &ListFilesResponse{
		Contents: []FileInfo{},
	}, nil
}

// getContentType 根据文件扩展名获取Content-Type
func (oc *OSSClient) getContentType(ext string) string {
	contentTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
	}

	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}
	return "application/octet-stream"
}

// parseContentLength 解析Content-Length
func (oc *OSSClient) parseContentLength(contentLength string) int64 {
	if contentLength == "" {
		return 0
	}

	// 简化的解析实现
	var size int64
	fmt.Sscanf(contentLength, "%d", &size)
	return size
}

// parseLastModified 解析Last-Modified
func (oc *OSSClient) parseLastModified(lastModified string) time.Time {
	if lastModified == "" {
		return time.Time{}
	}

	t, err := time.Parse(http.TimeFormat, lastModified)
	if err != nil {
		return time.Time{}
	}
	return t
}

// 数据结构

// UploadOptions 上传选项
type UploadOptions struct {
	ContentType  string            `json:"content_type"`
	StorageClass string            `json:"storage_class"`
	Metadata     map[string]string `json:"metadata"`
}

// ListFilesResponse 文件列表响应
type ListFilesResponse struct {
	Contents []FileInfo `json:"contents"`
}

// FileInfo 文件信息
type FileInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	StorageClass string            `json:"storage_class"`
	Metadata     map[string]string `json:"metadata"`
}
