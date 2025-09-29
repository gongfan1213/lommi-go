package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// WebhookAlert Webhook告警客户端
type WebhookAlert struct {
	logger     log.Logger
	config     *config.Config
	webhookURL string
	headers    map[string]string
	enabled    bool
	httpClient *http.Client
}

// NewWebhookAlert 创建Webhook告警客户端
func NewWebhookAlert(logger log.Logger, config *config.Config) *WebhookAlert {
	webhookURL := os.Getenv("WEBHOOK_URL")
	headersStr := os.Getenv("WEBHOOK_HEADERS")

	var headers map[string]string
	if headersStr != "" {
		headers = make(map[string]string)
		headerPairs := strings.Split(headersStr, ",")
		for _, pair := range headerPairs {
			if parts := strings.SplitN(pair, ":", 2); len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				headers[key] = value
			}
		}
	}

	enabled := webhookURL != "" && os.Getenv("WEBHOOK_ALERT_ENABLED") != "false"

	return &WebhookAlert{
		logger:     logger,
		config:     config,
		webhookURL: webhookURL,
		headers:    headers,
		enabled:    enabled,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send 发送Webhook告警
func (wa *WebhookAlert) Send(ctx context.Context, message *AlertMessage) error {
	if !wa.enabled {
		wa.logger.Debug(ctx, "Webhook告警已禁用，跳过发送")
		return nil
	}

	wa.logger.Info(ctx, "发送Webhook告警",
		"alert_id", message.ID,
		"title", message.Title,
		"level", message.Level,
		"webhook_url", wa.webhookURL)

	// 构建Webhook消息
	webhookMessage := wa.buildWebhookMessage(message)

	// 序列化消息
	jsonData, err := json.Marshal(webhookMessage)
	if err != nil {
		return fmt.Errorf("序列化Webhook消息失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", wa.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Loomi-Alert-System/1.0")

	// 添加自定义请求头
	for key, value := range wa.headers {
		req.Header.Set(key, value)
	}

	// 发送HTTP请求
	resp, err := wa.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Webhook API返回错误状态码: %d", resp.StatusCode)
	}

	wa.logger.Info(ctx, "Webhook告警发送成功", "alert_id", message.ID)
	return nil
}

// GetName 获取客户端名称
func (wa *WebhookAlert) GetName() string {
	return "webhook"
}

// IsEnabled 检查是否启用
func (wa *WebhookAlert) IsEnabled() bool {
	return wa.enabled
}

// GetConfig 获取配置
func (wa *WebhookAlert) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"webhook_url": wa.webhookURL,
		"headers":     wa.headers,
		"enabled":     wa.enabled,
	}
}

// buildWebhookMessage 构建Webhook消息
func (wa *WebhookAlert) buildWebhookMessage(message *AlertMessage) *WebhookMessage {
	return &WebhookMessage{
		ID:         message.ID,
		Title:      message.Title,
		Content:    message.Content,
		Level:      string(message.Level),
		Type:       string(message.Type),
		Source:     message.Source,
		Timestamp:  message.Timestamp,
		Metadata:   message.Metadata,
		Tags:       message.Tags,
		Priority:   message.Priority,
		Recipients: message.Recipients,
	}
}

// SendTestWebhook 发送测试Webhook
func (wa *WebhookAlert) SendTestWebhook(ctx context.Context) error {
	if !wa.enabled {
		return fmt.Errorf("Webhook告警已禁用")
	}

	testMessage := &AlertMessage{
		ID:      "test_webhook",
		Title:   "测试Webhook",
		Content: "这是一个测试Webhook消息，用于验证Webhook告警功能是否正常工作。",
		Level:   AlertLevelInfo,
		Type:    AlertTypeSystem,
		Source:  "webhook_test",
		Metadata: map[string]interface{}{
			"test": true,
		},
		Tags: []string{"test", "webhook"},
	}

	return wa.Send(ctx, testMessage)
}

// AddHeader 添加自定义请求头
func (wa *WebhookAlert) AddHeader(key, value string) {
	if wa.headers == nil {
		wa.headers = make(map[string]string)
	}
	wa.headers[key] = value
	wa.logger.Info(context.Background(), "添加Webhook请求头", "key", key)
}

// RemoveHeader 移除自定义请求头
func (wa *WebhookAlert) RemoveHeader(key string) {
	if wa.headers != nil {
		delete(wa.headers, key)
		wa.logger.Info(context.Background(), "移除Webhook请求头", "key", key)
	}
}

// GetHeaders 获取自定义请求头
func (wa *WebhookAlert) GetHeaders() map[string]string {
	return wa.headers
}

// SetHeaders 设置自定义请求头
func (wa *WebhookAlert) SetHeaders(headers map[string]string) {
	wa.headers = headers
	wa.logger.Info(context.Background(), "设置Webhook请求头", "count", len(headers))
}

// SetWebhookURL 设置Webhook URL
func (wa *WebhookAlert) SetWebhookURL(url string) {
	wa.webhookURL = url
	wa.enabled = url != ""
	wa.logger.Info(context.Background(), "设置Webhook URL", "url", url, "enabled", wa.enabled)
}

// GetWebhookURL 获取Webhook URL
func (wa *WebhookAlert) GetWebhookURL() string {
	return wa.webhookURL
}

// WebhookMessage Webhook消息结构
type WebhookMessage struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Content    string                 `json:"content"`
	Level      string                 `json:"level"`
	Type       string                 `json:"type"`
	Source     string                 `json:"source"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
	Tags       []string               `json:"tags"`
	Priority   int                    `json:"priority"`
	Recipients []string               `json:"recipients"`
}

// WebhookResponse Webhook响应结构
type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// ValidateWebhook 验证Webhook配置
func (wa *WebhookAlert) ValidateWebhook(ctx context.Context) error {
	if !wa.enabled {
		return fmt.Errorf("Webhook告警已禁用")
	}

	if wa.webhookURL == "" {
		return fmt.Errorf("Webhook URL未配置")
	}

	// 发送测试请求验证配置
	err := wa.SendTestWebhook(ctx)
	if err != nil {
		return fmt.Errorf("Webhook测试失败: %w", err)
	}

	wa.logger.Info(ctx, "Webhook配置验证成功")
	return nil
}

// GetWebhookStatus 获取Webhook状态
func (wa *WebhookAlert) GetWebhookStatus(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"enabled":     wa.enabled,
		"webhook_url": wa.webhookURL,
		"headers":     wa.headers,
	}

	// 尝试发送测试请求检查连通性
	if wa.enabled {
		err := wa.SendTestWebhook(ctx)
		status["reachable"] = err == nil
		if err != nil {
			status["error"] = err.Error()
		}
	}

	return status
}
