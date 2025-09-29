package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PortMonitor 端口监控器
type PortMonitor struct {
	config   *PortMonitorConfig
	results  map[string]*PortStatus
	mu       sync.RWMutex
	stopChan chan struct{}
	logger   Logger
}

// PortMonitorConfig 端口监控配置
type PortMonitorConfig struct {
	CheckInterval time.Duration `json:"check_interval"`
	Ports         []PortConfig  `json:"ports"`
	Timeout       time.Duration `json:"timeout"`
	EnableAlerts  bool          `json:"enable_alerts"`
	AlertConfig   AlertConfig   `json:"alert_config"`
}

// PortConfig 端口配置
type PortConfig struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Description string `json:"description"`
	Critical    bool   `json:"critical"`
}

// AlertConfig 告警配置
type AlertConfig struct {
	FeishuWebhookURL string        `json:"feishu_webhook_url"`
	EmailConfig      EmailConfig   `json:"email_config"`
	RetryCount       int           `json:"retry_count"`
	RetryInterval    time.Duration `json:"retry_interval"`
}

// PortStatus 端口状态
type PortStatus struct {
	Port                PortConfig    `json:"port"`
	IsOpen              bool          `json:"is_open"`
	LastCheck           time.Time     `json:"last_check"`
	ResponseTime        time.Duration `json:"response_time"`
	Error               string        `json:"error,omitempty"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	LastSuccess         time.Time     `json:"last_success,omitempty"`
}

// NewPortMonitor 创建新的端口监控器
func NewPortMonitor(config *PortMonitorConfig, logger Logger) *PortMonitor {
	return &PortMonitor{
		config:   config,
		results:  make(map[string]*PortStatus),
		stopChan: make(chan struct{}),
		logger:   logger,
	}
}

// Start 启动端口监控
func (pm *PortMonitor) Start(ctx context.Context) error {
	// 初始化所有端口状态
	for _, port := range pm.config.Ports {
		pm.results[port.Name] = &PortStatus{
			Port:                port,
			IsOpen:              false,
			LastCheck:           time.Time{},
			ResponseTime:        0,
			ConsecutiveFailures: 0,
		}
	}

	// 启动监控循环
	go pm.monitorLoop(ctx)

	return nil
}

// Stop 停止端口监控
func (pm *PortMonitor) Stop() {
	close(pm.stopChan)
}

// monitorLoop 监控循环
func (pm *PortMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(pm.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.checkAllPorts(ctx)
		}
	}
}

// checkAllPorts 检查所有端口
func (pm *PortMonitor) checkAllPorts(ctx context.Context) {
	var wg sync.WaitGroup

	for _, port := range pm.config.Ports {
		wg.Add(1)
		go func(port PortConfig) {
			defer wg.Done()
			pm.checkPort(ctx, port)
		}(port)
	}

	wg.Wait()
}

// checkPort 检查单个端口
func (pm *PortMonitor) checkPort(ctx context.Context, port PortConfig) {
	start := time.Now()

	// 尝试连接端口
	address := fmt.Sprintf("%s:%d", port.Host, port.Port)
	conn, err := net.DialTimeout("tcp", address, pm.config.Timeout)
	responseTime := time.Since(start)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	status := pm.results[port.Name]
	status.LastCheck = time.Now()
	status.ResponseTime = responseTime

	if err != nil {
		// 端口不可用
		status.IsOpen = false
		status.Error = err.Error()
		status.ConsecutiveFailures++

		pm.logger.Warn(ctx, "端口检查失败",
			"port_name", port.Name,
			"host", port.Host,
			"port", port.Port,
			"error", err.Error(),
			"consecutive_failures", status.ConsecutiveFailures)

		// 检查是否需要发送告警
		if pm.config.EnableAlerts && port.Critical && status.ConsecutiveFailures >= pm.config.AlertConfig.RetryCount {
			go pm.sendPortDownAlert(ctx, status)
		}
	} else {
		// 端口可用
		conn.Close()
		status.IsOpen = true
		status.Error = ""

		// 如果之前失败，现在恢复，发送恢复告警
		if status.ConsecutiveFailures > 0 {
			pm.logger.Info(ctx, "端口恢复可用",
				"port_name", port.Name,
				"host", port.Host,
				"port", port.Port,
				"response_time", responseTime)

			if pm.config.EnableAlerts && port.Critical {
				go pm.sendPortUpAlert(ctx, status)
			}
		}

		status.ConsecutiveFailures = 0
		status.LastSuccess = time.Now()
	}
}

// sendPortDownAlert 发送端口宕机告警
func (pm *PortMonitor) sendPortDownAlert(ctx context.Context, status *PortStatus) {
	message := fmt.Sprintf("🚨 端口宕机告警\n端口: %s (%s:%d)\n描述: %s\n连续失败次数: %d\n最后检查时间: %s",
		status.Port.Name,
		status.Port.Host,
		status.Port.Port,
		status.Port.Description,
		status.ConsecutiveFailures,
		status.LastCheck.Format("2006-01-02 15:04:05"))

	// 发送飞书告警
	if pm.config.AlertConfig.FeishuWebhookURL != "" {
		go pm.sendFeishuAlert(message)
	}

	// 发送邮件告警
	if len(pm.config.AlertConfig.EmailConfig.To) > 0 {
		go pm.sendEmailAlert("端口宕机告警", message)
	}

	pm.logger.Error(ctx, "发送端口宕机告警",
		"port_name", status.Port.Name,
		"consecutive_failures", status.ConsecutiveFailures)
}

// sendPortUpAlert 发送端口恢复告警
func (pm *PortMonitor) sendPortUpAlert(ctx context.Context, status *PortStatus) {
	message := fmt.Sprintf("✅ 端口恢复告警\n端口: %s (%s:%d)\n描述: %s\n响应时间: %v\n恢复时间: %s",
		status.Port.Name,
		status.Port.Host,
		status.Port.Port,
		status.Port.Description,
		status.ResponseTime,
		status.LastCheck.Format("2006-01-02 15:04:05"))

	// 发送飞书告警
	if pm.config.AlertConfig.FeishuWebhookURL != "" {
		go pm.sendFeishuAlert(message)
	}

	// 发送邮件告警
	if len(pm.config.AlertConfig.EmailConfig.To) > 0 {
		go pm.sendEmailAlert("端口恢复告警", message)
	}

	pm.logger.Info(ctx, "发送端口恢复告警",
		"port_name", status.Port.Name,
		"response_time", status.ResponseTime)
}

// sendFeishuAlert 发送飞书告警
func (pm *PortMonitor) sendFeishuAlert(message string) {
	// 构建飞书消息
	feishuMessage := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": message,
		},
	}

	// 发送HTTP请求
	jsonData, _ := json.Marshal(feishuMessage)
	resp, err := http.Post(pm.config.AlertConfig.FeishuWebhookURL, "application/json",
		strings.NewReader(string(jsonData)))

	if err != nil {
		pm.logger.Error(context.Background(), "发送飞书告警失败", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pm.logger.Error(context.Background(), "飞书告警发送失败", "status_code", resp.StatusCode)
	}
}

// sendEmailAlert 发送邮件告警
func (pm *PortMonitor) sendEmailAlert(subject, message string) {
	// 这里应该实现邮件发送逻辑
	// 可以使用第三方库如 gomail
	pm.logger.Info(context.Background(), "发送邮件告警", "subject", subject)
}

// GetStatus 获取端口状态
func (pm *PortMonitor) GetStatus(portName string) (*PortStatus, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status, exists := pm.results[portName]
	return status, exists
}

// GetAllStatuses 获取所有端口状态
func (pm *PortMonitor) GetAllStatuses() map[string]*PortStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 创建副本
	result := make(map[string]*PortStatus)
	for name, status := range pm.results {
		result[name] = status
	}
	return result
}

// GetHealthyPorts 获取健康的端口
func (pm *PortMonitor) GetHealthyPorts() []*PortStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var healthy []*PortStatus
	for _, status := range pm.results {
		if status.IsOpen {
			healthy = append(healthy, status)
		}
	}
	return healthy
}

// GetUnhealthyPorts 获取不健康的端口
func (pm *PortMonitor) GetUnhealthyPorts() []*PortStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var unhealthy []*PortStatus
	for _, status := range pm.results {
		if !status.IsOpen {
			unhealthy = append(unhealthy, status)
		}
	}
	return unhealthy
}

// GetCriticalUnhealthyPorts 获取关键不健康端口
func (pm *PortMonitor) GetCriticalUnhealthyPorts() []*PortStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var critical []*PortStatus
	for _, status := range pm.results {
		if !status.IsOpen && status.Port.Critical {
			critical = append(critical, status)
		}
	}
	return critical
}

// AddPort 添加端口监控
func (pm *PortMonitor) AddPort(port PortConfig) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.results[port.Name] = &PortStatus{
		Port:                port,
		IsOpen:              false,
		LastCheck:           time.Time{},
		ResponseTime:        0,
		ConsecutiveFailures: 0,
	}
}

// RemovePort 移除端口监控
func (pm *PortMonitor) RemovePort(portName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.results, portName)
}

// UpdatePort 更新端口配置
func (pm *PortMonitor) UpdatePort(port PortConfig) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if status, exists := pm.results[port.Name]; exists {
		status.Port = port
	}
}

// GetSummary 获取监控摘要
func (pm *PortMonitor) GetSummary() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	total := len(pm.results)
	healthy := 0
	unhealthy := 0
	critical := 0
	criticalUnhealthy := 0

	for _, status := range pm.results {
		if status.IsOpen {
			healthy++
		} else {
			unhealthy++
		}

		if status.Port.Critical {
			critical++
			if !status.IsOpen {
				criticalUnhealthy++
			}
		}
	}

	return map[string]interface{}{
		"total_ports":        total,
		"healthy_ports":      healthy,
		"unhealthy_ports":    unhealthy,
		"critical_ports":     critical,
		"critical_unhealthy": criticalUnhealthy,
		"health_percentage":  float64(healthy) / float64(total) * 100,
		"last_check":         time.Now(),
	}
}
