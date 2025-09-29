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

// SystemMonitor 系统监控器
type SystemMonitor struct {
	config   *SystemMonitorConfig
	metrics  *SystemMetrics
	mu       sync.RWMutex
	stopChan chan struct{}
	logger   Logger
}

// SystemMonitorConfig 系统监控配置
type SystemMonitorConfig struct {
	CheckInterval    time.Duration   `json:"check_interval"`
	EnableAlerts     bool            `json:"enable_alerts"`
	AlertThresholds  AlertThresholds `json:"alert_thresholds"`
	FeishuWebhookURL string          `json:"feishu_webhook_url"`
	EmailConfig      EmailConfig     `json:"email_config"`
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	MemoryTotal   uint64    `json:"memory_total"`
	MemoryAlloc   uint64    `json:"memory_alloc"`
	MemorySys     uint64    `json:"memory_sys"`
	MemoryHeap    uint64    `json:"memory_heap"`
	MemoryStack   uint64    `json:"memory_stack"`
	GCPauseTotal  uint64    `json:"gc_pause_total"`
	GCPauseCount  uint32    `json:"gc_pause_count"`
	NumGoroutines int       `json:"num_goroutines"`
	NumCGoCalls   int64     `json:"num_cgo_calls"`
	GoVersion     string    `json:"go_version"`
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
	NumCPU        int       `json:"num_cpu"`
}

// NewSystemMonitor 创建新的系统监控器
func NewSystemMonitor(config *SystemMonitorConfig, logger Logger) *SystemMonitor {
	return &SystemMonitor{
		config:   config,
		metrics:  &SystemMetrics{},
		stopChan: make(chan struct{}),
		logger:   logger,
	}
}

// Start 启动系统监控
func (sm *SystemMonitor) Start(ctx context.Context) error {
	// 启动监控循环
	go sm.monitorLoop(ctx)

	return nil
}

// Stop 停止系统监控
func (sm *SystemMonitor) Stop() {
	close(sm.stopChan)
}

// monitorLoop 监控循环
func (sm *SystemMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(sm.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.collectMetrics(ctx)
		}
	}
}

// collectMetrics 收集系统指标
func (sm *SystemMonitor) collectMetrics(ctx context.Context) {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 更新指标
	sm.metrics = &SystemMetrics{
		Timestamp:     time.Now(),
		CPUUsage:      sm.calculateCPUUsage(), // 需要额外的库来计算CPU使用率
		MemoryUsage:   float64(mstat.Alloc) / float64(mstat.Sys) * 100,
		MemoryTotal:   mstat.Sys,
		MemoryAlloc:   mstat.Alloc,
		MemorySys:     mstat.Sys,
		MemoryHeap:    mstat.HeapAlloc,
		MemoryStack:   mstat.StackInuse,
		GCPauseTotal:  mstat.PauseTotalNs,
		GCPauseCount:  mstat.NumGC,
		NumGoroutines: runtime.NumGoroutine(),
		NumCGoCalls:   runtime.NumCgoCall(),
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		NumCPU:        runtime.NumCPU(),
	}

	// 检查告警条件
	if sm.config.EnableAlerts {
		sm.checkAlertConditions(ctx)
	}

	// 记录日志
	sm.logger.Debug(ctx, "收集系统指标",
		"cpu_usage", sm.metrics.CPUUsage,
		"memory_usage", sm.metrics.MemoryUsage,
		"num_goroutines", sm.metrics.NumGoroutines,
		"gc_pause_count", sm.metrics.GCPauseCount)
}

// calculateCPUUsage 计算CPU使用率
func (sm *SystemMonitor) calculateCPUUsage() float64 {
	// 这里应该使用额外的库来计算CPU使用率
	// 比如 github.com/shirou/gopsutil
	// 现在返回0作为占位符
	return 0.0
}

