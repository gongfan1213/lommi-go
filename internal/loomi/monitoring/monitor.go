package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Monitor 监控器
type Monitor struct {
	config    *MonitorConfig
	alerts    []Alert
	health    *HealthStatus
	mu        sync.RWMutex
	alertChan chan Alert
	stopChan  chan struct{}
	logger    Logger
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Port             int             `json:"port"`
	CheckInterval    time.Duration   `json:"check_interval"`
	HealthCheckPath  string          `json:"health_check_path"`
	MetricsPath      string          `json:"metrics_path"`
	EnableAlerts     bool            `json:"enable_alerts"`
	AlertThresholds  AlertThresholds `json:"alert_thresholds"`
	FeishuWebhookURL string          `json:"feishu_webhook_url"`
	EmailConfig      EmailConfig     `json:"email_config"`
}

// AlertThresholds 告警阈值
type AlertThresholds struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	ResponseTimeMs     int64   `json:"response_time_ms"`
	ErrorRatePercent   float64 `json:"error_rate_percent"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTPHost  string   `json:"smtp_host"`
	SMTPPort  int      `json:"smtp_port"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	From      string   `json:"from"`
	To        []string `json:"to"`
	EnableTLS bool     `json:"enable_tls"`
}

// Alert 告警
type Alert struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Resolved  bool                   `json:"resolved"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Uptime      time.Duration          `json:"uptime"`
	SystemInfo  SystemInfo             `json:"system_info"`
	Services    map[string]ServiceInfo `json:"services"`
	LastChecked time.Time              `json:"last_checked"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	GoVersion    string  `json:"go_version"`
	NumGoroutine int     `json:"num_goroutine"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	LastChecked  time.Time `json:"last_checked"`
	ResponseTime int64     `json:"response_time"`
	Error        string    `json:"error,omitempty"`
}

// Logger 日志接口
type Logger interface {
	Info(ctx context.Context, message string, fields ...interface{})
	Error(ctx context.Context, message string, fields ...interface{})
	Warn(ctx context.Context, message string, fields ...interface{})
	Debug(ctx context.Context, message string, fields ...interface{})
}

// NewMonitor 创建新的监控器
func NewMonitor(config *MonitorConfig, logger Logger) *Monitor {
	return &Monitor{
		config:    config,
		alerts:    make([]Alert, 0),
		health:    &HealthStatus{},
		alertChan: make(chan Alert, 100),
		stopChan:  make(chan struct{}),
		logger:    logger,
	}
}

// Start 启动监控器
func (m *Monitor) Start(ctx context.Context) error {
	// 启动健康检查循环
	go m.healthCheckLoop(ctx)

	// 启动告警处理循环
	go m.alertProcessingLoop(ctx)

	// 启动HTTP服务器
	return m.startHTTPServer(ctx)
}

// Stop 停止监控器
func (m *Monitor) Stop() {
	close(m.stopChan)
}

// healthCheckLoop 健康检查循环
func (m *Monitor) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck 执行健康检查
func (m *Monitor) performHealthCheck(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新系统信息
	systemInfo := m.getSystemInfo()

	// 检查服务状态
	services := make(map[string]ServiceInfo)

	// 检查数据库连接
	dbInfo := m.checkDatabase(ctx)
	services["database"] = dbInfo

	// 检查Redis连接
	redisInfo := m.checkRedis(ctx)
	services["redis"] = redisInfo

	// 检查API服务
	apiInfo := m.checkAPI(ctx)
	services["api"] = apiInfo

	// 更新健康状态
	m.health = &HealthStatus{
		Status:      m.determineOverallStatus(services),
		Timestamp:   time.Now(),
		Version:     "1.0.0",                // 从配置获取
		Uptime:      time.Since(time.Now()), // 实际应该是启动时间
		SystemInfo:  systemInfo,
		Services:    services,
		LastChecked: time.Now(),
	}

	// 检查告警条件
	m.checkAlertConditions(systemInfo, services)
}

// getSystemInfo 获取系统信息
func (m *Monitor) getSystemInfo() SystemInfo {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	// 计算内存使用率
	memoryUsage := float64(mstat.Alloc) / float64(mstat.Sys) * 100

	return SystemInfo{
		CPUUsage:     0, // 需要额外的库来计算CPU使用率
		MemoryUsage:  memoryUsage,
		DiskUsage:    0, // 需要额外的库来计算磁盘使用率
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

// checkDatabase 检查数据库连接
func (m *Monitor) checkDatabase(ctx context.Context) ServiceInfo {
	start := time.Now()

	// 这里应该实际检查数据库连接
	// 现在只是模拟
	time.Sleep(10 * time.Millisecond)

	return ServiceInfo{
		Name:         "database",
		Status:       "healthy",
		LastChecked:  time.Now(),
		ResponseTime: time.Since(start).Milliseconds(),
	}
}

// checkRedis 检查Redis连接
func (m *Monitor) checkRedis(ctx context.Context) ServiceInfo {
	start := time.Now()

	// 这里应该实际检查Redis连接
	// 现在只是模拟
	time.Sleep(5 * time.Millisecond)

	return ServiceInfo{
		Name:         "redis",
		Status:       "healthy",
		LastChecked:  time.Now(),
		ResponseTime: time.Since(start).Milliseconds(),
	}
}

// checkAPI 检查API服务
func (m *Monitor) checkAPI(ctx context.Context) ServiceInfo {
	start := time.Now()

	// 检查API端点
	url := fmt.Sprintf("http://localhost:%d%s", m.config.Port, m.config.HealthCheckPath)
	resp, err := http.Get(url)

	if err != nil {
		return ServiceInfo{
			Name:         "api",
			Status:       "unhealthy",
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).Milliseconds(),
			Error:        err.Error(),
		}
	}
	defer resp.Body.Close()

	status := "healthy"
	if resp.StatusCode != http.StatusOK {
		status = "unhealthy"
	}

	return ServiceInfo{
		Name:         "api",
		Status:       status,
		LastChecked:  time.Now(),
		ResponseTime: time.Since(start).Milliseconds(),
	}
}

// determineOverallStatus 确定整体状态
func (m *Monitor) determineOverallStatus(services map[string]ServiceInfo) string {
	for _, service := range services {
		if service.Status != "healthy" {
			return "unhealthy"
		}
	}
	return "healthy"
}

// checkAlertConditions 检查告警条件
func (m *Monitor) checkAlertConditions(systemInfo SystemInfo, services map[string]ServiceInfo) {
	if !m.config.EnableAlerts {
		return
	}

	// 检查CPU使用率
	if systemInfo.CPUUsage > m.config.AlertThresholds.CPUUsagePercent {
		m.triggerAlert("high_cpu_usage", "warning",
			fmt.Sprintf("CPU使用率过高: %.2f%%", systemInfo.CPUUsage),
			map[string]interface{}{
				"cpu_usage": systemInfo.CPUUsage,
				"threshold": m.config.AlertThresholds.CPUUsagePercent,
			})
	}

	// 检查内存使用率
	if systemInfo.MemoryUsage > m.config.AlertThresholds.MemoryUsagePercent {
		m.triggerAlert("high_memory_usage", "warning",
			fmt.Sprintf("内存使用率过高: %.2f%%", systemInfo.MemoryUsage),
			map[string]interface{}{
				"memory_usage": systemInfo.MemoryUsage,
				"threshold":    m.config.AlertThresholds.MemoryUsagePercent,
			})
	}

	// 检查服务状态
	for name, service := range services {
		if service.Status != "healthy" {
			m.triggerAlert("service_unhealthy", "critical",
				fmt.Sprintf("服务 %s 不健康", name),
				map[string]interface{}{
					"service": name,
					"status":  service.Status,
					"error":   service.Error,
				})
		}
	}
}

// triggerAlert 触发告警
func (m *Monitor) triggerAlert(alertType, severity, message string, data map[string]interface{}) {
	alert := Alert{
		ID:        fmt.Sprintf("%s_%d", alertType, time.Now().Unix()),
		Type:      alertType,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data,
		Resolved:  false,
	}

	// 发送到告警通道
	select {
	case m.alertChan <- alert:
	default:
		// 通道满了，记录日志
		m.logger.Warn(context.Background(), "告警通道已满，丢弃告警", "alert_type", alertType)
	}
}

// alertProcessingLoop 告警处理循环
func (m *Monitor) alertProcessingLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case alert := <-m.alertChan:
			m.processAlert(ctx, alert)
		}
	}
}

// processAlert 处理告警
func (m *Monitor) processAlert(ctx context.Context, alert Alert) {
	// 添加到告警列表
	m.mu.Lock()
	m.alerts = append(m.alerts, alert)
	m.mu.Unlock()

	// 记录日志
	m.logger.Warn(ctx, "触发告警",
		"alert_id", alert.ID,
		"alert_type", alert.Type,
		"severity", alert.Severity,
		"message", alert.Message)

	// 发送告警通知
	if m.config.FeishuWebhookURL != "" {
		go m.sendFeishuAlert(alert)
	}

	if len(m.config.EmailConfig.To) > 0 {
		go m.sendEmailAlert(alert)
	}
}

// sendFeishuAlert 发送飞书告警
func (m *Monitor) sendFeishuAlert(alert Alert) {
	// 构建飞书消息
	message := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": fmt.Sprintf("🚨 系统告警\n类型: %s\n严重程度: %s\n消息: %s\n时间: %s",
				alert.Type, alert.Severity, alert.Message, alert.Timestamp.Format("2006-01-02 15:04:05")),
		},
	}

	// 发送HTTP请求
	jsonData, _ := json.Marshal(message)
	resp, err := http.Post(m.config.FeishuWebhookURL, "application/json",
		strings.NewReader(string(jsonData)))

	if err != nil {
		m.logger.Error(context.Background(), "发送飞书告警失败", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error(context.Background(), "飞书告警发送失败", "status_code", resp.StatusCode)
	}
}

// sendEmailAlert 发送邮件告警
func (m *Monitor) sendEmailAlert(alert Alert) {
	// 这里应该实现邮件发送逻辑
	// 可以使用第三方库如 gomail
	m.logger.Info(context.Background(), "发送邮件告警", "alert_id", alert.ID)
}

// startHTTPServer 启动HTTP服务器
func (m *Monitor) startHTTPServer(ctx context.Context) error {
	mux := http.NewServeMux()

	// 健康检查端点
	mux.HandleFunc(m.config.HealthCheckPath, m.handleHealthCheck)

	// 指标端点
	mux.HandleFunc(m.config.MetricsPath, m.handleMetrics)

	// 告警端点
	mux.HandleFunc("/alerts", m.handleAlerts)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.Port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// handleHealthCheck 处理健康检查请求
func (m *Monitor) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	health := m.health
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	if health.Status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}

// handleMetrics 处理指标请求
func (m *Monitor) handleMetrics(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	health := m.health
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleAlerts 处理告警请求
func (m *Monitor) handleAlerts(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	alerts := m.alerts
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// GetHealth 获取健康状态
func (m *Monitor) GetHealth() *HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health
}

// GetAlerts 获取告警列表
func (m *Monitor) GetAlerts() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.alerts
}

// ResolveAlert 解决告警
func (m *Monitor) ResolveAlert(alertID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.alerts {
		if m.alerts[i].ID == alertID {
			m.alerts[i].Resolved = true
			return true
		}
	}
	return false
}
