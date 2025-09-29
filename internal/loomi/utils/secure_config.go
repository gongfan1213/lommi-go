package utils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"

	"golang.org/x/crypto/pbkdf2"
)

// SecureConfigManager 安全配置管理器
type SecureConfigManager struct {
	logger        log.Logger
	configDir     string
	masterKeyEnv  string
	saltFile      string
	originalFile  string
	encryptedFile string
	binaryFile    string
}

// NewSecureConfigManager 创建安全配置管理器
func NewSecureConfigManager(logger log.Logger, configDir string) *SecureConfigManager {
	return &SecureConfigManager{
		logger:        logger,
		configDir:     configDir,
		masterKeyEnv:  "BLUEPLAN_MASTER_KEY",
		saltFile:      filepath.Join(configDir, ".config_salt"),
		originalFile:  filepath.Join(configDir, "bluelab.json"),
		encryptedFile: filepath.Join(configDir, "bluelab.enc"),
		binaryFile:    filepath.Join(configDir, "bluelab.bin"),
	}
}

// Initialize 初始化安全配置管理器
func (scm *SecureConfigManager) Initialize(ctx context.Context) error {
	scm.logger.Info(ctx, "初始化安全配置管理器", "config_dir", scm.configDir)

	// 确保配置目录存在
	err := os.MkdirAll(scm.configDir, 0755)
	if err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 检查主密钥
	masterKey := os.Getenv(scm.masterKeyEnv)
	if masterKey == "" {
		scm.logger.Warn(ctx, "未设置主密钥环境变量", "env_var", scm.masterKeyEnv)
		return fmt.Errorf("主密钥环境变量未设置: %s", scm.masterKeyEnv)
	}

	scm.logger.Info(ctx, "安全配置管理器初始化成功")
	return nil
}

// EncryptConfig 加密配置文件
func (scm *SecureConfigManager) EncryptConfig(ctx context.Context, configData map[string]interface{}) error {
	scm.logger.Info(ctx, "开始加密配置文件")

	// 检查原始配置文件是否存在
	if _, err := os.Stat(scm.originalFile); os.IsNotExist(err) {
		return fmt.Errorf("原始配置文件不存在: %s", scm.originalFile)
	}

	// 读取原始配置
	configBytes, err := os.ReadFile(scm.originalFile)
	if err != nil {
		return fmt.Errorf("读取原始配置文件失败: %w", err)
	}

	// 生成盐值
	salt, err := scm.generateSalt()
	if err != nil {
		return fmt.Errorf("生成盐值失败: %w", err)
	}

	// 保存盐值
	err = os.WriteFile(scm.saltFile, salt, 0600)
	if err != nil {
		return fmt.Errorf("保存盐值失败: %w", err)
	}

	// 获取主密钥
	masterKey := os.Getenv(scm.masterKeyEnv)
	if masterKey == "" {
		return fmt.Errorf("主密钥环境变量未设置: %s", scm.masterKeyEnv)
	}

	// 生成加密密钥
	key := pbkdf2.Key([]byte(masterKey), salt, 4096, 32, sha256.New)

	// 加密数据
	encryptedData, err := scm.encryptData(configBytes, key)
	if err != nil {
		return fmt.Errorf("加密数据失败: %w", err)
	}

	// 保存加密文件
	err = os.WriteFile(scm.encryptedFile, encryptedData, 0600)
	if err != nil {
		return fmt.Errorf("保存加密文件失败: %w", err)
	}

	scm.logger.Info(ctx, "配置文件加密成功", "encrypted_file", scm.encryptedFile)
	return nil
}

