package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// PersistenceManager 持久化管理器
type PersistenceManager struct {
	config         *config.Config
	logger         *logx.Logger
	client         Client
	initialized    bool
	initialization sync.Once
}

// NewPersistenceManager 创建新的持久化管理器
func NewPersistenceManager(cfg *config.Config, logger *logx.Logger) *PersistenceManager {
	return &PersistenceManager{
		config: cfg,
		logger: logger,
	}
}

// Initialize 初始化持久化管理器
func (pm *PersistenceManager) Initialize() error {
	var err error
	pm.initialization.Do(func() {
		pm.logger.Info(context.Background(), "开始初始化持久化管理器...")

		// 测试数据库连接
		if !pm.testConnection() {
			err = fmt.Errorf("数据库连接测试失败")
			return
		}

		// 初始化数据库客户端
		pm.client, err = GetSupabaseClientWithConfig(pm.config.Database)
		if err != nil {
			pm.logger.Error(context.Background(), "初始化数据库客户端失败", logx.KV("error", err))
			return
		}

		pm.initialized = true
		pm.logger.Info(context.Background(), "持久化管理器初始化完成")

		// 记录系统信息
		pm.logger.Info(context.Background(), "持久化层服务状态:")
		pm.logger.Info(context.Background(), "  - SupabaseClient: ✅ 已初始化")
		pm.logger.Info(context.Background(), "  - 检查点存储: ✅ 已就绪")
		pm.logger.Info(context.Background(), "  - 文件存储: ✅ 已就绪")
		pm.logger.Info(context.Background(), "  - 用户存储: ✅ 已就绪")
		pm.logger.Info(context.Background(), "  - 上下文存储: ✅ 已就绪")
		pm.logger.Info(context.Background(), "  - Notes存储: ✅ 已就绪")
	})

	return err
}

// IsInitialized 检查是否已初始化
func (pm *PersistenceManager) IsInitialized() bool {
	return pm.initialized
}

// EnsureInitialized 确保持久化管理器已初始化
func (pm *PersistenceManager) EnsureInitialized() error {
	if !pm.initialized {
		pm.logger.Info(context.Background(), "持久化管理器未初始化，开始初始化...")
		return pm.Initialize()
	}
	return nil
}

// GetStatus 获取持久化层状态信息
func (pm *PersistenceManager) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"initialized":        pm.initialized,
		"client_available":   false,
		"database_connected": false,
	}

	if pm.client != nil {
		status["client_available"] = true
		status["database_connected"] = pm.client.IsAvailable()
	}

	return status
}

// GetClient 获取数据库客户端
func (pm *PersistenceManager) GetClient() Client {
	return pm.client
}

// testConnection 测试数据库连接
func (pm *PersistenceManager) testConnection() bool {
	return TestConnection()
}

// SaveCheckpoint 保存检查点
func (pm *PersistenceManager) SaveCheckpoint(ctx context.Context, req SaveCheckpointRequest) (*SaveCheckpointResponse, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.SaveCheckpoint(ctx, req)
}

// GetCheckpoint 获取检查点
func (pm *PersistenceManager) GetCheckpoint(ctx context.Context, req GetCheckpointRequest) (*CheckpointRecord, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.GetCheckpoint(ctx, req)
}

// SaveCheckpointWrites 保存检查点写入记录
func (pm *PersistenceManager) SaveCheckpointWrites(ctx context.Context, req SaveCheckpointWritesRequest) error {
	if err := pm.EnsureInitialized(); err != nil {
		return err
	}

	return pm.client.SaveCheckpointWrites(ctx, req)
}

// SaveFile 保存文件
func (pm *PersistenceManager) SaveFile(ctx context.Context, req SaveFileRequest) (*SaveFileResponse, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.SaveFile(ctx, req)
}

// GetFile 获取文件
func (pm *PersistenceManager) GetFile(ctx context.Context, req GetFileRequest) (*FileRecord, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.GetFile(ctx, req)
}

// DeleteFile 删除文件
func (pm *PersistenceManager) DeleteFile(ctx context.Context, req DeleteFileRequest) error {
	if err := pm.EnsureInitialized(); err != nil {
		return err
	}

	return pm.client.DeleteFile(ctx, req)
}

// ListFiles 列出文件
func (pm *PersistenceManager) ListFiles(ctx context.Context, req ListFilesRequest) (*ListFilesResponse, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.ListFiles(ctx, req)
}

// SaveUser 保存用户
func (pm *PersistenceManager) SaveUser(ctx context.Context, req SaveUserRequest) error {
	if err := pm.EnsureInitialized(); err != nil {
		return err
	}

	return pm.client.SaveUser(ctx, req)
}

// GetUser 获取用户
func (pm *PersistenceManager) GetUser(ctx context.Context, req GetUserRequest) (*UserRecord, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.GetUser(ctx, req)
}

// UpdateUserStats 更新用户统计
func (pm *PersistenceManager) UpdateUserStats(ctx context.Context, req UpdateUserStatsRequest) error {
	if err := pm.EnsureInitialized(); err != nil {
		return err
	}

	return pm.client.UpdateUserStats(ctx, req)
}

// SaveContext 保存上下文
func (pm *PersistenceManager) SaveContext(ctx context.Context, req SaveContextRequest) error {
	if err := pm.EnsureInitialized(); err != nil {
		return err
	}

	return pm.client.SaveContext(ctx, req)
}

// GetContext 获取上下文
func (pm *PersistenceManager) GetContext(ctx context.Context, req GetContextRequest) (*ContextRecord, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.GetContext(ctx, req)
}

// SaveNote 保存笔记
func (pm *PersistenceManager) SaveNote(ctx context.Context, req SaveNoteRequest) (*SaveNoteResponse, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.SaveNote(ctx, req)
}

// GetNote 获取笔记
func (pm *PersistenceManager) GetNote(ctx context.Context, req GetNoteRequest) (*NoteRecord, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.GetNote(ctx, req)
}

// UpdateNote 更新笔记
func (pm *PersistenceManager) UpdateNote(ctx context.Context, req UpdateNoteRequest) error {
	if err := pm.EnsureInitialized(); err != nil {
		return err
	}

	return pm.client.UpdateNote(ctx, req)
}

// DeleteNote 删除笔记
func (pm *PersistenceManager) DeleteNote(ctx context.Context, req DeleteNoteRequest) error {
	if err := pm.EnsureInitialized(); err != nil {
		return err
	}

	return pm.client.DeleteNote(ctx, req)
}

// ListNotes 列出笔记
func (pm *PersistenceManager) ListNotes(ctx context.Context, req ListNotesRequest) (*ListNotesResponse, error) {
	if err := pm.EnsureInitialized(); err != nil {
		return nil, err
	}

	return pm.client.ListNotes(ctx, req)
}

// 全局持久化管理器实例
var (
	globalPersistenceManager *PersistenceManager
	globalPersistenceOnce    sync.Once
)

// GetPersistenceManager 获取全局持久化管理器
func GetPersistenceManager() *PersistenceManager {
	return globalPersistenceManager
}

// InitializePersistenceManager 初始化全局持久化管理器
func InitializePersistenceManager(cfg *config.Config, logger *logx.Logger) (*PersistenceManager, error) {
	var err error
	globalPersistenceOnce.Do(func() {
		globalPersistenceManager = NewPersistenceManager(cfg, logger)
		err = globalPersistenceManager.Initialize()
	})

	return globalPersistenceManager, err
}

// InitPersistenceLayer 初始化持久化层（兼容Python接口）
func InitPersistenceLayer() error {
	// 这里可以添加额外的初始化逻辑
	return nil
}
