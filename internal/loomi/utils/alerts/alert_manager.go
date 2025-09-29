package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeSystem      AlertType = "system"
	AlertTypePerformance AlertType = "performance"
	AlertTypeSecurity    AlertType = "security"
	AlertTypeBusiness    AlertType = "business"
	AlertTypeNetwork     AlertType = "network"
	AlertTypeDatabase    AlertType = "database"
)

// AlertMessage 告警消息
type AlertMessage struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Content    string                 `json:"content"`
	Level      AlertLevel             `json:"level"`
	Type       AlertType              `json:"type"`
	Source     string                 `json:"source"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
	Tags       []string               `json:"tags"`
	Recipients []string               `json:"recipients"`
	Priority   int                    `json:"priority"`
}

// BaseAlert 基础告警接口
type BaseAlert interface {
	Send(ctx context.Context, message *AlertMessage) error
	GetName() string
	IsEnabled() bool
	GetConfig() map[string]interface{}
}

// AlertManager 告警管理器
type AlertManager struct {
	logger       log.Logger
	config       *config.Config
	alertClients map[string]BaseAlert
	enabled      bool
}

// NewAlertManager 创建告警管理器
func NewAlertManager(logger log.Logger, config *config.Config) *AlertManager {
	return &AlertManager{
		logger:       logger,
		config:       config,
		alertClients: make(map[string]BaseAlert),
		enabled:      true,
	}
}

// Initialize 初始化告警管理器
func (am *AlertManager) Initialize(ctx context.Context) error {
	am.logger.Info(ctx, "初始化告警管理器")

	// 从配置加载告警设置
	err := am.loadConfig()
	if err != nil {
		return fmt.Errorf("加载告警配置失败: %w", err)
	}

	// 初始化告警客户端
	err = am.initAlertClients(ctx)
	if err != nil {
		return fmt.Errorf("初始化告警客户端失败: %w", err)
	}

	am.logger.Info(ctx, "告警管理器初始化完成", "clients_count", len(am.alertClients))
	return nil
}

// SendAlert 发送告警
func (am *AlertManager) SendAlert(ctx context.Context, message *AlertMessage) error {
	if !am.enabled {
		am.logger.Debug(ctx, "告警管理器已禁用，跳过发送告警")
		return nil
	}

	am.logger.Info(ctx, "发送告警",
		"alert_id", message.ID,
		"title", message.Title,
		"level", message.Level,
		"type", message.Type,
		"source", message.Source)

	// 设置默认值
	if message.ID == "" {
		message.ID = am.generateAlertID()
	}
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	if message.Priority == 0 {
		message.Priority = am.getDefaultPriority(message.Level)
	}

	// 发送到所有启用的告警客户端
	var errors []error
	for name, client := range am.alertClients {
		if client.IsEnabled() {
			err := client.Send(ctx, message)
			if err != nil {
				am.logger.Error(ctx, "发送告警失败",
					"client", name,
					"alert_id", message.ID,
					"error", err)
				errors = append(errors, fmt.Errorf("客户端 %s 发送失败: %w", name, err))
			} else {
				am.logger.Info(ctx, "告警发送成功", "client", name, "alert_id", message.ID)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("部分告警发送失败: %v", errors)
	}

	return nil
}

// SendSystemAlert 发送系统告警
func (am *AlertManager) SendSystemAlert(ctx context.Context, title, content string, level AlertLevel, metadata map[string]interface{}) error {
	message := &AlertMessage{
		Title:    title,
		Content:  content,
		Level:    level,
		Type:     AlertTypeSystem,
		Source:   "system",
		Metadata: metadata,
		Tags:     []string{"system"},
	}

	return am.SendAlert(ctx, message)
}

// SendPerformanceAlert 发送性能告警
func (am *AlertManager) SendPerformanceAlert(ctx context.Context, title, content string, level AlertLevel, metadata map[string]interface{}) error {
	message := &AlertMessage{
		Title:    title,
		Content:  content,
		Level:    level,
		Type:     AlertTypePerformance,
		Source:   "performance_monitor",
		Metadata: metadata,
		Tags:     []string{"performance"},
	}

	return am.SendAlert(ctx, message)
}

// SendSecurityAlert 发送安全告警
func (am *AlertManager) SendSecurityAlert(ctx context.Context, title, content string, level AlertLevel, metadata map[string]interface{}) error {
	message := &AlertMessage{
		Title:    title,
		Content:  content,
		Level:    level,
		Type:     AlertTypeSecurity,
		Source:   "security_monitor",
		Metadata: metadata,
		Tags:     []string{"security"},
	}

	return am.SendAlert(ctx, message)
}

// SendBusinessAlert 发送业务告警
func (am *AlertManager) SendBusinessAlert(ctx context.Context, title, content string, level AlertLevel, metadata map[string]interface{}) error {
	message := &AlertMessage{
		Title:    title,
		Content:  content,
		Level:    level,
		Type:     AlertTypeBusiness,
		Source:   "business_monitor",
		Metadata: metadata,
		Tags:     []string{"business"},
	}

	return am.SendAlert(ctx, message)
}

// RegisterAlertClient 注册告警客户端
func (am *AlertManager) RegisterAlertClient(name string, client BaseAlert) {
	am.alertClients[name] = client
	am.logger.Info(context.Background(), "注册告警客户端", "name", name)
}

// GetAlertClients 获取所有告警客户端
func (am *AlertManager) GetAlertClients() map[string]BaseAlert {
	return am.alertClients
}

// Enable 启用告警管理器
func (am *AlertManager) Enable() {
	am.enabled = true
	am.logger.Info(context.Background(), "告警管理器已启用")
}

// Disable 禁用告警管理器
func (am *AlertManager) Disable() {
	am.enabled = false
	am.logger.Info(context.Background(), "告警管理器已禁用")
}

// IsEnabled 检查是否启用
func (am *AlertManager) IsEnabled() bool {
	return am.enabled
}

// GetStatus 获取告警管理器状态
func (am *AlertManager) GetStatus(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"enabled":         am.enabled,
		"clients_count":   len(am.alertClients),
		"enabled_clients": 0,
	}

	enabledCount := 0
	for _, client := range am.alertClients {
		if client.IsEnabled() {
			enabledCount++
		}
	}
	status["enabled_clients"] = enabledCount

	// 添加客户端详情
	clientDetails := make(map[string]interface{})
	for name, client := range am.alertClients {
		clientDetails[name] = map[string]interface{}{
			"name":    client.GetName(),
			"enabled": client.IsEnabled(),
			"config":  client.GetConfig(),
		}
	}
	status["clients"] = clientDetails

	return status
}

// 私有方法

// loadConfig 加载配置
func (am *AlertManager) loadConfig() error {
	// 从环境变量或配置文件加载告警配置
	// 这里简化实现，实际项目中应该从配置文件加载

	am.logger.Info(context.Background(), "加载告警配置")
	return nil
}

// initAlertClients 初始化告警客户端
func (am *AlertManager) initAlertClients(ctx context.Context) error {
	am.logger.Info(ctx, "初始化告警客户端")

	// 初始化飞书告警客户端
	feishuClient := NewFeishuAlert(am.logger, am.config)
	am.RegisterAlertClient("feishu", feishuClient)

	// 初始化邮件告警客户端
	emailClient := NewEmailAlert(am.logger, am.config)
	am.RegisterAlertClient("email", emailClient)

	// 初始化Webhook告警客户端
	webhookClient := NewWebhookAlert(am.logger, am.config)
	am.RegisterAlertClient("webhook", webhookClient)

	am.logger.Info(ctx, "告警客户端初始化完成", "clients_count", len(am.alertClients))
	return nil
}

// generateAlertID 生成告警ID
func (am *AlertManager) generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// getDefaultPriority 获取默认优先级
func (am *AlertManager) getDefaultPriority(level AlertLevel) int {
	switch level {
	case AlertLevelInfo:
		return 1
	case AlertLevelWarning:
		return 2
	case AlertLevelError:
		return 3
	case AlertLevelCritical:
		return 4
	default:
		return 1
	}
}

// AlertConfig 告警配置
type AlertConfig struct {
	Enabled     bool                   `json:"enabled" yaml:"enabled"`
	Clients     map[string]interface{} `json:"clients" yaml:"clients"`
	DefaultTags []string               `json:"default_tags" yaml:"default_tags"`
	RateLimit   RateLimitConfig        `json:"rate_limit" yaml:"rate_limit"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled    bool          `json:"enabled" yaml:"enabled"`
	MaxPerMin  int           `json:"max_per_min" yaml:"max_per_min"`
	MaxPerHour int           `json:"max_per_hour" yaml:"max_per_hour"`
	Window     time.Duration `json:"window" yaml:"window"`
}

