package database

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
)

// Client 抽象数据库客户端接口
type Client interface {
	// 基础操作
	Ping(ctx context.Context) error
	IsAvailable() bool

	// 检查点操作
	SaveCheckpoint(ctx context.Context, req SaveCheckpointRequest) (*SaveCheckpointResponse, error)
	GetCheckpoint(ctx context.Context, req GetCheckpointRequest) (*CheckpointRecord, error)
	SaveCheckpointWrites(ctx context.Context, req SaveCheckpointWritesRequest) error

	// 文件存储操作
	SaveFile(ctx context.Context, req SaveFileRequest) (*SaveFileResponse, error)
	GetFile(ctx context.Context, req GetFileRequest) (*FileRecord, error)
	DeleteFile(ctx context.Context, req DeleteFileRequest) error
	ListFiles(ctx context.Context, req ListFilesRequest) (*ListFilesResponse, error)

	// 用户存储操作
	SaveUser(ctx context.Context, req SaveUserRequest) error
	GetUser(ctx context.Context, req GetUserRequest) (*UserRecord, error)
	UpdateUserStats(ctx context.Context, req UpdateUserStatsRequest) error

	// 上下文存储操作
	SaveContext(ctx context.Context, req SaveContextRequest) error
	GetContext(ctx context.Context, req GetContextRequest) (*ContextRecord, error)

	// Notes存储操作
	SaveNote(ctx context.Context, req SaveNoteRequest) (*SaveNoteResponse, error)
	GetNote(ctx context.Context, req GetNoteRequest) (*NoteRecord, error)
	UpdateNote(ctx context.Context, req UpdateNoteRequest) error
	DeleteNote(ctx context.Context, req DeleteNoteRequest) error
	ListNotes(ctx context.Context, req ListNotesRequest) (*ListNotesResponse, error)

	// 流存储操作
	SaveStream(ctx context.Context, req SaveStreamRequest) error
	LoadStream(ctx context.Context, req LoadStreamRequest) (*StreamEvent, error)
	DeleteStream(ctx context.Context, req DeleteStreamRequest) error
	ListStreams(ctx context.Context, req ListStreamsRequest) (*ListStreamsResponse, error)
}

// SupabaseClient Supabase客户端实现
type SupabaseClient struct {
	url        string
	key        string
	secret     string
	httpClient *http.Client
}

// NewSupabaseClient 创建新的Supabase客户端
func NewSupabaseClient(cfg config.DatabaseConfig) (*SupabaseClient, error) {
	if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
		return nil, errors.New("missing SUPABASE_URL or SUPABASE_KEY")
	}

	return &SupabaseClient{
		url:    cfg.SupabaseURL,
		key:    cfg.SupabaseKey,
		secret: cfg.SupabaseSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Ping 测试数据库连接
func (c *SupabaseClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.url+"/rest/v1/", nil)
	if err != nil {
		return err
	}

	req.Header.Set("apikey", c.key)
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("database ping failed with status: %d", resp.StatusCode)
	}

	return nil
}

// IsAvailable 检查数据库是否可用
func (c *SupabaseClient) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Ping(ctx) == nil
}

// makeRequest 发送HTTP请求到Supabase
func (c *SupabaseClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	url := c.url + "/rest/v1/" + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", c.key)
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	return c.httpClient.Do(req)
}

// 检查点相关结构体
type SaveCheckpointRequest struct {
	ThreadID           string                 `json:"thread_id"`
	CheckpointNS       string                 `json:"checkpoint_ns"`
	CheckpointID       string                 `json:"checkpoint_id"`
	ParentCheckpointID string                 `json:"parent_checkpoint_id,omitempty"`
	UserID             string                 `json:"user_id"`
	SessionID          string                 `json:"session_id"`
	CheckpointType     string                 `json:"checkpoint_type"`
	CheckpointData     []byte                 `json:"checkpoint_data"`
	MetadataData       map[string]interface{} `json:"metadata_data"`
	ChannelVersions    map[string]interface{} `json:"channel_versions"`
	RedisKey           string                 `json:"redis_key,omitempty"`
	RedisTTLExpiresAt  *time.Time             `json:"redis_ttl_expires_at,omitempty"`
}

