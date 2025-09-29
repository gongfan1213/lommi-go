package database

import (
	"context"
	"sync"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// InMemClient 内存数据库客户端实现
type InMemClient struct {
	checkpoints map[string]*CheckpointRecord
	files       map[string]*FileRecord
	users       map[string]*UserRecord
	contexts    map[string]*ContextRecord
	notes       map[string]*NoteRecord
	mu          sync.RWMutex
	logger      *logx.Logger
}

// NewInMemClient 创建内存数据库客户端
func NewInMemClient(logger *logx.Logger) *InMemClient {
	return &InMemClient{
		checkpoints: make(map[string]*CheckpointRecord),
		files:       make(map[string]*FileRecord),
		users:       make(map[string]*UserRecord),
		contexts:    make(map[string]*ContextRecord),
		notes:       make(map[string]*NoteRecord),
		logger:      logger,
	}
}

// Ping 测试连接
func (c *InMemClient) Ping(ctx context.Context) error {
	return nil
}

// IsAvailable 检查是否可用
func (c *InMemClient) IsAvailable() bool {
	return true
}

// SaveCheckpoint 保存检查点
func (c *InMemClient) SaveCheckpoint(ctx context.Context, req SaveCheckpointRequest) (*SaveCheckpointResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := req.ThreadID + ":" + req.CheckpointNS + ":" + req.CheckpointID
	record := &CheckpointRecord{
		ID:                 int64(len(c.checkpoints) + 1),
		ThreadID:           req.ThreadID,
		CheckpointNS:       req.CheckpointNS,
		CheckpointID:       req.CheckpointID,
		ParentCheckpointID: req.ParentCheckpointID,
		UserID:             req.UserID,
		SessionID:          req.SessionID,
		CheckpointType:     req.CheckpointType,
		CheckpointData:     req.CheckpointData,
		MetadataData:       req.MetadataData,
		ChannelVersions:    req.ChannelVersions,
		IsActive:           true,
		RedisKey:           req.RedisKey,
		RedisTTLExpiresAt:  req.RedisTTLExpiresAt,
		BackupStatus:       "active",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		LastAccessedAt:     &time.Time{},
	}

	c.checkpoints[key] = record

	return &SaveCheckpointResponse{
		ID: record.ID,
	}, nil
}

// GetCheckpoint 获取检查点
func (c *InMemClient) GetCheckpoint(ctx context.Context, req GetCheckpointRequest) (*CheckpointRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := req.ThreadID + ":" + req.CheckpointNS
	if req.CheckpointID != "" {
		key += ":" + req.CheckpointID
	}

	record, exists := c.checkpoints[key]
	if !exists {
		return nil, nil
	}

	return record, nil
}

// SaveCheckpointWrites 保存检查点写入记录
func (c *InMemClient) SaveCheckpointWrites(ctx context.Context, req SaveCheckpointWritesRequest) error {
	// 内存实现中暂时不保存写入记录
	return nil
}

// SaveFile 保存文件
func (c *InMemClient) SaveFile(ctx context.Context, req SaveFileRequest) (*SaveFileResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := req.UserID + ":" + req.FileName
	record := &FileRecord{
		ID:          int64(len(c.files) + 1),
		UserID:      req.UserID,
		SessionID:   req.SessionID,
		FileName:    req.FileName,
		FileType:    req.FileType,
		FileSize:    req.FileSize,
		OSSKey:      req.OSSKey,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	c.files[key] = record

	return &SaveFileResponse{
		ID: record.ID,
	}, nil
}

// GetFile 获取文件
func (c *InMemClient) GetFile(ctx context.Context, req GetFileRequest) (*FileRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if req.ID != 0 {
		// 根据ID查找
		for _, record := range c.files {
			if record.ID == req.ID {
				return record, nil
			}
		}
		return nil, nil
	}

	if req.UserID != "" && req.FileName != "" {
		key := req.UserID + ":" + req.FileName
		record, exists := c.files[key]
		if !exists {
			return nil, nil
		}
		return record, nil
	}

	return nil, nil
}

// DeleteFile 删除文件
func (c *InMemClient) DeleteFile(ctx context.Context, req DeleteFileRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, record := range c.files {
		if record.ID == req.ID {
			delete(c.files, key)
			break
		}
	}

	return nil
}

// ListFiles 列出文件
func (c *InMemClient) ListFiles(ctx context.Context, req ListFilesRequest) (*ListFilesResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var files []FileRecord
	for _, record := range c.files {
		if record.UserID == req.UserID {
			if req.SessionID == "" || record.SessionID == req.SessionID {
				files = append(files, *record)
			}
		}
	}

	// 应用分页
	start := req.Offset
	end := start + req.Limit
	if req.Limit == 0 {
		end = len(files)
	}

	if start >= len(files) {
		files = []FileRecord{}
	} else if end > len(files) {
		files = files[start:]
	} else {
		files = files[start:end]
	}

	return &ListFilesResponse{
		Files: files,
		Total: int64(len(files)),
	}, nil
}

// SaveUser 保存用户
func (c *InMemClient) SaveUser(ctx context.Context, req SaveUserRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	record := &UserRecord{
		UserID:      req.UserID,
		Email:       req.Email,
		Metadata:    req.Metadata,
		AccessCount: 0,
		LastAccess:  nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	c.users[req.UserID] = record
	return nil
}

// GetUser 获取用户
func (c *InMemClient) GetUser(ctx context.Context, req GetUserRequest) (*UserRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	record, exists := c.users[req.UserID]
	if !exists {
		return nil, nil
	}

	return record, nil
}

// UpdateUserStats 更新用户统计
func (c *InMemClient) UpdateUserStats(ctx context.Context, req UpdateUserStatsRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	record, exists := c.users[req.UserID]
	if !exists {
		return nil
	}

	record.AccessCount++
	now := time.Now()
	record.LastAccess = &now
	record.UpdatedAt = now

	return nil
}

// SaveContext 保存上下文
func (c *InMemClient) SaveContext(ctx context.Context, req SaveContextRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := req.UserID + ":" + req.SessionID
	record := &ContextRecord{
		ID:        int64(len(c.contexts) + 1),
		UserID:    req.UserID,
		SessionID: req.SessionID,
		Context:   req.Context,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	c.contexts[key] = record
	return nil
}

// GetContext 获取上下文
func (c *InMemClient) GetContext(ctx context.Context, req GetContextRequest) (*ContextRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := req.UserID + ":" + req.SessionID
	record, exists := c.contexts[key]
	if !exists {
		return nil, nil
	}

	return record, nil
}

// SaveNote 保存笔记
func (c *InMemClient) SaveNote(ctx context.Context, req SaveNoteRequest) (*SaveNoteResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := req.UserID + ":" + req.SessionID + ":" + req.AgentName
	record := &NoteRecord{
		ID:        int64(len(c.notes) + 1),
		UserID:    req.UserID,
		SessionID: req.SessionID,
		AgentName: req.AgentName,
		Content:   req.Content,
		Metadata:  req.Metadata,
		NoteType:  req.NoteType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	c.notes[key] = record

	return &SaveNoteResponse{
		ID: record.ID,
	}, nil
}

// GetNote 获取笔记
func (c *InMemClient) GetNote(ctx context.Context, req GetNoteRequest) (*NoteRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, record := range c.notes {
		if record.ID == req.ID {
			return record, nil
		}
	}

	return nil, nil
}

// UpdateNote 更新笔记
func (c *InMemClient) UpdateNote(ctx context.Context, req UpdateNoteRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, record := range c.notes {
		if record.ID == req.ID {
			if req.Content != "" {
				record.Content = req.Content
			}
			if req.Metadata != nil {
				record.Metadata = req.Metadata
			}
			record.UpdatedAt = time.Now()
			break
		}
	}

	return nil
}

// DeleteNote 删除笔记
func (c *InMemClient) DeleteNote(ctx context.Context, req DeleteNoteRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, record := range c.notes {
		if record.ID == req.ID {
			delete(c.notes, key)
			break
		}
	}

	return nil
}

// ListNotes 列出笔记
func (c *InMemClient) ListNotes(ctx context.Context, req ListNotesRequest) (*ListNotesResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var notes []NoteRecord
	for _, record := range c.notes {
		if record.UserID == req.UserID {
			if req.SessionID == "" || record.SessionID == req.SessionID {
				if req.AgentName == "" || record.AgentName == req.AgentName {
					notes = append(notes, *record)
				}
			}
		}
	}

	// 应用分页
	start := req.Offset
	end := start + req.Limit
	if req.Limit == 0 {
		end = len(notes)
	}

	if start >= len(notes) {
		notes = []NoteRecord{}
	} else if end > len(notes) {
		notes = notes[start:]
	} else {
		notes = notes[start:end]
	}

	return &ListNotesResponse{
		Notes: notes,
		Total: int64(len(notes)),
	}, nil
}

// GetInMemClient 获取内存数据库客户端
func GetInMemClient(logger *logx.Logger) Client {
	return NewInMemClient(logger)
}