// LoadAlertConfigFromFile 从文件加载告警配置
func LoadAlertConfigFromFile(filePath string) (*AlertConfig, error) {
	// 这里应该实现从YAML文件加载配置的逻辑
	// 目前返回默认配置
	return &AlertConfig{
		Enabled:     true,
		Clients:     make(map[string]interface{}),
		DefaultTags: []string{"system"},
		RateLimit: RateLimitConfig{
			Enabled:    true,
			MaxPerMin:  60,
			MaxPerHour: 1000,
			Window:     time.Minute,
		},
	}, nil
}

// LoadAlertConfigFromEnv 从环境变量加载告警配置
func LoadAlertConfigFromEnv() *AlertConfig {
	return &AlertConfig{
		Enabled:     getEnvBool("ALERT_ENABLED", true),
		Clients:     make(map[string]interface{}),
		DefaultTags: getEnvSlice("ALERT_DEFAULT_TAGS", []string{"system"}),
		RateLimit: RateLimitConfig{
			Enabled:    getEnvBool("ALERT_RATE_LIMIT_ENABLED", true),
			MaxPerMin:  getEnvInt("ALERT_MAX_PER_MIN", 60),
			MaxPerHour: getEnvInt("ALERT_MAX_PER_HOUR", 1000),
			Window:     time.Duration(getEnvInt("ALERT_RATE_LIMIT_WINDOW_MINUTES", 1)) * time.Minute,
		},
	}
}

// 辅助函数
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := json.Unmarshal([]byte(value), new(int)); err == nil {
			return *intValue
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		var slice []string
		if err := json.Unmarshal([]byte(value), &slice); err == nil {
			return slice
		}
	}
	return defaultValue
}