type SaveCheckpointResponse struct {
	ID int64 `json:"id"`
}

type GetCheckpointRequest struct {
	ThreadID     string `json:"thread_id"`
	CheckpointNS string `json:"checkpoint_ns"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
}

type CheckpointRecord struct {
	ID                 int64                  `json:"id"`
	ThreadID           string                 `json:"thread_id"`
	CheckpointNS       string                 `json:"checkpoint_ns"`
	CheckpointID       string                 `json:"checkpoint_id"`
	ParentCheckpointID string                 `json:"parent_checkpoint_id"`
	UserID             string                 `json:"user_id"`
	SessionID          string                 `json:"session_id"`
	CheckpointType     string                 `json:"checkpoint_type"`
	CheckpointData     []byte                 `json:"checkpoint_data"`
	MetadataData       map[string]interface{} `json:"metadata_data"`
	ChannelVersions    map[string]interface{} `json:"channel_versions"`
	IsActive           bool                   `json:"is_active"`
	RedisKey           string                 `json:"redis_key"`
	RedisTTLExpiresAt  *time.Time             `json:"redis_ttl_expires_at"`
	BackupStatus       string                 `json:"backup_status"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	LastAccessedAt     *time.Time             `json:"last_accessed_at"`
}

type SaveCheckpointWritesRequest struct {
	CheckpointRefID int64                   `json:"checkpoint_ref_id"`
	ThreadID        string                  `json:"thread_id"`
	CheckpointNS    string                  `json:"checkpoint_ns"`
	CheckpointID    string                  `json:"checkpoint_id"`
	Writes          []CheckpointWriteRecord `json:"writes"`
}

type CheckpointWriteRecord struct {
	TaskID    string `json:"task_id"`
	WriteIdx  int    `json:"write_idx"`
	Channel   string `json:"channel"`
	WriteType string `json:"write_type"`
	WriteData []byte `json:"write_data"`
}

// SaveCheckpoint 保存检查点
func (c *SupabaseClient) SaveCheckpoint(ctx context.Context, req SaveCheckpointRequest) (*SaveCheckpointResponse, error) {
	// 编码二进制数据为base64
	encodedData := base64.StdEncoding.EncodeToString(req.CheckpointData)

	payload := map[string]interface{}{
		"thread_id":            req.ThreadID,
		"checkpoint_ns":        req.CheckpointNS,
		"checkpoint_id":        req.CheckpointID,
		"parent_checkpoint_id": req.ParentCheckpointID,
		"user_id":              req.UserID,
		"session_id":           req.SessionID,
		"checkpoint_type":      req.CheckpointType,
		"checkpoint_data":      encodedData,
		"metadata_data":        req.MetadataData,
		"channel_versions":     req.ChannelVersions,
		"redis_key":            req.RedisKey,
		"redis_ttl_expires_at": req.RedisTTLExpiresAt,
		"backup_status":        "active",
		"is_active":            true,
	}

	resp, err := c.makeRequest(ctx, "POST", "graph_checkpoints", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("save checkpoint failed with status: %d", resp.StatusCode)
	}

	var result []SaveCheckpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("no data returned from save checkpoint")
	}

	return &result[0], nil
}

// GetCheckpoint 获取检查点
func (c *SupabaseClient) GetCheckpoint(ctx context.Context, req GetCheckpointRequest) (*CheckpointRecord, error) {
	endpoint := fmt.Sprintf("graph_checkpoints?thread_id=eq.%s&checkpoint_ns=eq.%s", req.ThreadID, req.CheckpointNS)
	if req.CheckpointID != "" {
		endpoint += fmt.Sprintf("&checkpoint_id=eq.%s", req.CheckpointID)
	}
	endpoint += "&order=created_at.desc&limit=1"

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get checkpoint failed with status: %d", resp.StatusCode)
	}

	var result []CheckpointRecord
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	// 解码二进制数据
	if decoded, err := base64.StdEncoding.DecodeString(string(result[0].CheckpointData)); err == nil {
		result[0].CheckpointData = decoded
	}

	return &result[0], nil
}

