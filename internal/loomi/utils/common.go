package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// StringUtils 字符串工具类
type StringUtils struct{}

// IsEmpty 检查字符串是否为空
func (su *StringUtils) IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotEmpty 检查字符串是否不为空
func (su *StringUtils) IsNotEmpty(s string) bool {
	return !su.IsEmpty(s)
}

// Contains 检查字符串是否包含子字符串
func (su *StringUtils) Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// StartsWith 检查字符串是否以指定前缀开始
func (su *StringUtils) StartsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// EndsWith 检查字符串是否以指定后缀结束
func (su *StringUtils) EndsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// Trim 去除字符串首尾空白字符
func (su *StringUtils) Trim(s string) string {
	return strings.TrimSpace(s)
}

// ToLower 转换为小写
func (su *StringUtils) ToLower(s string) string {
	return strings.ToLower(s)
}

// ToUpper 转换为大写
func (su *StringUtils) ToUpper(s string) string {
	return strings.ToUpper(s)
}

// Replace 替换字符串
func (su *StringUtils) Replace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// Split 分割字符串
func (su *StringUtils) Split(s, sep string) []string {
	return strings.Split(s, sep)
}

// Join 连接字符串
func (su *StringUtils) Join(elems []string, sep string) string {
	return strings.Join(elems, sep)
}

// IsValidEmail 验证邮箱格式
func (su *StringUtils) IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// IsValidURL 验证URL格式
func (su *StringUtils) IsValidURL(url string) bool {
	pattern := `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

// IsValidPhone 验证手机号格式
func (su *StringUtils) IsValidPhone(phone string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// RemoveSpecialChars 移除特殊字符
func (su *StringUtils) RemoveSpecialChars(s string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9\u4e00-\u9fa5]`).ReplaceAllString(s, "")
}

// CamelToSnake 驼峰转蛇形
func (su *StringUtils) CamelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// SnakeToCamel 蛇形转驼峰
func (su *StringUtils) SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// NumberUtils 数字工具类
type NumberUtils struct{}