// DecryptConfig 解密配置文件
func (scm *SecureConfigManager) DecryptConfig(ctx context.Context) (map[string]interface{}, error) {
	scm.logger.Info(ctx, "开始解密配置文件")

	// 检查加密文件是否存在
	if _, err := os.Stat(scm.encryptedFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("加密配置文件不存在: %s", scm.encryptedFile)
	}

	// 检查盐值文件是否存在
	if _, err := os.Stat(scm.saltFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("盐值文件不存在: %s", scm.saltFile)
	}

	// 读取加密文件
	encryptedData, err := os.ReadFile(scm.encryptedFile)
	if err != nil {
		return nil, fmt.Errorf("读取加密文件失败: %w", err)
	}

	// 读取盐值
	salt, err := os.ReadFile(scm.saltFile)
	if err != nil {
		return nil, fmt.Errorf("读取盐值失败: %w", err)
	}

	// 获取主密钥
	masterKey := os.Getenv(scm.masterKeyEnv)
	if masterKey == "" {
		return nil, fmt.Errorf("主密钥环境变量未设置: %s", scm.masterKeyEnv)
	}

	// 生成解密密钥
	key := pbkdf2.Key([]byte(masterKey), salt, 4096, 32, sha256.New)

	// 解密数据
	decryptedData, err := scm.decryptData(encryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("解密数据失败: %w", err)
	}

	// 解析JSON
	var config map[string]interface{}
	err = json.Unmarshal(decryptedData, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	scm.logger.Info(ctx, "配置文件解密成功")
	return config, nil
}

// CompileToBinary 编译为二进制文件
func (scm *SecureConfigManager) CompileToBinary(ctx context.Context, configData map[string]interface{}) error {
	scm.logger.Info(ctx, "开始编译配置文件为二进制")

	// 序列化配置数据
	configBytes, err := json.Marshal(configData)
	if err != nil {
		return fmt.Errorf("序列化配置数据失败: %w", err)
	}

	// 生成盐值
	salt, err := scm.generateSalt()
	if err != nil {
		return fmt.Errorf("生成盐值失败: %w", err)
	}

	// 获取主密钥
	masterKey := os.Getenv(scm.masterKeyEnv)
	if masterKey == "" {
		return fmt.Errorf("主密钥环境变量未设置: %s", scm.masterKeyEnv)
	}

	// 生成加密密钥
	key := pbkdf2.Key([]byte(masterKey), salt, 4096, 32, sha256.New)

	// 加密数据
	encryptedData, err := scm.encryptData(configBytes, key)
	if err != nil {
		return fmt.Errorf("加密数据失败: %w", err)
	}

	// 创建二进制文件结构
	binaryData := map[string]interface{}{
		"version":    "1.0",
		"salt":       base64.StdEncoding.EncodeToString(salt),
		"data":       base64.StdEncoding.EncodeToString(encryptedData),
		"created_at": ctx.Value("timestamp"),
	}

	// 序列化二进制数据
	binaryBytes, err := json.Marshal(binaryData)
	if err != nil {
		return fmt.Errorf("序列化二进制数据失败: %w", err)
	}

	// 保存二进制文件
	err = os.WriteFile(scm.binaryFile, binaryBytes, 0600)
	if err != nil {
		return fmt.Errorf("保存二进制文件失败: %w", err)
	}

	scm.logger.Info(ctx, "配置文件编译为二进制成功", "binary_file", scm.binaryFile)
	return nil
}

// LoadSecureConfig 加载安全配置
func (scm *SecureConfigManager) LoadSecureConfig(ctx context.Context) (map[string]interface{}, error) {
	scm.logger.Info(ctx, "开始加载安全配置")

	// 优先尝试从二进制文件加载
	if _, err := os.Stat(scm.binaryFile); err == nil {
		config, err := scm.loadFromBinary(ctx)
		if err != nil {
			scm.logger.Warn(ctx, "从二进制文件加载失败，尝试从加密文件加载", "error", err)
		} else {
			return config, nil
		}
	}

	// 尝试从加密文件加载
	if _, err := os.Stat(scm.encryptedFile); err == nil {
		config, err := scm.DecryptConfig(ctx)
		if err != nil {
			scm.logger.Warn(ctx, "从加密文件加载失败，尝试从原始文件加载", "error", err)
		} else {
			return config, nil
		}
	}

	// 最后尝试从原始文件加载
	if _, err := os.Stat(scm.originalFile); err == nil {
		config, err := scm.loadFromOriginal(ctx)
		if err != nil {
			return nil, fmt.Errorf("从原始文件加载失败: %w", err)
		}
		return config, nil
	}

	return nil, fmt.Errorf("未找到任何配置文件")
}

// ValidateConfig 验证配置
func (scm *SecureConfigManager) ValidateConfig(ctx context.Context, config map[string]interface{}) error {
	scm.logger.Info(ctx, "开始验证配置")

	// 检查必需的配置项
	requiredFields := []string{"database", "redis", "llm", "security"}
	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("缺少必需的配置项: %s", field)
		}
	}

	// 验证数据库配置
	if dbConfig, ok := config["database"].(map[string]interface{}); ok {
		if _, exists := dbConfig["supabase_url"]; !exists {
			return fmt.Errorf("缺少数据库配置: supabase_url")
		}
		if _, exists := dbConfig["supabase_key"]; !exists {
			return fmt.Errorf("缺少数据库配置: supabase_key")
		}
	}

	// 验证LLM配置
	if llmConfig, ok := config["llm"].(map[string]interface{}); ok {
		if _, exists := llmConfig["default_provider"]; !exists {
			return fmt.Errorf("缺少LLM配置: default_provider")
		}
	}

	scm.logger.Info(ctx, "配置验证成功")
	return nil
}

