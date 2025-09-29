package api

import (
	"net/http"
	"strings"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	origins []string
	logger  *logx.Logger
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(cfg *config.APIConfig, logger *logx.Logger) *CORSMiddleware {
	return &CORSMiddleware{
		origins: cfg.CORSOrigins,
		logger:  logger,
	}
}

// CORS CORS中间件
func (cm *CORSMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查origin是否允许
		if cm.isOriginAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 设置其他CORS头
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-ID, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed 检查origin是否允许
func (cm *CORSMiddleware) isOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}

	for _, allowedOrigin := range cm.origins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}

		// 支持通配符匹配
		if strings.Contains(allowedOrigin, "*") {
			pattern := strings.ReplaceAll(allowedOrigin, "*", ".*")
			if strings.HasPrefix(pattern, ".*") || strings.HasSuffix(pattern, ".*") {
				// 简单的通配符匹配
				if strings.HasPrefix(allowedOrigin, "*") && strings.HasSuffix(origin, allowedOrigin[1:]) {
					return true
				}
				if strings.HasSuffix(allowedOrigin, "*") && strings.HasPrefix(origin, allowedOrigin[:len(allowedOrigin)-1]) {
					return true
				}
			}
		}
	}

	return false
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	logger *logx.Logger
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware(logger *logx.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// LogRequest 请求日志中间件
func (lm *LoggingMiddleware) LogRequest() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 使用结构化日志
		lm.logger.Info(param.TimeStamp, "HTTP请求",
			logx.KV("method", param.Method),
			logx.KV("path", param.Path),
			logx.KV("status", param.StatusCode),
			logx.KV("latency", param.Latency),
			logx.KV("client_ip", param.ClientIP),
			logx.KV("user_agent", param.Request.UserAgent()),
			logx.KV("body_size", param.BodySize),
			logx.KV("error", param.ErrorMessage))

		return ""
	})
}

// LogError 错误日志中间件
func (lm *LoggingMiddleware) LogError() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 记录错误
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				lm.logger.Error(c.Request.Context(), "请求处理错误",
					logx.KV("method", c.Request.Method),
					logx.KV("path", c.Request.URL.Path),
					logx.KV("error", err.Error()),
					logx.KV("error_type", err.Type),
					logx.KV("client_ip", c.ClientIP()))
			}
		}
	}
}

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	logger *logx.Logger
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(logger *logx.Logger) *SecurityMiddleware {
	return &SecurityMiddleware{
		logger: logger,
	}
}

// SecurityHeaders 安全头中间件
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

// RequestSizeLimit 请求大小限制中间件
type RequestSizeLimit struct {
	maxSize int64
	logger  *logx.Logger
}

// NewRequestSizeLimit 创建请求大小限制中间件
func NewRequestSizeLimit(maxSize int64, logger *logx.Logger) *RequestSizeLimit {
	return &RequestSizeLimit{
		maxSize: maxSize,
		logger:  logger,
	}
}

// LimitRequestSize 限制请求大小
func (rsl *RequestSizeLimit) LimitRequestSize() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > rsl.maxSize {
			rsl.logger.Warn(c.Request.Context(), "请求大小超出限制",
				logx.KV("content_length", c.Request.ContentLength),
				logx.KV("max_size", rsl.maxSize),
				logx.KV("path", c.Request.URL.Path))

			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "请求过大",
				"message": "请求大小超出限制",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TimeoutMiddleware 超时中间件
type TimeoutMiddleware struct {
	timeout int
	logger  *logx.Logger
}

// NewTimeoutMiddleware 创建超时中间件
func NewTimeoutMiddleware(timeout int, logger *logx.Logger) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		timeout: timeout,
		logger:  logger,
	}
}

// Timeout 超时中间件
func (tm *TimeoutMiddleware) Timeout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 这里应该使用context.WithTimeout，但由于gin的限制，
		// 实际的超时处理应该在路由层面实现
		c.Next()
	}
}

// RecoveryMiddleware 恢复中间件
type RecoveryMiddleware struct {
	logger *logx.Logger
}

// NewRecoveryMiddleware 创建恢复中间件
func NewRecoveryMiddleware(logger *logx.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
	}
}

// Recovery 恢复中间件
func (rm *RecoveryMiddleware) Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		rm.logger.Error(c.Request.Context(), "请求处理panic",
			logx.KV("error", recovered),
			logx.KV("method", c.Request.Method),
			logx.KV("path", c.Request.URL.Path),
			logx.KV("client_ip", c.ClientIP()))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "服务器内部错误",
			"message": "请求处理失败",
		})
	})
}