// checkAlertConditions 检查告警条件
func (sm *SystemMonitor) checkAlertConditions(ctx context.Context) {
	// 检查内存使用率
	if sm.metrics.MemoryUsage > sm.config.AlertThresholds.MemoryUsagePercent {
		sm.triggerAlert(ctx, "high_memory_usage", "warning",
			fmt.Sprintf("内存使用率过高: %.2f%%", sm.metrics.MemoryUsage),
			map[string]interface{}{
				"memory_usage": sm.metrics.MemoryUsage,
				"threshold":    sm.config.AlertThresholds.MemoryUsagePercent,
				"memory_alloc": sm.metrics.MemoryAlloc,
				"memory_total": sm.metrics.MemoryTotal,
			})
	}

	// 检查CPU使用率
	if sm.metrics.CPUUsage > sm.config.AlertThresholds.CPUUsagePercent {
		sm.triggerAlert(ctx, "high_cpu_usage", "warning",
			fmt.Sprintf("CPU使用率过高: %.2f%%", sm.metrics.CPUUsage),
			map[string]interface{}{
				"cpu_usage": sm.metrics.CPUUsage,
				"threshold": sm.config.AlertThresholds.CPUUsagePercent,
				"num_cpu":   sm.metrics.NumCPU,
			})
	}

	// 检查Goroutine数量
	if sm.metrics.NumGoroutines > 10000 { // 可配置的阈值
		sm.triggerAlert(ctx, "high_goroutine_count", "warning",
			fmt.Sprintf("Goroutine数量过多: %d", sm.metrics.NumGoroutines),
			map[string]interface{}{
				"num_goroutines": sm.metrics.NumGoroutines,
				"threshold":      10000,
			})
	}

	// 检查GC暂停时间
	if sm.metrics.GCPauseCount > 0 {
		avgPauseTime := float64(sm.metrics.GCPauseTotal) / float64(sm.metrics.GCPauseCount) / 1e6 // 转换为毫秒
		if avgPauseTime > 100 {                                                                   // 100ms阈值
			sm.triggerAlert(ctx, "high_gc_pause_time", "warning",
				fmt.Sprintf("GC暂停时间过长: %.2fms", avgPauseTime),
				map[string]interface{}{
					"avg_pause_time": avgPauseTime,
					"pause_count":    sm.metrics.GCPauseCount,
					"threshold":      100,
				})
		}
	}
}

// triggerAlert 触发告警
func (sm *SystemMonitor) triggerAlert(ctx context.Context, alertType, severity, message string, data map[string]interface{}) {
	sm.logger.Warn(ctx, "触发系统告警",
		"alert_type", alertType,
		"severity", severity,
		"message", message,
		"data", data)

	// 发送飞书告警
	if sm.config.FeishuWebhookURL != "" {
		go sm.sendFeishuAlert(alertType, severity, message, data)
	}

	// 发送邮件告警
	if len(sm.config.EmailConfig.To) > 0 {
		go sm.sendEmailAlert(alertType, severity, message, data)
	}
}

// sendFeishuAlert 发送飞书告警
func (sm *SystemMonitor) sendFeishuAlert(alertType, severity, message string, data map[string]interface{}) {
	// 构建飞书消息
	feishuMessage := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": fmt.Sprintf("🚨 系统告警\n类型: %s\n严重程度: %s\n消息: %s\n时间: %s\n数据: %v",
				alertType, severity, message, time.Now().Format("2006-01-02 15:04:05"), data),
		},
	}

	// 发送HTTP请求
	jsonData, _ := json.Marshal(feishuMessage)
	resp, err := http.Post(sm.config.FeishuWebhookURL, "application/json",
		strings.NewReader(string(jsonData)))

	if err != nil {
		sm.logger.Error(context.Background(), "发送飞书告警失败", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sm.logger.Error(context.Background(), "飞书告警发送失败", "status_code", resp.StatusCode)
	}
}

// sendEmailAlert 发送邮件告警
func (sm *SystemMonitor) sendEmailAlert(alertType, severity, message string, data map[string]interface{}) {
	// 这里应该实现邮件发送逻辑
	// 可以使用第三方库如 gomail
	sm.logger.Info(context.Background(), "发送邮件告警",
		"alert_type", alertType,
		"severity", severity,
		"message", message)
}

