package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// FeishuAlert 飞书告警客户端
type FeishuAlert struct {
	logger     log.Logger
	config     *config.Config
	webhookURL string
	enabled    bool
	httpClient *http.Client
}

// NewFeishuAlert 创建飞书告警客户端
func NewFeishuAlert(logger log.Logger, config *config.Config) *FeishuAlert {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	enabled := webhookURL != "" && os.Getenv("FEISHU_ALERT_ENABLED") != "false"

	return &FeishuAlert{
		logger:     logger,
		config:     config,
		webhookURL: webhookURL,
		enabled:    enabled,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send 发送飞书告警
func (fa *FeishuAlert) Send(ctx context.Context, message *AlertMessage) error {
	if !fa.enabled {
		fa.logger.Debug(ctx, "飞书告警已禁用，跳过发送")
		return nil
	}

	fa.logger.Info(ctx, "发送飞书告警",
		"alert_id", message.ID,
		"title", message.Title,
		"level", message.Level)

	// 构建飞书消息
	feishuMessage := fa.buildFeishuMessage(message)

	// 序列化消息
	jsonData, err := json.Marshal(feishuMessage)
	if err != nil {
		return fmt.Errorf("序列化飞书消息失败: %w", err)
	}

	// 发送HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", fa.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := fa.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("飞书API返回错误状态码: %d", resp.StatusCode)
	}

	fa.logger.Info(ctx, "飞书告警发送成功", "alert_id", message.ID)
	return nil
}

// GetName 获取客户端名称
func (fa *FeishuAlert) GetName() string {
	return "feishu"
}

// IsEnabled 检查是否启用
func (fa *FeishuAlert) IsEnabled() bool {
	return fa.enabled
}

// GetConfig 获取配置
func (fa *FeishuAlert) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"webhook_url": fa.webhookURL,
		"enabled":     fa.enabled,
	}
}

// buildFeishuMessage 构建飞书消息
func (fa *FeishuAlert) buildFeishuMessage(message *AlertMessage) *FeishuMessage {
	// 根据告警级别选择颜色
	color := fa.getColorByLevel(message.Level)

	// 构建消息内容
	content := fmt.Sprintf("**%s**\n\n%s", message.Title, message.Content)

	// 添加元数据
	if len(message.Metadata) > 0 {
		content += "\n\n**详细信息:**\n"
		for key, value := range message.Metadata {
			content += fmt.Sprintf("- %s: %v\n", key, value)
		}
	}

	// 添加标签
	if len(message.Tags) > 0 {
		content += fmt.Sprintf("\n**标签:** %s\n", joinStrings(message.Tags, ", "))
	}

	// 添加时间戳
	content += fmt.Sprintf("\n**时间:** %s", message.Timestamp.Format("2006-01-02 15:04:05"))

	return &FeishuMessage{
		MsgType: "interactive",
		Card: FeishuCard{
			Header: FeishuHeader{
				Template: color,
				Title: FeishuTitle{
					Tag:     "plain_text",
					Content: message.Title,
				},
			},
			Elements: []FeishuElement{
				{
					Tag: "div",
					Text: FeishuText{
						Tag:     "lark_md",
						Content: content,
					},
				},
				{
					Tag: "hr",
				},
				{
					Tag: "div",
					Fields: []FeishuField{
						{
							IsShort: true,
							Text: FeishuText{
								Tag:     "lark_md",
								Content: fmt.Sprintf("**级别:** %s", message.Level),
							},
						},
						{
							IsShort: true,
							Text: FeishuText{
								Tag:     "lark_md",
								Content: fmt.Sprintf("**类型:** %s", message.Type),
							},
						},
						{
							IsShort: true,
							Text: FeishuText{
								Tag:     "lark_md",
								Content: fmt.Sprintf("**来源:** %s", message.Source),
							},
						},
						{
							IsShort: true,
							Text: FeishuText{
								Tag:     "lark_md",
								Content: fmt.Sprintf("**优先级:** %d", message.Priority),
							},
						},
					},
				},
			},
		},
	}
}

// getColorByLevel 根据告警级别获取颜色
func (fa *FeishuAlert) getColorByLevel(level AlertLevel) string {
	switch level {
	case AlertLevelInfo:
		return "blue"
	case AlertLevelWarning:
		return "orange"
	case AlertLevelError:
		return "red"
	case AlertLevelCritical:
		return "red"
	default:
		return "grey"
	}
}

// FeishuMessage 飞书消息结构
type FeishuMessage struct {
	MsgType string     `json:"msg_type"`
	Card    FeishuCard `json:"card"`
}

// FeishuCard 飞书卡片
type FeishuCard struct {
	Header   FeishuHeader    `json:"header"`
	Elements []FeishuElement `json:"elements"`
}

// FeishuHeader 飞书卡片头部
type FeishuHeader struct {
	Template string      `json:"template"`
	Title    FeishuTitle `json:"title"`
}

// FeishuTitle 飞书标题
type FeishuTitle struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// FeishuElement 飞书元素
type FeishuElement struct {
	Tag    string        `json:"tag"`
	Text   FeishuText    `json:"text,omitempty"`
	Fields []FeishuField `json:"fields,omitempty"`
}

// FeishuText 飞书文本
type FeishuText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// FeishuField 飞书字段
type FeishuField struct {
	IsShort bool       `json:"is_short"`
	Text    FeishuText `json:"text"`
}

// 辅助函数
func joinStrings(strs []string, separator string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += separator + strs[i]
	}
	return result
}
