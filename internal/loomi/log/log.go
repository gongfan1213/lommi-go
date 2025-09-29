package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 日志记录器
type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	warnLogger  *log.Logger
	debugLogger *log.Logger
	mu          sync.RWMutex
	config      *LogConfig
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"` // "text" or "json"
	MaxSize    int64  `json:"max_size"`
	MaxAge     int    `json:"max_age"`
	MaxBackups int    `json:"max_backups"`
	Compress   bool   `json:"compress"`
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// KV 键值对
type KV struct {
	Key   string
	Value interface{}
}

// NewLogger 创建新的日志记录器
func NewLogger(logDir string) *Logger {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(fmt.Sprintf("创建日志目录失败: %v", err))
	}

	// 创建日志文件
	infoFile := filepath.Join(logDir, "info.log")
	errorFile := filepath.Join(logDir, "error.log")
	warnFile := filepath.Join(logDir, "warn.log")
	debugFile := filepath.Join(logDir, "debug.log")

	// 打开日志文件
	infoWriter := openLogFile(infoFile)
	errorWriter := openLogFile(errorFile)
	warnWriter := openLogFile(warnFile)
	debugWriter := openLogFile(debugFile)

	// 创建日志记录器
	infoLogger := log.New(infoWriter, "[INFO] ", log.LstdFlags|log.Lshortfile)
	errorLogger := log.New(errorWriter, "[ERROR] ", log.LstdFlags|log.Lshortfile)
	warnLogger := log.New(warnWriter, "[WARN] ", log.LstdFlags|log.Lshortfile)
	debugLogger := log.New(debugWriter, "[DEBUG] ", log.LstdFlags|log.Lshortfile)

	return &Logger{
		infoLogger:  infoLogger,
		errorLogger: errorLogger,
		warnLogger:  warnLogger,
		debugLogger: debugLogger,
		config: &LogConfig{
			Level:  "INFO",
			Format: "text",
		},
	}
}

// openLogFile 打开日志文件
func openLogFile(filename string) io.Writer {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("打开日志文件失败: %v", err))
	}
	return file
}

// Info 记录信息日志
func (l *Logger) Info(ctx context.Context, message string, fields ...KV) {
	l.log(ctx, "INFO", message, fields...)
}

// Error 记录错误日志
func (l *Logger) Error(ctx context.Context, message string, fields ...KV) {
	l.log(ctx, "ERROR", message, fields...)
}

// Warn 记录警告日志
func (l *Logger) Warn(ctx context.Context, message string, fields ...KV) {
	l.log(ctx, "WARN", message, fields...)
}

// Debug 记录调试日志
func (l *Logger) Debug(ctx context.Context, message string, fields ...KV) {
	l.log(ctx, "DEBUG", message, fields...)
}

// log 记录日志
func (l *Logger) log(ctx context.Context, level, message string, fields ...KV) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 构建日志条目
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}

	// 从上下文获取信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry.RequestID = fmt.Sprintf("%v", requestID)
	}
	if userID := ctx.Value("user_id"); userID != nil {
		entry.UserID = fmt.Sprintf("%v", userID)
	}
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		entry.SessionID = fmt.Sprintf("%v", sessionID)
	}

	// 添加字段
	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}

	// 格式化日志
	var logMessage string
	if l.config.Format == "json" {
		logBytes, err := json.Marshal(entry)
		if err != nil {
			logMessage = fmt.Sprintf("日志序列化失败: %v", err)
		} else {
			logMessage = string(logBytes)
		}
	} else {
		// 文本格式
		logMessage = l.formatTextLog(entry)
	}

	// 写入日志
	switch level {
	case "INFO":
		l.infoLogger.Println(logMessage)
	case "ERROR":
		l.errorLogger.Println(logMessage)
	case "WARN":
		l.warnLogger.Println(logMessage)
	case "DEBUG":
		l.debugLogger.Println(logMessage)
	}
}

// formatTextLog 格式化文本日志
func (l *Logger) formatTextLog(entry LogEntry) string {
	message := fmt.Sprintf("%s [%s] %s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Level,
		entry.Message)

	if entry.RequestID != "" {
		message = fmt.Sprintf("[%s] %s", entry.RequestID, message)
	}

	if entry.UserID != "" {
		message = fmt.Sprintf("user:%s %s", entry.UserID, message)
	}

	if entry.SessionID != "" {
		message = fmt.Sprintf("session:%s %s", entry.SessionID, message)
	}

	// 添加字段
	if len(entry.Fields) > 0 {
		message += " |"
		for key, value := range entry.Fields {
			message += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	return message
}

// KV 创建键值对
func KV(key string, value interface{}) KV {
	return KV{Key: key, Value: value}
}

// SetConfig 设置日志配置
func (l *Logger) SetConfig(config *LogConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config = config
}

// GetConfig 获取日志配置
func (l *Logger) GetConfig() *LogConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// RequireIDFilter 要求ID过滤器
type RequireIDFilter struct {
	next io.Writer
}

// NewRequireIDFilter 创建要求ID过滤器
func NewRequireIDFilter(next io.Writer) *RequireIDFilter {
	return &RequireIDFilter{
		next: next,
	}
}

// Write 实现io.Writer接口
func (rif *RequireIDFilter) Write(p []byte) (n int, err error) {
	// 这里可以添加ID过滤逻辑
	return rif.next.Write(p)
}

// dateRotateWriter 日期轮转写入器
type dateRotateWriter struct {
	filename string
	file     *os.File
	lastDate string
	mu       sync.Mutex
}

// newDateRotateWriter 创建日期轮转写入器
func newDateRotateWriter(filename string) *dateRotateWriter {
	return &dateRotateWriter{
		filename: filename,
	}
}

// Write 实现io.Writer接口
func (drw *dateRotateWriter) Write(p []byte) (n int, err error) {
	drw.mu.Lock()
	defer drw.mu.Unlock()

	// 检查是否需要轮转
	currentDate := time.Now().Format("2006-01-02")
	if drw.lastDate != currentDate {
		// 关闭旧文件
		if drw.file != nil {
			drw.file.Close()
		}

		// 打开新文件
		newFilename := fmt.Sprintf("%s.%s", drw.filename, currentDate)
		drw.file, err = os.OpenFile(newFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return 0, err
		}

		drw.lastDate = currentDate
	}

	// 写入日志
	return drw.file.Write(p)
}

// Close 关闭写入器
func (drw *dateRotateWriter) Close() error {
	drw.mu.Lock()
	defer drw.mu.Unlock()

	if drw.file != nil {
		return drw.file.Close()
	}
	return nil
}

// 全局日志记录器实例
var (
	globalLogger     *Logger
	globalLoggerOnce sync.Once
)

// GetLogger 获取全局日志记录器
func GetLogger() *Logger {
	return globalLogger
}

// InitializeLogger 初始化全局日志记录器
func InitializeLogger(logDir string) *Logger {
	globalLoggerOnce.Do(func() {
		globalLogger = NewLogger(logDir)
	})

	return globalLogger
}

// SetGlobalLogger 设置全局日志记录器
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}