// GetMetrics 获取系统指标
func (sm *SystemMonitor) GetMetrics() *SystemMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 返回副本
	return &SystemMetrics{
		Timestamp:     sm.metrics.Timestamp,
		CPUUsage:      sm.metrics.CPUUsage,
		MemoryUsage:   sm.metrics.MemoryUsage,
		MemoryTotal:   sm.metrics.MemoryTotal,
		MemoryAlloc:   sm.metrics.MemoryAlloc,
		MemorySys:     sm.metrics.MemorySys,
		MemoryHeap:    sm.metrics.MemoryHeap,
		MemoryStack:   sm.metrics.MemoryStack,
		GCPauseTotal:  sm.metrics.GCPauseTotal,
		GCPauseCount:  sm.metrics.GCPauseCount,
		NumGoroutines: sm.metrics.NumGoroutines,
		NumCGoCalls:   sm.metrics.NumCGoCalls,
		GoVersion:     sm.metrics.GoVersion,
		OS:            sm.metrics.OS,
		Arch:          sm.metrics.Arch,
		NumCPU:        sm.metrics.NumCPU,
	}
}

// GetMemoryInfo 获取内存信息
func (sm *SystemMonitor) GetMemoryInfo() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"usage_percent": sm.metrics.MemoryUsage,
		"total_bytes":   sm.metrics.MemoryTotal,
		"alloc_bytes":   sm.metrics.MemoryAlloc,
		"sys_bytes":     sm.metrics.MemorySys,
		"heap_bytes":    sm.metrics.MemoryHeap,
		"stack_bytes":   sm.metrics.MemoryStack,
	}
}

// GetGoroutineInfo 获取Goroutine信息
func (sm *SystemMonitor) GetGoroutineInfo() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"count":     sm.metrics.NumGoroutines,
		"cgo_calls": sm.metrics.NumCGoCalls,
	}
}

// GetGCInfo 获取GC信息
func (sm *SystemMonitor) GetGCInfo() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	avgPauseTime := float64(0)
	if sm.metrics.GCPauseCount > 0 {
		avgPauseTime = float64(sm.metrics.GCPauseTotal) / float64(sm.metrics.GCPauseCount) / 1e6 // 转换为毫秒
	}

	return map[string]interface{}{
		"pause_count":    sm.metrics.GCPauseCount,
		"pause_total_ns": sm.metrics.GCPauseTotal,
		"avg_pause_ms":   avgPauseTime,
	}
}

// GetSystemInfo 获取系统信息
func (sm *SystemMonitor) GetSystemInfo() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"go_version": sm.metrics.GoVersion,
		"os":         sm.metrics.OS,
		"arch":       sm.metrics.Arch,
		"num_cpu":    sm.metrics.NumCPU,
		"cpu_usage":  sm.metrics.CPUUsage,
	}
}

// GetSummary 获取监控摘要
func (sm *SystemMonitor) GetSummary() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"timestamp":      sm.metrics.Timestamp,
		"cpu_usage":      sm.metrics.CPUUsage,
		"memory_usage":   sm.metrics.MemoryUsage,
		"num_goroutines": sm.metrics.NumGoroutines,
		"gc_pause_count": sm.metrics.GCPauseCount,
		"go_version":     sm.metrics.GoVersion,
		"os":             sm.metrics.OS,
		"arch":           sm.metrics.Arch,
		"num_cpu":        sm.metrics.NumCPU,
	}
}

// ForceGC 强制垃圾回收
func (sm *SystemMonitor) ForceGC() {
	runtime.GC()
	sm.logger.Info(context.Background(), "强制垃圾回收完成")
}

// ReadMemStats 读取内存统计
func (sm *SystemMonitor) ReadMemStats() *runtime.MemStats {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)
	return &mstat
}
