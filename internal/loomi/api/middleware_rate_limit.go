package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/utils"
	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware 速率限制中间件
type RateLimitMiddleware struct {
	rateLimiter utils.RateLimiter
	logger      *logx.Logger
	limiterName string
}

// NewRateLimitMiddleware 创建速率限制中间件
func NewRateLimitMiddleware(rateLimiter utils.RateLimiter, limiterName string, logger *logx.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		rateLimiter: rateLimiter,
		logger:      logger,
		limiterName: limiterName,
	}
}

// RateLimit 速率限制中间件
func (rlm *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成限流key
		key := rlm.generateRateLimitKey(c)

		// 检查是否允许请求
		allowed, err := rlm.rateLimiter.Allow(c.Request.Context(), key)
		if err != nil {
			rlm.logger.Error(c.Request.Context(), "速率限制检查失败",
				logx.KV("error", err),
				logx.KV("limiter_name", rlm.limiterName),
				logx.KV("key", key))

			// 发生错误时允许请求通过，避免影响正常服务
			c.Next()
			return
		}

		if !allowed {
			// 获取限制信息
			limitInfo, err := rlm.rateLimiter.GetLimitInfo(c.Request.Context(), key)
			if err != nil {
				rlm.logger.Error(c.Request.Context(), "获取速率限制信息失败",
					logx.KV("error", err),
					logx.KV("limiter_name", rlm.limiterName),
					logx.KV("key", key))
			}

			// 设置速率限制响应头
			if limitInfo != nil {
				c.Header("X-RateLimit-Limit", strconv.Itoa(limitInfo.Limit))
				c.Header("X-RateLimit-Remaining", strconv.Itoa(limitInfo.Remaining))
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limitInfo.ResetTime.Unix()))
				c.Header("Retry-After", fmt.Sprintf("%.0f", time.Until(limitInfo.ResetTime).Seconds()))
			}

			rlm.logger.Warn(c.Request.Context(), "请求被速率限制",
				logx.KV("limiter_name", rlm.limiterName),
				logx.KV("key", key),
				logx.KV("path", c.Request.URL.Path),
				logx.KV("method", c.Request.Method),
				logx.KV("client_ip", c.ClientIP()))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "请求过于频繁",
				"message":     "请稍后再试",
				"retry_after": limitInfo.ResetTime.Unix(),
				"limit":       limitInfo.Limit,
				"remaining":   limitInfo.Remaining,
			})
			c.Abort()
			return
		}

		// 获取限制信息并设置响应头
		limitInfo, err := rlm.rateLimiter.GetLimitInfo(c.Request.Context(), key)
		if err == nil && limitInfo != nil {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limitInfo.Limit))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(limitInfo.Remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limitInfo.ResetTime.Unix()))
		}

		c.Next()
	}
}

// generateRateLimitKey 生成速率限制key
func (rlm *RateLimitMiddleware) generateRateLimitKey(c *gin.Context) string {
	// 优先使用用户ID
	if userID := c.GetString("user_id"); userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}

	// 其次使用IP地址
	clientIP := c.ClientIP()
	if clientIP != "" {
		return fmt.Sprintf("ip:%s", clientIP)
	}

	// 最后使用默认key
	return "default"
}

// GetRateLimiter 获取速率限制器中间件
func GetRateLimiter(limiterName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从全局速率限制器管理器获取限流器
		manager := utils.GetRateLimiterManager()
		if manager == nil {
			c.Next()
			return
		}

		rateLimiter, exists := manager.GetRateLimiter(limiterName)
		if !exists {
			c.Next()
			return
		}

		// 创建速率限制中间件并执行
		middleware := NewRateLimitMiddleware(rateLimiter, limiterName, logx.GetLogger())
		middleware.RateLimit()(c)
	}
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	Window      time.Duration             `json:"window"`
	Limit       int                       `json:"limit"`
	LimiterName string                    `json:"limiter_name"`
	KeyFunc     func(*gin.Context) string `json:"-"`
}