// IsNumeric 检查是否为数字
func (nu *NumberUtils) IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// ToInt 转换为整数
func (nu *NumberUtils) ToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ToInt64 转换为64位整数
func (nu *NumberUtils) ToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ToFloat64 转换为64位浮点数
func (nu *NumberUtils) ToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// ToString 数字转字符串
func (nu *NumberUtils) ToString(i interface{}) string {
	switch v := i.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// FormatNumber 格式化数字
func (nu *NumberUtils) FormatNumber(num float64, decimals int) string {
	return fmt.Sprintf("%."+strconv.Itoa(decimals)+"f", num)
}

// RandomUtils 随机数工具类
type RandomUtils struct{}

// RandomString 生成随机字符串
func (ru *RandomUtils) RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

// RandomInt 生成随机整数
func (ru *RandomUtils) RandomInt(min, max int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return min + int(num.Int64())
}

// RandomFloat64 生成随机浮点数
func (ru *RandomUtils) RandomFloat64(min, max float64) float64 {
	num, _ := rand.Int(rand.Reader, big.NewInt(1<<53))
	return min + (max-min)*float64(num.Int64())/(1<<53)
}

// UUID 生成UUID
func (ru *RandomUtils) UUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	// 设置版本和变体
	b[6] = (b[6] & 0x0f) | 0x40 // 版本4
	b[8] = (b[8] & 0x3f) | 0x80 // 变体

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// CryptoUtils 加密工具类
type CryptoUtils struct{}

// MD5 计算MD5哈希
func (cu *CryptoUtils) MD5(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// SHA256 计算SHA256哈希
func (cu *CryptoUtils) SHA256(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// Base64Encode Base64编码
func (cu *CryptoUtils) Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Decode Base64解码
func (cu *CryptoUtils) Base64Decode(s string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// FileUtils 文件工具类
type FileUtils struct{}

// Exists 检查文件是否存在
func (fu *FileUtils) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsFile 检查是否为文件
func (fu *FileUtils) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// IsDir 检查是否为目录
func (fu *FileUtils) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// GetFileSize 获取文件大小
func (fu *FileUtils) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileExt 获取文件扩展名
func (fu *FileUtils) GetFileExt(path string) string {
	return filepath.Ext(path)
}

// GetFileName 获取文件名
func (fu *FileUtils) GetFileName(path string) string {
	return filepath.Base(path)
}

// GetDirName 获取目录名
func (fu *FileUtils) GetDirName(path string) string {
	return filepath.Dir(path)
}

// JoinPath 连接路径
func (fu *FileUtils) JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// CreateDir 创建目录
func (fu *FileUtils) CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ReadFile 读取文件内容
func (fu *FileUtils) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile 写入文件内容
func (fu *FileUtils) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// CopyFile 复制文件
func (fu *FileUtils) CopyFile(src, dst string) error {
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
	return err
}

// DeleteFile 删除文件
func (fu *FileUtils) DeleteFile(path string) error {
	return os.Remove(path)
}

// ListFiles 列出目录中的文件
func (fu *FileUtils) ListFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// TimeUtils 时间工具类
type TimeUtils struct{}

// Now 获取当前时间
func (tu *TimeUtils) Now() time.Time {
	return time.Now()
}

// NowUnix 获取当前时间戳
func (tu *TimeUtils) NowUnix() int64 {
	return time.Now().Unix()
}

// NowUnixNano 获取当前纳秒时间戳
func (tu *TimeUtils) NowUnixNano() int64 {
	return time.Now().UnixNano()
}

// FormatTime 格式化时间
func (tu *TimeUtils) FormatTime(t time.Time, layout string) string {
	return t.Format(layout)
}

// ParseTime 解析时间字符串
func (tu *TimeUtils) ParseTime(s, layout string) (time.Time, error) {
	return time.Parse(layout, s)
}

// AddDays 添加天数
func (tu *TimeUtils) AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddHours 添加小时
func (tu *TimeUtils) AddHours(t time.Time, hours int) time.Time {
	return t.Add(time.Duration(hours) * time.Hour)
}

// AddMinutes 添加分钟
func (tu *TimeUtils) AddMinutes(t time.Time, minutes int) time.Time {
	return t.Add(time.Duration(minutes) * time.Minute)
}

// AddSeconds 添加秒数
func (tu *TimeUtils) AddSeconds(t time.Time, seconds int) time.Time {
	return t.Add(time.Duration(seconds) * time.Second)
}

// IsSameDay 检查是否为同一天
func (tu *TimeUtils) IsSameDay(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.YearDay() == t2.YearDay()
}

// GetStartOfDay 获取一天的开始时间
func (tu *TimeUtils) GetStartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// GetEndOfDay 获取一天的结束时间
func (tu *TimeUtils) GetEndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// GetStartOfMonth 获取月初时间
func (tu *TimeUtils) GetStartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// GetEndOfMonth 获取月末时间
func (tu *TimeUtils) GetEndOfMonth(t time.Time) time.Time {
	return tu.GetStartOfMonth(t).AddDate(0, 1, -1)
}

// GetStartOfYear 获取年初时间
func (tu *TimeUtils) GetStartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// GetEndOfYear 获取年末时间
func (tu *TimeUtils) GetEndOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
}

// JSONUtils JSON工具类
type JSONUtils struct{}

// ToJSON 转换为JSON字符串
func (ju *JSONUtils) ToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToJSONPretty 转换为格式化的JSON字符串
func (ju *JSONUtils) ToJSONPretty(v interface{}) (string, error) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON 从JSON字符串解析
func (ju *JSONUtils) FromJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

// FromJSONBytes 从JSON字节数组解析
func (ju *JSONUtils) FromJSONBytes(bytes []byte, v interface{}) error {
	return json.Unmarshal(bytes, v)
}

// ArrayUtils 数组工具类
type ArrayUtils struct{}

// Contains 检查数组是否包含元素
func (au *ArrayUtils) Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// IndexOf 获取元素在数组中的索引
func (au *ArrayUtils) IndexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// Remove 从数组中移除元素
func (au *ArrayUtils) Remove(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// Unique 去重
func (au *ArrayUtils) Unique(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

// Sort 排序
func (au *ArrayUtils) Sort(slice []string) []string {
	result := make([]string, len(slice))
	copy(result, slice)
	sort.Strings(result)
	return result
}

// Reverse 反转数组
func (au *ArrayUtils) Reverse(slice []string) []string {
	result := make([]string, len(slice))
	for i, j := 0, len(slice)-1; i < len(slice); i, j = i+1, j-1 {
		result[i] = slice[j]
	}
	return result
}

// Chunk 分块
func (au *ArrayUtils) Chunk(slice []string, size int) [][]string {
	var result [][]string
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		result = append(result, slice[i:end])
	}
	return result
}

// 全局工具类实例
var (
	StringUtil = &StringUtils{}
	NumberUtil = &NumberUtils{}
	RandomUtil = &RandomUtils{}
	CryptoUtil = &CryptoUtils{}
	FileUtil   = &FileUtils{}
	TimeUtil   = &TimeUtils{}
	JSONUtil   = &JSONUtils{}
	ArrayUtil  = &ArrayUtils{}
)
