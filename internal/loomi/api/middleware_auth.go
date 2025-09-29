package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	jwtSecret      string
	whitelistPaths []string
	logger         *logx.Logger
	config         *config.SecurityConfig
}

// JWTClaims JWT声明
type JWTClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	jwt.RegisteredClaims
}

// AuthRequest 认证请求
type AuthRequest struct {
	Token string `json:"token" binding:"required"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Valid   bool   `json:"valid"`
	UserID  string `json:"user_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(cfg *config.SecurityConfig, logger *logx.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:      cfg.JWTSecretKey,
		whitelistPaths: cfg.Whitelist,
		logger:         logger,
		config:         cfg,
	}
}

// RequireAuth 要求认证的中间件
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否在白名单中
		if am.isWhitelisted(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 检查是否启用认证
		if !am.config.EnableAuth {
			c.Next()
			return
		}

		// 从请求头获取token
		token := am.extractToken(c)
		if token == "" {
			am.logger.Warn(c.Request.Context(), "未提供认证token",
				logx.KV("path", c.Request.URL.Path),
				logx.KV("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "未授权",
				"message": "请提供有效的认证token",
			})
			c.Abort()
			return
		}

		// 验证token
		claims, err := am.validateToken(token)
		if err != nil {
			am.logger.Warn(c.Request.Context(), "token验证失败",
				logx.KV("error", err),
				logx.KV("path", c.Request.URL.Path),
				logx.KV("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "认证失败",
				"message": "无效的token",
			})
			c.Abort()
			return
		}

		// 将用户信息添加到上下文
		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Set("role", claims.Role)

		am.logger.Info(c.Request.Context(), "用户认证成功",
			logx.KV("user_id", claims.UserID),
			logx.KV("session_id", claims.SessionID),
			logx.KV("role", claims.Role))

		c.Next()
	}
}

// RequireRole 要求特定角色的中间件
func (am *AuthMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先检查认证
		am.RequireAuth()(c)
		if c.IsAborted() {
			return
		}

		// 检查角色
		userRole, exists := c.Get("role")
		if !exists || userRole != requiredRole {
			am.logger.Warn(c.Request.Context(), "用户角色不足",
				logx.KV("user_id", c.GetString("user_id")),
				logx.KV("required_role", requiredRole),
				logx.KV("user_role", userRole))
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": fmt.Sprintf("需要角色: %s", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireID 要求ID的中间件
func (am *AuthMiddleware) RequireID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取ID
		id := c.GetHeader("X-User-ID")
		if id == "" {
			id = c.GetHeader("X-Request-ID")
		}
		if id == "" {
			// 从查询参数获取ID
			id = c.Query("user_id")
		}

		if id == "" {
			am.logger.Warn(c.Request.Context(), "未提供用户ID",
				logx.KV("path", c.Request.URL.Path),
				logx.KV("ip", c.ClientIP()))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "缺少用户ID",
				"message": "请提供用户ID",
			})
			c.Abort()
			return
		}

		// 将ID添加到上下文
		c.Set("user_id", id)

		am.logger.Info(c.Request.Context(), "用户ID验证成功",
			logx.KV("user_id", id),
			logx.KV("path", c.Request.URL.Path))

		c.Next()
	}
}

// ValidateToken 验证token端点
func (am *AuthMiddleware) ValidateToken(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求格式错误",
			"message": err.Error(),
		})
		return
	}

	claims, err := am.validateToken(req.Token)
	if err != nil {
		c.JSON(http.StatusOK, AuthResponse{
			Valid:   false,
			Message: "无效的token",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Valid:  true,
		UserID: claims.UserID,
	})
}

// GenerateToken 生成token
func (am *AuthMiddleware) GenerateToken(userID, sessionID, role string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(am.config.AccessTokenExpireMinutes) * time.Minute)

	claims := JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "loomi-api",
			Subject:   userID,
			Audience:  []string{"loomi-client"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(am.jwtSecret))
}