// BackupConfig 备份配置
func (scm *SecureConfigManager) BackupConfig(ctx context.Context, backupDir string) error {
	scm.logger.Info(ctx, "开始备份配置", "backup_dir", backupDir)

	// 创建备份目录
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 备份文件列表
	files := []string{
		scm.originalFile,
		scm.encryptedFile,
		scm.binaryFile,
		scm.saltFile,
	}

	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			backupFile := filepath.Join(backupDir, filepath.Base(file))
			err := scm.copyFile(file, backupFile)
			if err != nil {
				scm.logger.Error(ctx, "备份文件失败", "file", file, "error", err)
			} else {
				scm.logger.Info(ctx, "文件备份成功", "file", file, "backup", backupFile)
			}
		}
	}

	scm.logger.Info(ctx, "配置备份完成")
	return nil
}

// 私有方法

// generateSalt 生成盐值
func (scm *SecureConfigManager) generateSalt() ([]byte, error) {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// encryptData 加密数据
func (scm *SecureConfigManager) encryptData(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptData 解密数据
func (scm *SecureConfigManager) decryptData(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("密文长度不足")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// loadFromBinary 从二进制文件加载
func (scm *SecureConfigManager) loadFromBinary(ctx context.Context) (map[string]interface{}, error) {
	// 读取二进制文件
	binaryBytes, err := os.ReadFile(scm.binaryFile)
	if err != nil {
		return nil, fmt.Errorf("读取二进制文件失败: %w", err)
	}

	// 解析二进制数据
	var binaryData map[string]interface{}
	err = json.Unmarshal(binaryBytes, &binaryData)
	if err != nil {
		return nil, fmt.Errorf("解析二进制数据失败: %w", err)
	}

	// 获取盐值
	saltStr, ok := binaryData["salt"].(string)
	if !ok {
		return nil, fmt.Errorf("二进制文件中缺少盐值")
	}

	salt, err := base64.StdEncoding.DecodeString(saltStr)
	if err != nil {
		return nil, fmt.Errorf("解码盐值失败: %w", err)
	}

	// 获取加密数据
	dataStr, ok := binaryData["data"].(string)
	if !ok {
		return nil, fmt.Errorf("二进制文件中缺少数据")
	}

	encryptedData, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("解码加密数据失败: %w", err)
	}

	// 获取主密钥
	masterKey := os.Getenv(scm.masterKeyEnv)
	if masterKey == "" {
		return nil, fmt.Errorf("主密钥环境变量未设置: %s", scm.masterKeyEnv)
	}

	// 生成解密密钥
	key := pbkdf2.Key([]byte(masterKey), salt, 4096, 32, sha256.New)

	// 解密数据
	decryptedData, err := scm.decryptData(encryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("解密数据失败: %w", err)
	}

	// 解析配置
	var config map[string]interface{}
	err = json.Unmarshal(decryptedData, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return config, nil
}

// loadFromOriginal 从原始文件加载
func (scm *SecureConfigManager) loadFromOriginal(ctx context.Context) (map[string]interface{}, error) {
	// 读取原始文件
	configBytes, err := os.ReadFile(scm.originalFile)
	if err != nil {
		return nil, fmt.Errorf("读取原始配置文件失败: %w", err)
	}

	// 解析配置
	var config map[string]interface{}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return config, nil
}

// copyFile 复制文件
func (scm *SecureConfigManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
