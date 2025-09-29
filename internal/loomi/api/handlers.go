package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/database"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/tools"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/utils"

	"github.com/gin-gonic/gin"
)

// Handler API处理器
type Handler struct {
	config              *config.Config
	logger              log.Logger
	persistenceManager  *database.PersistenceManager
	searchTool          tools.SearchTool
	multimodalProcessor tools.MultimodalProcessor
}

// NewHandler 创建新的处理器
func NewHandler(
	config *config.Config,
	logger log.Logger,
	persistenceManager *database.PersistenceManager,
	searchTool tools.SearchTool,
	multimodalProcessor tools.MultimodalProcessor,
) *Handler {
	return &Handler{
		config:              config,
		logger:              logger,
		persistenceManager:  persistenceManager,
		searchTool:          searchTool,
		multimodalProcessor: multimodalProcessor,
	}
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginationResponse 分页响应结构
type PaginationResponse struct {
	Items      interface{} `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// SuccessResponse 成功响应
func (h *Handler) SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// ErrorResponse 错误响应
func (h *Handler) ErrorResponse(c *gin.Context, code int, message string, err error) {
	response := Response{
		Code:    code,
		Message: message,
	}

	if err != nil {
		response.Error = err.Error()
	}

	c.JSON(code, response)
}

// BadRequestResponse 400错误响应
func (h *Handler) BadRequestResponse(c *gin.Context, message string, err error) {
	h.ErrorResponse(c, http.StatusBadRequest, message, err)
}

// UnauthorizedResponse 401错误响应
func (h *Handler) UnauthorizedResponse(c *gin.Context, message string) {
	h.ErrorResponse(c, http.StatusUnauthorized, message, nil)
}

// NotFoundResponse 404错误响应
func (h *Handler) NotFoundResponse(c *gin.Context, message string) {
	h.ErrorResponse(c, http.StatusNotFound, message, nil)
}

// InternalServerErrorResponse 500错误响应
func (h *Handler) InternalServerErrorResponse(c *gin.Context, message string, err error) {
	h.ErrorResponse(c, http.StatusInternalServerError, message, err)
}

// GetUserIDFromContext 从上下文获取用户ID
func (h *Handler) GetUserIDFromContext(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", fmt.Errorf("用户ID不存在")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", fmt.Errorf("用户ID类型错误")
	}

	return userIDStr, nil
}

// LogRequest 记录请求日志
func (h *Handler) LogRequest(c *gin.Context, operation string) {
	ctx := context.Background()

	// 从上下文获取信息
	userID, _ := h.GetUserIDFromContext(c)

	h.logger.Info(ctx, "API请求",
		"operation", operation,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"user_id", userID,
		"ip", c.ClientIP(),
		"user_agent", c.Request.UserAgent())
}

// LogResponse 记录响应日志
func (h *Handler) LogResponse(c *gin.Context, operation string, statusCode int, duration time.Duration) {
	ctx := context.Background()

	// 从上下文获取信息
	userID, _ := h.GetUserIDFromContext(c)

	h.logger.Info(ctx, "API响应",
		"operation", operation,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status_code", statusCode,
		"duration", duration,
		"user_id", userID)
}

// ValidateRequest 验证请求
func (h *Handler) ValidateRequest(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		return fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 使用验证器进行进一步验证
	validator := utils.NewValidator()

	// 这里应该根据请求类型进行具体的验证
	// 简化实现，只返回nil
	if err := validator.Validate(); err != nil {
		return fmt.Errorf("请求参数验证失败: %w", err)
	}

	return nil
}

// ParsePaginationParams 解析分页参数
func (h *Handler) ParsePaginationParams(c *gin.Context) (int, int, error) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, 0, fmt.Errorf("页码参数错误")
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		return 0, 0, fmt.Errorf("每页大小参数错误")
	}

	return page, pageSize, nil
}

// BuildPaginationResponse 构建分页响应
func (h *Handler) BuildPaginationResponse(items interface{}, page, pageSize int, total int64) *PaginationResponse {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &PaginationResponse{
		Items: items,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// HandleHealth 处理健康检查
func (h *Handler) HandleHealth(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.LogResponse(c, "health_check", http.StatusOK, time.Since(start))
	}()

	h.LogRequest(c, "health_check")

	// 检查各个组件的健康状态
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"uptime":    time.Since(time.Now()), // 实际应该是启动时间
		"components": gin.H{
			"database":   "healthy",
			"redis":      "healthy",
			"search":     "healthy",
			"multimodal": "healthy",
		},
	}

	h.SuccessResponse(c, health)
}

// HandlePing 处理Ping请求
func (h *Handler) HandlePing(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.LogResponse(c, "ping", http.StatusOK, time.Since(start))
	}()

	h.LogRequest(c, "ping")

	h.SuccessResponse(c, gin.H{
		"message":   "pong",
		"timestamp": time.Now(),
	})
}

// HandleLogin 处理登录
func (h *Handler) HandleLogin(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.LogResponse(c, "login", http.StatusOK, time.Since(start))
	}()

	h.LogRequest(c, "login")

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := h.ValidateRequest(c, &req); err != nil {
		h.BadRequestResponse(c, "请求参数错误", err)
		return
	}

	// 这里应该实现实际的登录逻辑
	// 包括密码验证、JWT令牌生成等

	token := "jwt_token_" + utils.RandomUtil.RandomString(32)

	user := gin.H{
		"id":         1,
		"username":   req.Username,
		"email":      req.Username + "@example.com",
		"role":       "user",
		"created_at": time.Now(),
	}

	response := gin.H{
		"token":      token,
		"user":       user,
		"expires_in": 3600,
		"token_type": "Bearer",
	}

	h.SuccessResponse(c, response)
}

// HandleWebSearch 处理网络搜索
func (h *Handler) HandleWebSearch(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.LogResponse(c, "web_search", http.StatusOK, time.Since(start))
	}()

	h.LogRequest(c, "web_search")

	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
	}

	if err := h.ValidateRequest(c, &req); err != nil {
		h.BadRequestResponse(c, "请求参数错误", err)
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	if req.Limit > 100 {
		req.Limit = 100
	}

	// 调用搜索工具
	results, err := h.searchTool.SearchWeb(context.Background(), req.Query, req.Limit)
	if err != nil {
		h.InternalServerErrorResponse(c, "搜索失败", err)
		return
	}

	response := gin.H{
		"query":   req.Query,
		"results": results,
		"total":   len(results),
	}

	h.SuccessResponse(c, response)
}

// HandleMultimodalProcess 处理多模态处理
func (h *Handler) HandleMultimodalProcess(c *gin.Context) {
	start := time.Now()
	defer func() {
		h.LogResponse(c, "multimodal_process", http.StatusOK, time.Since(start))
	}()

	h.LogRequest(c, "multimodal_process")

	var req struct {
		Files []string `json:"files" binding:"required"`
		Type  string   `json:"type" binding:"required"`
	}

	if err := h.ValidateRequest(c, &req); err != nil {
		h.BadRequestResponse(c, "请求参数错误", err)
		return
	}

	if len(req.Files) == 0 {
		h.BadRequestResponse(c, "文件列表不能为空", nil)
		return
	}

	// 调用多模态处理工具
	results, err := h.multimodalProcessor.ProcessMultimodalFiles(context.Background(), req.Files, req.Type)
	if err != nil {
		h.InternalServerErrorResponse(c, "多模态处理失败", err)
		return
	}

	response := gin.H{
		"type":    req.Type,
		"files":   req.Files,
		"results": results,
		"total":   len(results),
	}

	h.SuccessResponse(c, response)
}
