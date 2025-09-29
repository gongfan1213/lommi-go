package database

import (
	"context"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// Storage 存储接口
type Storage interface {
	IsAvailable() bool
	Ping(ctx context.Context) error
	HealthCheck(ctx context.Context) map[string]interface{}
}

// CheckpointStorage 检查点存储接口
type CheckpointStorage interface {
	Storage
	SaveCheckpoint(ctx context.Context, req SaveCheckpointRequest) (*SaveCheckpointResponse, error)
	GetCheckpoint(ctx context.Context, req GetCheckpointRequest) (*CheckpointRecord, error)
	SaveCheckpointWrites(ctx context.Context, req SaveCheckpointWritesRequest) error
}

// FileStorage 文件存储接口
type FileStorage interface {
	Storage
	SaveFile(ctx context.Context, req SaveFileRequest) (*SaveFileResponse, error)
	GetFile(ctx context.Context, req GetFileRequest) (*FileRecord, error)
	DeleteFile(ctx context.Context, req DeleteFileRequest) error
	ListFiles(ctx context.Context, req ListFilesRequest) (*ListFilesResponse, error)
}

// UserStorage 用户存储接口
type UserStorage interface {
	Storage
	SaveUser(ctx context.Context, req SaveUserRequest) error
	GetUser(ctx context.Context, req GetUserRequest) (*UserRecord, error)
	UpdateUserStats(ctx context.Context, req UpdateUserStatsRequest) error
}

// ContextStorage 上下文存储接口
type ContextStorage interface {
	Storage
	SaveContext(ctx context.Context, req SaveContextRequest) error
	GetContext(ctx context.Context, req GetContextRequest) (*ContextRecord, error)
}

// NotesStorage Notes存储接口
type NotesStorage interface {
	Storage
	SaveNote(ctx context.Context, req SaveNoteRequest) (*SaveNoteResponse, error)
	GetNote(ctx context.Context, req GetNoteRequest) (*NoteRecord, error)
	UpdateNote(ctx context.Context, req UpdateNoteRequest) error
	DeleteNote(ctx context.Context, req DeleteNoteRequest) error
	ListNotes(ctx context.Context, req ListNotesRequest) (*ListNotesResponse, error)
}

// StreamEvent 流事件结构
type StreamEvent struct {
	ID        int64                  `json:"id"`
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// SupabaseCheckpointStorage Supabase检查点存储实现
type SupabaseCheckpointStorage struct {
	client Client
	logger *logx.Logger
}

// NewSupabaseCheckpointStorage 创建Supabase检查点存储
func NewSupabaseCheckpointStorage(client Client, logger *logx.Logger) *SupabaseCheckpointStorage {
	return &SupabaseCheckpointStorage{
		client: client,
		logger: logger,
	}
}

func (s *SupabaseCheckpointStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

func (s *SupabaseCheckpointStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

func (s *SupabaseCheckpointStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_checkpoint_storage",
	}
}

func (s *SupabaseCheckpointStorage) SaveCheckpoint(ctx context.Context, req SaveCheckpointRequest) (*SaveCheckpointResponse, error) {
	return s.client.SaveCheckpoint(ctx, req)
}

func (s *SupabaseCheckpointStorage) GetCheckpoint(ctx context.Context, req GetCheckpointRequest) (*CheckpointRecord, error) {
	return s.client.GetCheckpoint(ctx, req)
}

func (s *SupabaseCheckpointStorage) SaveCheckpointWrites(ctx context.Context, req SaveCheckpointWritesRequest) error {
	return s.client.SaveCheckpointWrites(ctx, req)
}

// SupabaseFileStorage Supabase文件存储实现
type SupabaseFileStorage struct {
	client Client
	logger *logx.Logger
}

// NewSupabaseFileStorage 创建Supabase文件存储
func NewSupabaseFileStorage(client Client, logger *logx.Logger) *SupabaseFileStorage {
	return &SupabaseFileStorage{
		client: client,
		logger: logger,
	}
}

func (s *SupabaseFileStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

func (s *SupabaseFileStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

func (s *SupabaseFileStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_file_storage",
	}
}

func (s *SupabaseFileStorage) SaveFile(ctx context.Context, req SaveFileRequest) (*SaveFileResponse, error) {
	return s.client.SaveFile(ctx, req)
}

func (s *SupabaseFileStorage) GetFile(ctx context.Context, req GetFileRequest) (*FileRecord, error) {
	return s.client.GetFile(ctx, req)
}

func (s *SupabaseFileStorage) DeleteFile(ctx context.Context, req DeleteFileRequest) error {
	return s.client.DeleteFile(ctx, req)
}

func (s *SupabaseFileStorage) ListFiles(ctx context.Context, req ListFilesRequest) (*ListFilesResponse, error) {
	return s.client.ListFiles(ctx, req)
}

// SupabaseUserStorage Supabase用户存储实现
type SupabaseUserStorage struct {
	client Client
	logger *logx.Logger
}

// NewSupabaseUserStorage 创建Supabase用户存储
func NewSupabaseUserStorage(client Client, logger *logx.Logger) *SupabaseUserStorage {
	return &SupabaseUserStorage{
		client: client,
		logger: logger,
	}
}

func (s *SupabaseUserStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

func (s *SupabaseUserStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

func (s *SupabaseUserStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_user_storage",
	}
}

func (s *SupabaseUserStorage) SaveUser(ctx context.Context, req SaveUserRequest) error {
	return s.client.SaveUser(ctx, req)
}

func (s *SupabaseUserStorage) GetUser(ctx context.Context, req GetUserRequest) (*UserRecord, error) {
	return s.client.GetUser(ctx, req)
}

func (s *SupabaseUserStorage) UpdateUserStats(ctx context.Context, req UpdateUserStatsRequest) error {
	return s.client.UpdateUserStats(ctx, req)
}

// SupabaseContextStorage Supabase上下文存储实现
type SupabaseContextStorage struct {
	client Client
	logger *logx.Logger
}

// NewSupabaseContextStorage 创建Supabase上下文存储
func NewSupabaseContextStorage(client Client, logger *logx.Logger) *SupabaseContextStorage {
	return &SupabaseContextStorage{
		client: client,
		logger: logger,
	}
}

func (s *SupabaseContextStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

func (s *SupabaseContextStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

func (s *SupabaseContextStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_context_storage",
	}
}

func (s *SupabaseContextStorage) SaveContext(ctx context.Context, req SaveContextRequest) error {
	return s.client.SaveContext(ctx, req)
}

func (s *SupabaseContextStorage) GetContext(ctx context.Context, req GetContextRequest) (*ContextRecord, error) {
	return s.client.GetContext(ctx, req)
}

// SupabaseNotesStorage Supabase Notes存储实现
type SupabaseNotesStorage struct {
	client Client
	logger *logx.Logger
}

// NewSupabaseNotesStorage 创建Supabase Notes存储
func NewSupabaseNotesStorage(client Client, logger *logx.Logger) *SupabaseNotesStorage {
	return &SupabaseNotesStorage{
		client: client,
		logger: logger,
	}
}

func (s *SupabaseNotesStorage) IsAvailable() bool {
	return s.client.IsAvailable()
}

func (s *SupabaseNotesStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

func (s *SupabaseNotesStorage) HealthCheck(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"available": s.IsAvailable(),
		"type":      "supabase_notes_storage",
	}
}

func (s *SupabaseNotesStorage) SaveNote(ctx context.Context, req SaveNoteRequest) (*SaveNoteResponse, error) {
	return s.client.SaveNote(ctx, req)
}

func (s *SupabaseNotesStorage) GetNote(ctx context.Context, req GetNoteRequest) (*NoteRecord, error) {
	return s.client.GetNote(ctx, req)
}

func (s *SupabaseNotesStorage) UpdateNote(ctx context.Context, req UpdateNoteRequest) error {
	return s.client.UpdateNote(ctx, req)
}

func (s *SupabaseNotesStorage) DeleteNote(ctx context.Context, req DeleteNoteRequest) error {
	return s.client.DeleteNote(ctx, req)
}

func (s *SupabaseNotesStorage) ListNotes(ctx context.Context, req ListNotesRequest) (*ListNotesResponse, error) {
	return s.client.ListNotes(ctx, req)
}
