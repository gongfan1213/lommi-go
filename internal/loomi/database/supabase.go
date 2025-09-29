package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	"github.com/blueplan/loomi-go/internal/loomi/log"
)

// SupabaseContextStorage 上下文存储实现
type SupabaseContextStorage struct {
	client Client
	logger log.Logger
}

// NewSupabaseContextStorage 创建Supabase上下文存储
func NewSupabaseContextStorage(client Client, logger log.Logger) *SupabaseContextStorage {
	return &SupabaseContextStorage{
		client: client,
		logger: logger,
	}
}

// IsAvailable 检查是否可用
func (s *SupabaseContextStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

// Ping 测试连接
func (s *SupabaseContextStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

// HealthCheck 健康检查
func (s *SupabaseContextStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_context_storage",
		"timestamp": time.Now(),
	}
}

// SaveContext 保存上下文
func (s *SupabaseContextStorage) SaveContext(ctx context.Context, req SaveContextRequest) error {
	s.logger.Info(ctx, "保存上下文",
		"user_id", req.UserID,
		"session_id", req.SessionID)

	return s.client.SaveContext(ctx, req)
}

// GetContext 获取上下文
func (s *SupabaseContextStorage) GetContext(ctx context.Context, req GetContextRequest) (*ContextRecord, error) {
	s.logger.Info(ctx, "获取上下文",
		"user_id", req.UserID,
		"session_id", req.SessionID)

	return s.client.GetContext(ctx, req)
}

// SupabaseNotesStorage Notes存储实现
type SupabaseNotesStorage struct {
	client Client
	logger log.Logger
}

// NewSupabaseNotesStorage 创建Supabase Notes存储
func NewSupabaseNotesStorage(client Client, logger log.Logger) *SupabaseNotesStorage {
	return &SupabaseNotesStorage{
		client: client,
		logger: logger,
	}
}

// IsAvailable 检查是否可用
func (s *SupabaseNotesStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

// Ping 测试连接
func (s *SupabaseNotesStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

// HealthCheck 健康检查
func (s *SupabaseNotesStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_notes_storage",
		"timestamp": time.Now(),
	}
}

// SaveNote 保存笔记
func (s *SupabaseNotesStorage) SaveNote(ctx context.Context, req SaveNoteRequest) (*SaveNoteResponse, error) {
	s.logger.Info(ctx, "保存笔记",
		"user_id", req.UserID,
		"session_id", req.SessionID,
		"agent_name", req.AgentName)

	return s.client.SaveNote(ctx, req)
}

// GetNote 获取笔记
func (s *SupabaseNotesStorage) GetNote(ctx context.Context, req GetNoteRequest) (*NoteRecord, error) {
	s.logger.Info(ctx, "获取笔记", "id", req.ID)
	return s.client.GetNote(ctx, req)
}

// UpdateNote 更新笔记
func (s *SupabaseNotesStorage) UpdateNote(ctx context.Context, req UpdateNoteRequest) error {
	s.logger.Info(ctx, "更新笔记", "id", req.ID)
	return s.client.UpdateNote(ctx, req)
}

// DeleteNote 删除笔记
func (s *SupabaseNotesStorage) DeleteNote(ctx context.Context, req DeleteNoteRequest) error {
	s.logger.Info(ctx, "删除笔记", "id", req.ID)
	return s.client.DeleteNote(ctx, req)
}

// ListNotes 列出笔记
func (s *SupabaseNotesStorage) ListNotes(ctx context.Context, req ListNotesRequest) (*ListNotesResponse, error) {
	s.logger.Info(ctx, "列出笔记",
		"user_id", req.UserID,
		"session_id", req.SessionID,
		"agent_name", req.AgentName)

	return s.client.ListNotes(ctx, req)
}

// SupabaseUserStorage 用户存储实现
type SupabaseUserStorage struct {
	client Client
	logger log.Logger
}

// NewSupabaseUserStorage 创建Supabase用户存储
func NewSupabaseUserStorage(client Client, logger log.Logger) *SupabaseUserStorage {
	return &SupabaseUserStorage{
		client: client,
		logger: logger,
	}
}

// IsAvailable 检查是否可用
func (s *SupabaseUserStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

// Ping 测试连接
func (s *SupabaseUserStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

// HealthCheck 健康检查
func (s *SupabaseUserStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_user_storage",
		"timestamp": time.Now(),
	}
}

// SaveUser 保存用户
func (s *SupabaseUserStorage) SaveUser(ctx context.Context, req SaveUserRequest) error {
	s.logger.Info(ctx, "保存用户", "user_id", req.UserID)
	return s.client.SaveUser(ctx, req)
}

// GetUser 获取用户
func (s *SupabaseUserStorage) GetUser(ctx context.Context, req GetUserRequest) (*UserRecord, error) {
	s.logger.Info(ctx, "获取用户", "user_id", req.UserID)
	return s.client.GetUser(ctx, req)
}

// UpdateUserStats 更新用户统计
func (s *SupabaseUserStorage) UpdateUserStats(ctx context.Context, req UpdateUserStatsRequest) error {
	s.logger.Info(ctx, "更新用户统计", "user_id", req.UserID)
	return s.client.UpdateUserStats(ctx, req)
}

// SupabaseStreamStorage 流存储实现
type SupabaseStreamStorage struct {
	client Client
	logger log.Logger
}

// NewSupabaseStreamStorage 创建Supabase流存储
func NewSupabaseStreamStorage(client Client, logger log.Logger) *SupabaseStreamStorage {
	return &SupabaseStreamStorage{
		client: client,
		logger: logger,
	}
}

// IsAvailable 检查是否可用
func (s *SupabaseStreamStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

// Ping 测试连接
func (s *SupabaseStreamStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

// HealthCheck 健康检查
func (s *SupabaseStreamStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_stream_storage",
		"timestamp": time.Now(),
	}
}

// SaveStream 保存流
func (s *SupabaseStreamStorage) SaveStream(ctx context.Context, req SaveStreamRequest) error {
	s.logger.Info(ctx, "保存流",
		"user_id", req.UserID,
		"session_id", req.SessionID,
		"event_type", req.EventType)

	return s.client.SaveStream(ctx, req)
}

// LoadStream 加载流
func (s *SupabaseStreamStorage) LoadStream(ctx context.Context, req LoadStreamRequest) (*StreamEvent, error) {
	s.logger.Info(ctx, "加载流",
		"user_id", req.UserID,
		"session_id", req.SessionID)

	return s.client.LoadStream(ctx, req)
}

// DeleteStream 删除流
func (s *SupabaseStreamStorage) DeleteStream(ctx context.Context, req DeleteStreamRequest) error {
	s.logger.Info(ctx, "删除流", "id", req.ID)
	return s.client.DeleteStream(ctx, req)
}

// ListStreams 列出流
func (s *SupabaseStreamStorage) ListStreams(ctx context.Context, req ListStreamsRequest) (*ListStreamsResponse, error) {
	s.logger.Info(ctx, "列出流",
		"user_id", req.UserID,
		"session_id", req.SessionID)

	return s.client.ListStreams(ctx, req)
}

// 流存储相关结构体
type SaveStreamRequest struct {
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
}

type LoadStreamRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	EventType string `json:"event_type,omitempty"`
}

type DeleteStreamRequest struct {
	ID int64 `json:"id"`
}

type ListStreamsRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type ListStreamsResponse struct {
	Streams []StreamEvent `json:"streams"`
	Total   int64         `json:"total"`
}

// 扩展SupabaseClient实现流存储方法
func (c *SupabaseClient) SaveStream(ctx context.Context, req SaveStreamRequest) error {
	payload := map[string]interface{}{
		"user_id":    req.UserID,
		"session_id": req.SessionID,
		"event_type": req.EventType,
		"data":       req.Data,
	}

	resp, err := c.makeRequest(ctx, "POST", "stream_events", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("save stream failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *SupabaseClient) LoadStream(ctx context.Context, req LoadStreamRequest) (*StreamEvent, error) {
	endpoint := fmt.Sprintf("stream_events?user_id=eq.%s&session_id=eq.%s", req.UserID, req.SessionID)

	if req.EventType != "" {
		endpoint += fmt.Sprintf("&event_type=eq.%s", req.EventType)
	}

	endpoint += "&order=timestamp.desc&limit=1"

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("load stream failed with status: %d", resp.StatusCode)
	}

	var result []StreamEvent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return &result[0], nil
}

func (c *SupabaseClient) DeleteStream(ctx context.Context, req DeleteStreamRequest) error {
	endpoint := fmt.Sprintf("stream_events?id=eq.%d", req.ID)

	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete stream failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *SupabaseClient) ListStreams(ctx context.Context, req ListStreamsRequest) (*ListStreamsResponse, error) {
	endpoint := fmt.Sprintf("stream_events?user_id=eq.%s", req.UserID)

	if req.SessionID != "" {
		endpoint += fmt.Sprintf("&session_id=eq.%s", req.SessionID)
	}

	if req.EventType != "" {
		endpoint += fmt.Sprintf("&event_type=eq.%s", req.EventType)
	}

	endpoint += "&order=timestamp.desc"

	if req.Limit > 0 {
		endpoint += fmt.Sprintf("&limit=%d", req.Limit)
	}

	if req.Offset > 0 {
		endpoint += fmt.Sprintf("&offset=%d", req.Offset)
	}

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list streams failed with status: %d", resp.StatusCode)
	}

	var streams []StreamEvent
	if err := json.NewDecoder(resp.Body).Decode(&streams); err != nil {
		return nil, err
	}

	return &ListStreamsResponse{
		Streams: streams,
		Total:   int64(len(streams)),
	}, nil
}

// Token 校验：调用 /auth/v1/user 获取用户信息
func ValidateToken(ctx context.Context, jwt string) (string, error) {
	url := os.Getenv("SUPABASE_URL")
	svc := os.Getenv("SUPABASE_SERVICE_KEY")
	if url == "" || svc == "" || jwt == "" {
		return "", errors.New("missing supabase settings or jwt")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/auth/v1/user", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("apikey", svc)
	req.Header.Set("Authorization", "Bearer "+jwt)

	httpc := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("auth status=%d", resp.StatusCode)
	}

	var out struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	if out.User.ID == "" {
		return "", errors.New("no user id")
	}

	return out.User.ID, nil
}

// 工厂函数
func NewSupabaseContextStorageFromConfig(cfg config.DatabaseConfig, logger log.Logger) (*SupabaseContextStorage, error) {
	client, err := NewSupabaseClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewSupabaseContextStorage(client, logger), nil
}

func NewSupabaseNotesStorageFromConfig(cfg config.DatabaseConfig, logger log.Logger) (*SupabaseNotesStorage, error) {
	client, err := NewSupabaseClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewSupabaseNotesStorage(client, logger), nil
}

func NewSupabaseUserStorageFromConfig(cfg config.DatabaseConfig, logger log.Logger) (*SupabaseUserStorage, error) {
	client, err := NewSupabaseClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewSupabaseUserStorage(client, logger), nil
}

func NewSupabaseStreamStorageFromConfig(cfg config.DatabaseConfig, logger log.Logger) (*SupabaseStreamStorage, error) {
	client, err := NewSupabaseClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewSupabaseStreamStorage(client, logger), nil
}