// SaveCheckpointWrites 保存检查点写入记录
func (c *SupabaseClient) SaveCheckpointWrites(ctx context.Context, req SaveCheckpointWritesRequest) error {
	var writeRecords []map[string]interface{}

	for _, write := range req.Writes {
		encodedData := base64.StdEncoding.EncodeToString(write.WriteData)
		writeRecords = append(writeRecords, map[string]interface{}{
			"checkpoint_ref_id": req.CheckpointRefID,
			"thread_id":         req.ThreadID,
			"checkpoint_ns":     req.CheckpointNS,
			"checkpoint_id":     req.CheckpointID,
			"task_id":           write.TaskID,
			"write_idx":         write.WriteIdx,
			"channel":           write.Channel,
			"write_type":        write.WriteType,
			"write_data":        encodedData,
		})
	}

	if len(writeRecords) == 0 {
		return nil
	}

	resp, err := c.makeRequest(ctx, "POST", "graph_checkpoint_writes", writeRecords)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("save checkpoint writes failed with status: %d", resp.StatusCode)
	}

	return nil
}

// 文件存储相关结构体
type SaveFileRequest struct {
	UserID      string `json:"user_id"`
	SessionID   string `json:"session_id"`
	FileName    string `json:"file_name"`
	FileType    string `json:"file_type"`
	FileSize    int64  `json:"file_size"`
	FileData    []byte `json:"file_data"`
	OSSKey      string `json:"oss_key,omitempty"`
	Description string `json:"description,omitempty"`
}

type SaveFileResponse struct {
	ID int64 `json:"id"`
}

type GetFileRequest struct {
	ID       int64  `json:"id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	FileName string `json:"file_name,omitempty"`
}

type FileRecord struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	SessionID   string    `json:"session_id"`
	FileName    string    `json:"file_name"`
	FileType    string    `json:"file_type"`
	FileSize    int64     `json:"file_size"`
	OSSKey      string    `json:"oss_key"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DeleteFileRequest struct {
	ID int64 `json:"id"`
}

type ListFilesRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type ListFilesResponse struct {
	Files []FileRecord `json:"files"`
	Total int64        `json:"total"`
}

// SaveFile 保存文件
func (c *SupabaseClient) SaveFile(ctx context.Context, req SaveFileRequest) (*SaveFileResponse, error) {
	encodedData := base64.StdEncoding.EncodeToString(req.FileData)

	payload := map[string]interface{}{
		"user_id":     req.UserID,
		"session_id":  req.SessionID,
		"file_name":   req.FileName,
		"file_type":   req.FileType,
		"file_size":   req.FileSize,
		"file_data":   encodedData,
		"oss_key":     req.OSSKey,
		"description": req.Description,
	}

	resp, err := c.makeRequest(ctx, "POST", "file_storage", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("save file failed with status: %d", resp.StatusCode)
	}

	var result []SaveFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("no data returned from save file")
	}

	return &result[0], nil
}

// GetFile 获取文件
func (c *SupabaseClient) GetFile(ctx context.Context, req GetFileRequest) (*FileRecord, error) {
	var endpoint string

	if req.ID != 0 {
		endpoint = fmt.Sprintf("file_storage?id=eq.%d", req.ID)
	} else if req.UserID != "" && req.FileName != "" {
		endpoint = fmt.Sprintf("file_storage?user_id=eq.%s&file_name=eq.%s", req.UserID, req.FileName)
	} else {
		return nil, errors.New("either id or user_id+file_name must be provided")
	}

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get file failed with status: %d", resp.StatusCode)
	}

	var result []FileRecord
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return &result[0], nil
}