// CustomRateLimit 自定义速率限制中间件
func CustomRateLimit(config RateLimitConfig, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取速率限制器
		manager := utils.GetRateLimiterManager()
		if manager == nil {
			c.Next()
			return
		}

		rateLimiter, exists := manager.GetRateLimiter(config.LimiterName)
		if !exists {
			c.Next()
			return
		}

		// 生成限流key
		var key string
		if config.KeyFunc != nil {
			key = config.KeyFunc(c)
		} else {
			// 默认key生成逻辑
			if userID := c.GetString("user_id"); userID != "" {
				key = fmt.Sprintf("user:%s", userID)
			} else {
				key = fmt.Sprintf("ip:%s", c.ClientIP())
			}
		}

		// 检查是否允许请求
		allowed, err := rateLimiter.Allow(c.Request.Context(), key)
		if err != nil {
			logger.Error(c.Request.Context(), "速率限制检查失败",
				logx.KV("error", err),
				logx.KV("limiter_name", config.LimiterName),
				logx.KV("key", key))
			c.Next()
			return
		}

		if !allowed {
			// 获取限制信息
			limitInfo, err := rateLimiter.GetLimitInfo(c.Request.Context(), key)
			if err != nil {
				logger.Error(c.Request.Context(), "获取速率限制信息失败",
					logx.KV("error", err),
					logx.KV("limiter_name", config.LimiterName),
					logx.KV("key", key))
			}

			// 设置响应头
			if limitInfo != nil {
				c.Header("X-RateLimit-Limit", strconv.Itoa(limitInfo.Limit))
				c.Header("X-RateLimit-Remaining", strconv.Itoa(limitInfo.Remaining))
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limitInfo.ResetTime.Unix()))
				c.Header("Retry-After", fmt.Sprintf("%.0f", time.Until(limitInfo.ResetTime).Seconds()))
			}

			logger.Warn(c.Request.Context(), "请求被速率限制",
				logx.KV("limiter_name", config.LimiterName),
				logx.KV("key", key),
				logx.KV("path", c.Request.URL.Path),
				logx.KV("method", c.Request.Method))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "请求过于频繁",
				"message":     "请稍后再试",
				"retry_after": limitInfo.ResetTime.Unix(),
				"limit":       limitInfo.Limit,
				"remaining":   limitInfo.Remaining,
			})
			c.Abort()
			return
		}

		// 设置成功响应头
		limitInfo, err := rateLimiter.GetLimitInfo(c.Request.Context(), key)
		if err == nil && limitInfo != nil {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limitInfo.Limit))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(limitInfo.Remaining))
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limitInfo.ResetTime.Unix()))
		}

		c.Next()
	}
}

// RateLimitByUser 按用户限制速率
func RateLimitByUser(limiterName string) gin.HandlerFunc {
	return CustomRateLimit(RateLimitConfig{
		LimiterName: limiterName,
		KeyFunc: func(c *gin.Context) string {
			if userID := c.GetString("user_id"); userID != "" {
				return fmt.Sprintf("user:%s", userID)
			}
			return fmt.Sprintf("ip:%s", c.ClientIP())
		},
	}, logx.GetLogger())
}

// RateLimitByIP 按IP限制速率
func RateLimitByIP(limiterName string) gin.HandlerFunc {
	return CustomRateLimit(RateLimitConfig{
		LimiterName: limiterName,
		KeyFunc: func(c *gin.Context) string {
			return fmt.Sprintf("ip:%s", c.ClientIP())
		},
	}, logx.GetLogger())
}

// RateLimitByEndpoint 按端点限制速率
func RateLimitByEndpoint(limiterName string) gin.HandlerFunc {
	return CustomRateLimit(RateLimitConfig{
		LimiterName: limiterName,
		KeyFunc: func(c *gin.Context) string {
			return fmt.Sprintf("endpoint:%s:%s", c.Request.Method, c.Request.URL.Path)
		},
	}, logx.GetLogger())
}

// RateLimitByUserAndEndpoint 按用户和端点限制速率
func RateLimitByUserAndEndpoint(limiterName string) gin.HandlerFunc {
	return CustomRateLimit(RateLimitConfig{
		LimiterName: limiterName,
		KeyFunc: func(c *gin.Context) string {
			userID := c.GetString("user_id")
			if userID == "" {
				userID = c.ClientIP()
			}
			return fmt.Sprintf("user_endpoint:%s:%s:%s", userID, c.Request.Method, c.Request.URL.Path)
		},
	}, logx.GetLogger())
}