// RefreshToken 刷新token
func (am *AuthMiddleware) RefreshToken(c *gin.Context) {
	// 从当前请求获取用户信息
	userID := c.GetString("user_id")
	sessionID := c.GetString("session_id")
	role := c.GetString("role")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "未认证",
			"message": "请先进行认证",
		})
		return
	}

	// 生成新token
	newToken, err := am.GenerateToken(userID, sessionID, role)
	if err != nil {
		am.logger.Error(c.Request.Context(), "生成新token失败",
			logx.KV("user_id", userID),
			logx.KV("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "服务器错误",
			"message": "生成token失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      newToken,
		"expires_in": am.config.AccessTokenExpireMinutes * 60,
	})
}

// extractToken 从请求中提取token
func (am *AuthMiddleware) extractToken(c *gin.Context) string {
	// 从Authorization头获取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// 从查询参数获取
	if token := c.Query("token"); token != "" {
		return token
	}

	// 从cookie获取
	if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
		return cookie
	}

	return ""
}

// validateToken 验证token
func (am *AuthMiddleware) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return []byte(am.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// 检查token是否过期
		if time.Now().Unix() > claims.ExpiresAt {
			return nil, fmt.Errorf("token已过期")
		}

		return claims, nil
	}

	return nil, fmt.Errorf("无效的token")
}

// isWhitelisted 检查路径是否在白名单中
func (am *AuthMiddleware) isWhitelisted(path string) bool {
	for _, whitelistPath := range am.whitelistPaths {
		if path == whitelistPath || strings.HasPrefix(path, whitelistPath) {
			return true
		}
	}
	return false
}

// SupabaseAuthMiddleware Supabase认证中间件
type SupabaseAuthMiddleware struct {
	supabaseURL string
	supabaseKey string
	logger      *logx.Logger
	httpClient  *http.Client
}

// NewSupabaseAuthMiddleware 创建Supabase认证中间件
func NewSupabaseAuthMiddleware(supabaseURL, supabaseKey string, logger *logx.Logger) *SupabaseAuthMiddleware {
	return &SupabaseAuthMiddleware{
		supabaseURL: supabaseURL,
		supabaseKey: supabaseKey,
		logger:      logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SupabaseCheckToken 检查Supabase token
func (sam *SupabaseAuthMiddleware) SupabaseCheckToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用认证
		if sam.supabaseURL == "" || sam.supabaseKey == "" {
			c.Next()
			return
		}

		// 从请求头获取token
		token := c.GetHeader("Authorization")
		if token == "" {
			sam.logger.Warn(c.Request.Context(), "未提供Supabase认证token",
				logx.KV("path", c.Request.URL.Path),
				logx.KV("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "未授权",
				"message": "请提供有效的认证token",
			})
			c.Abort()
			return
		}

		// 验证token
		user, err := sam.validateSupabaseToken(c.Request.Context(), token)
		if err != nil {
			sam.logger.Warn(c.Request.Context(), "Supabase token验证失败",
				logx.KV("error", err),
				logx.KV("path", c.Request.URL.Path),
				logx.KV("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "认证失败",
				"message": "无效的token",
			})
			c.Abort()
			return
		}

		// 将用户信息添加到上下文
		c.Set("user_id", user.ID)
		c.Set("user_email", user.Email)
		c.Set("user_role", user.Role)

		sam.logger.Info(c.Request.Context(), "Supabase用户认证成功",
			logx.KV("user_id", user.ID),
			logx.KV("user_email", user.Email),
			logx.KV("user_role", user.Role))

		c.Next()
	}
}

// SupabaseUser Supabase用户信息
type SupabaseUser struct {
	ID       string                 `json:"id"`
	Email    string                 `json:"email"`
	Role     string                 `json:"role"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SupabaseTokenResponse Supabase token响应
type SupabaseTokenResponse struct {
	User SupabaseUser `json:"user"`
}

// validateSupabaseToken 验证Supabase token
func (sam *SupabaseAuthMiddleware) validateSupabaseToken(ctx context.Context, tokenString string) (*SupabaseUser, error) {
	// 这里应该调用Supabase的token验证API
	// 由于没有实际的Supabase客户端，这里返回模拟数据
	if tokenString == "invalid" {
		return nil, fmt.Errorf("无效的token")
	}

	// 模拟验证成功
	user := &SupabaseUser{
		ID:       "user_123",
		Email:    "user@example.com",
		Role:     "user",
		Metadata: make(map[string]interface{}),
	}

	return user, nil
}