// DeleteFile 删除文件
func (c *SupabaseClient) DeleteFile(ctx context.Context, req DeleteFileRequest) error {
	endpoint := fmt.Sprintf("file_storage?id=eq.%d", req.ID)

	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete file failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ListFiles 列出文件
func (c *SupabaseClient) ListFiles(ctx context.Context, req ListFilesRequest) (*ListFilesResponse, error) {
	endpoint := fmt.Sprintf("file_storage?user_id=eq.%s", req.UserID)

	if req.SessionID != "" {
		endpoint += fmt.Sprintf("&session_id=eq.%s", req.SessionID)
	}

	endpoint += "&order=created_at.desc"

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
		return nil, fmt.Errorf("list files failed with status: %d", resp.StatusCode)
	}

	var files []FileRecord
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	return &ListFilesResponse{
		Files: files,
		Total: int64(len(files)),
	}, nil
}

// 用户存储相关结构体
type SaveUserRequest struct {
	UserID   string                 `json:"user_id"`
	Email    string                 `json:"email,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type GetUserRequest struct {
	UserID string `json:"user_id"`
}

type UserRecord struct {
	UserID      string                 `json:"user_id"`
	Email       string                 `json:"email"`
	Metadata    map[string]interface{} `json:"metadata"`
	AccessCount int64                  `json:"access_count"`
	LastAccess  *time.Time             `json:"last_access"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type UpdateUserStatsRequest struct {
	UserID string `json:"user_id"`
}

// SaveUser 保存用户
func (c *SupabaseClient) SaveUser(ctx context.Context, req SaveUserRequest) error {
	payload := map[string]interface{}{
		"user_id":      req.UserID,
		"email":        req.Email,
		"metadata":     req.Metadata,
		"access_count": 0,
		"last_access":  nil,
	}

	resp, err := c.makeRequest(ctx, "POST", "user_statistics", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("save user failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetUser 获取用户
func (c *SupabaseClient) GetUser(ctx context.Context, req GetUserRequest) (*UserRecord, error) {
	endpoint := fmt.Sprintf("user_statistics?user_id=eq.%s", req.UserID)

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get user failed with status: %d", resp.StatusCode)
	}

	var result []UserRecord
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return &result[0], nil
}

// UpdateUserStats 更新用户统计
func (c *SupabaseClient) UpdateUserStats(ctx context.Context, req UpdateUserStatsRequest) error {
	endpoint := fmt.Sprintf("user_statistics?user_id=eq.%s", req.UserID)

	payload := map[string]interface{}{
		"access_count": "user_statistics.access_count + 1",
		"last_access":  "now()",
	}

	resp, err := c.makeRequest(ctx, "PATCH", endpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("update user stats failed with status: %d", resp.StatusCode)
	}

	return nil
}

// 上下文存储相关结构体
type SaveContextRequest struct {
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	Context   map[string]interface{} `json:"context"`
}

type GetContextRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

type ContextRecord struct {
	ID        int64                  `json:"id"`
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	Context   map[string]interface{} `json:"context"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SaveContext 保存上下文
func (c *SupabaseClient) SaveContext(ctx context.Context, req SaveContextRequest) error {
	payload := map[string]interface{}{
		"user_id":    req.UserID,
		"session_id": req.SessionID,
		"context":    req.Context,
	}

	resp, err := c.makeRequest(ctx, "POST", "contexts", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("save context failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetContext 获取上下文
func (c *SupabaseClient) GetContext(ctx context.Context, req GetContextRequest) (*ContextRecord, error) {
	endpoint := fmt.Sprintf("contexts?user_id=eq.%s&session_id=eq.%s&order=updated_at.desc&limit=1", req.UserID, req.SessionID)

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get context failed with status: %d", resp.StatusCode)
	}

	var result []ContextRecord
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return &result[0], nil
}

// Notes存储相关结构体
type SaveNoteRequest struct {
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	AgentName string                 `json:"agent_name"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	NoteType  string                 `json:"note_type,omitempty"`
}

type SaveNoteResponse struct {
	ID int64 `json:"id"`
}

type GetNoteRequest struct {
	ID int64 `json:"id"`
}

type NoteRecord struct {
	ID        int64                  `json:"id"`
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	AgentName string                 `json:"agent_name"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	NoteType  string                 `json:"note_type"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type UpdateNoteRequest struct {
	ID       int64                  `json:"id"`
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type DeleteNoteRequest struct {
	ID int64 `json:"id"`
}

type ListNotesRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type ListNotesResponse struct {
	Notes []NoteRecord `json:"notes"`
	Total int64        `json:"total"`
}

// SaveNote 保存笔记
func (c *SupabaseClient) SaveNote(ctx context.Context, req SaveNoteRequest) (*SaveNoteResponse, error) {
	payload := map[string]interface{}{
		"user_id":    req.UserID,
		"session_id": req.SessionID,
		"agent_name": req.AgentName,
		"content":    req.Content,
		"metadata":   req.Metadata,
		"note_type":  req.NoteType,
	}

	resp, err := c.makeRequest(ctx, "POST", "notes", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("save note failed with status: %d", resp.StatusCode)
	}

	var result []SaveNoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("no data returned from save note")
	}

	return &result[0], nil
}

// GetNote 获取笔记
func (c *SupabaseClient) GetNote(ctx context.Context, req GetNoteRequest) (*NoteRecord, error) {
	endpoint := fmt.Sprintf("notes?id=eq.%d", req.ID)

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get note failed with status: %d", resp.StatusCode)
	}

	var result []NoteRecord
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return &result[0], nil
}

// UpdateNote 更新笔记
func (c *SupabaseClient) UpdateNote(ctx context.Context, req UpdateNoteRequest) error {
	endpoint := fmt.Sprintf("notes?id=eq.%d", req.ID)

	payload := make(map[string]interface{})
	if req.Content != "" {
		payload["content"] = req.Content
	}
	if req.Metadata != nil {
		payload["metadata"] = req.Metadata
	}

	resp, err := c.makeRequest(ctx, "PATCH", endpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("update note failed with status: %d", resp.StatusCode)
	}

	return nil
}

// DeleteNote 删除笔记
func (c *SupabaseClient) DeleteNote(ctx context.Context, req DeleteNoteRequest) error {
	endpoint := fmt.Sprintf("notes?id=eq.%d", req.ID)

	resp, err := c.makeRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete note failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ListNotes 列出笔记
func (c *SupabaseClient) ListNotes(ctx context.Context, req ListNotesRequest) (*ListNotesResponse, error) {
	endpoint := fmt.Sprintf("notes?user_id=eq.%s", req.UserID)

	if req.SessionID != "" {
		endpoint += fmt.Sprintf("&session_id=eq.%s", req.SessionID)
	}

	if req.AgentName != "" {
		endpoint += fmt.Sprintf("&agent_name=eq.%s", req.AgentName)
	}

	endpoint += "&order=created_at.desc"

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
		return nil, fmt.Errorf("list notes failed with status: %d", resp.StatusCode)
	}

	var notes []NoteRecord
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		return nil, err
	}

	return &ListNotesResponse{
		Notes: notes,
		Total: int64(len(notes)),
	}, nil
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

// SaveStream 保存流事件
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

// LoadStream 加载流事件
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

// DeleteStream 删除流事件
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

// ListStreams 列出流事件
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

// 全局客户端管理
var (
	globalClient Client
)

// GetSupabaseClient 获取Supabase客户端
func GetSupabaseClient() (Client, error) {
	if globalClient != nil {
		return globalClient, nil
	}

	// 从环境变量创建配置
	cfg := config.DatabaseConfig{
		SupabaseURL:    os.Getenv("SUPABASE_URL"),
		SupabaseKey:    os.Getenv("SUPABASE_KEY"),
		SupabaseSecret: os.Getenv("SUPABASE_SECRET"),
	}

	client, err := NewSupabaseClient(cfg)
	if err != nil {
		return nil, err
	}

	globalClient = client
	return globalClient, nil
}

// GetSupabaseClientWithConfig 使用配置创建Supabase客户端
func GetSupabaseClientWithConfig(cfg config.DatabaseConfig) (Client, error) {
	return NewSupabaseClient(cfg)
}

// ResetClient 重置全局客户端
func ResetClient() {
	globalClient = nil
}

// TestConnection 测试数据库连接
func TestConnection() bool {
	client, err := GetSupabaseClient()
	if err != nil {
		return false
	}

	return client.IsAvailable()
}
