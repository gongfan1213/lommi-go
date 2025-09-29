package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Validator 验证器
type Validator struct {
	errors []string
}

// NewValidator 创建新的验证器
func NewValidator() *Validator {
	return &Validator{
		errors: make([]string, 0),
	}
}

// AddError 添加错误
func (v *Validator) AddError(message string) {
	v.errors = append(v.errors, message)
}

// HasErrors 检查是否有错误
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// GetErrors 获取所有错误
func (v *Validator) GetErrors() []string {
	return v.errors
}

// GetErrorString 获取错误字符串
func (v *Validator) GetErrorString() string {
	return strings.Join(v.errors, "; ")
}

// Clear 清除所有错误
func (v *Validator) Clear() {
	v.errors = make([]string, 0)
}

// Validate 验证并返回错误
func (v *Validator) Validate() error {
	if v.HasErrors() {
		return errors.New(v.GetErrorString())
	}
	return nil
}

// Required 验证必填字段
func (v *Validator) Required(value interface{}, fieldName string) *Validator {
	if value == nil {
		v.AddError(fmt.Sprintf("%s 是必填字段", fieldName))
		return v
	}

	switch val := value.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			v.AddError(fmt.Sprintf("%s 不能为空", fieldName))
		}
	case []string:
		if len(val) == 0 {
			v.AddError(fmt.Sprintf("%s 不能为空", fieldName))
		}
	case map[string]interface{}:
		if len(val) == 0 {
			v.AddError(fmt.Sprintf("%s 不能为空", fieldName))
		}
	}

	return v
}

// MinLength 验证最小长度
func (v *Validator) MinLength(value string, minLength int, fieldName string) *Validator {
	if len(value) < minLength {
		v.AddError(fmt.Sprintf("%s 长度不能少于 %d 个字符", fieldName, minLength))
	}
	return v
}

// MaxLength 验证最大长度
func (v *Validator) MaxLength(value string, maxLength int, fieldName string) *Validator {
	if len(value) > maxLength {
		v.AddError(fmt.Sprintf("%s 长度不能超过 %d 个字符", fieldName, maxLength))
	}
	return v
}

// Length 验证长度范围
func (v *Validator) Length(value string, minLength, maxLength int, fieldName string) *Validator {
	length := len(value)
	if length < minLength || length > maxLength {
		v.AddError(fmt.Sprintf("%s 长度必须在 %d 到 %d 个字符之间", fieldName, minLength, maxLength))
	}
	return v
}

// Min 验证最小值
func (v *Validator) Min(value int, min int, fieldName string) *Validator {
	if value < min {
		v.AddError(fmt.Sprintf("%s 不能小于 %d", fieldName, min))
	}
	return v
}

// Max 验证最大值
func (v *Validator) Max(value int, max int, fieldName string) *Validator {
	if value > max {
		v.AddError(fmt.Sprintf("%s 不能大于 %d", fieldName, max))
	}
	return v
}

// Range 验证范围
func (v *Validator) Range(value int, min, max int, fieldName string) *Validator {
	if value < min || value > max {
		v.AddError(fmt.Sprintf("%s 必须在 %d 到 %d 之间", fieldName, min, max))
	}
	return v
}

// MinFloat 验证浮点数最小值
func (v *Validator) MinFloat(value float64, min float64, fieldName string) *Validator {
	if value < min {
		v.AddError(fmt.Sprintf("%s 不能小于 %.2f", fieldName, min))
	}
	return v
}

// MaxFloat 验证浮点数最大值
func (v *Validator) MaxFloat(value float64, max float64, fieldName string) *Validator {
	if value > max {
		v.AddError(fmt.Sprintf("%s 不能大于 %.2f", fieldName, max))
	}
	return v
}

// RangeFloat 验证浮点数范围
func (v *Validator) RangeFloat(value float64, min, max float64, fieldName string) *Validator {
	if value < min || value > max {
		v.AddError(fmt.Sprintf("%s 必须在 %.2f 到 %.2f 之间", fieldName, min, max))
	}
	return v
}

// Email 验证邮箱格式
func (v *Validator) Email(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// URL 验证URL格式
func (v *Validator) URL(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// Phone 验证手机号格式
func (v *Validator) Phone(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^1[3-9]\d{9}$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// Numeric 验证数字格式
func (v *Validator) Numeric(value string, fieldName string) *Validator {
	if value != "" {
		_, err := strconv.Atoi(value)
		if err != nil {
			v.AddError(fmt.Sprintf("%s 必须是数字", fieldName))
		}
	}
	return v
}

// Float 验证浮点数格式
func (v *Validator) Float(value string, fieldName string) *Validator {
	if value != "" {
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			v.AddError(fmt.Sprintf("%s 必须是数字", fieldName))
		}
	}
	return v
}

// Alpha 验证字母格式
func (v *Validator) Alpha(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^[a-zA-Z]+$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 只能包含字母", fieldName))
		}
	}
	return v
}

// AlphaNumeric 验证字母数字格式
func (v *Validator) AlphaNumeric(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^[a-zA-Z0-9]+$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 只能包含字母和数字", fieldName))
		}
	}
	return v
}

// Chinese 验证中文字符
func (v *Validator) Chinese(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^[\u4e00-\u9fa5]+$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 只能包含中文字符", fieldName))
		}
	}
	return v
}

// IP 验证IP地址格式
func (v *Validator) IP(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// IPv6 验证IPv6地址格式
func (v *Validator) IPv6(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// UUID 验证UUID格式
func (v *Validator) UUID(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// Date 验证日期格式
func (v *Validator) Date(value string, format string, fieldName string) *Validator {
	if value != "" {
		// 这里应该根据format验证日期格式
		// 简化实现，只验证基本格式
		pattern := `^\d{4}-\d{2}-\d{2}$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// Time 验证时间格式
func (v *Validator) Time(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^\d{2}:\d{2}:\d{2}$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// DateTime 验证日期时间格式
func (v *Validator) DateTime(value string, fieldName string) *Validator {
	if value != "" {
		pattern := `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// In 验证值是否在指定范围内
func (v *Validator) In(value string, options []string, fieldName string) *Validator {
	if value != "" {
		found := false
		for _, option := range options {
			if value == option {
				found = true
				break
			}
		}
		if !found {
			v.AddError(fmt.Sprintf("%s 必须是以下值之一: %s", fieldName, strings.Join(options, ", ")))
		}
	}
	return v
}

// NotIn 验证值是否不在指定范围内
func (v *Validator) NotIn(value string, options []string, fieldName string) *Validator {
	if value != "" {
		for _, option := range options {
			if value == option {
				v.AddError(fmt.Sprintf("%s 不能是以下值之一: %s", fieldName, strings.Join(options, ", ")))
				break
			}
		}
	}
	return v
}

// Equal 验证值是否相等
func (v *Validator) Equal(value1, value2 string, fieldName string) *Validator {
	if value1 != value2 {
		v.AddError(fmt.Sprintf("%s 不匹配", fieldName))
	}
	return v
}

// NotEqual 验证值是否不相等
func (v *Validator) NotEqual(value1, value2 string, fieldName string) *Validator {
	if value1 == value2 {
		v.AddError(fmt.Sprintf("%s 不能相同", fieldName))
	}
	return v
}

// Contains 验证是否包含子字符串
func (v *Validator) Contains(value, substr string, fieldName string) *Validator {
	if value != "" && substr != "" {
		if !strings.Contains(value, substr) {
			v.AddError(fmt.Sprintf("%s 必须包含 %s", fieldName, substr))
		}
	}
	return v
}

// NotContains 验证是否不包含子字符串
func (v *Validator) NotContains(value, substr string, fieldName string) *Validator {
	if value != "" && substr != "" {
		if strings.Contains(value, substr) {
			v.AddError(fmt.Sprintf("%s 不能包含 %s", fieldName, substr))
		}
	}
	return v
}

// StartsWith 验证是否以指定字符串开始
func (v *Validator) StartsWith(value, prefix string, fieldName string) *Validator {
	if value != "" && prefix != "" {
		if !strings.HasPrefix(value, prefix) {
			v.AddError(fmt.Sprintf("%s 必须以 %s 开始", fieldName, prefix))
		}
	}
	return v
}

// EndsWith 验证是否以指定字符串结束
func (v *Validator) EndsWith(value, suffix string, fieldName string) *Validator {
	if value != "" && suffix != "" {
		if !strings.HasSuffix(value, suffix) {
			v.AddError(fmt.Sprintf("%s 必须以 %s 结束", fieldName, suffix))
		}
	}
	return v
}

// Regex 验证正则表达式
func (v *Validator) Regex(value, pattern string, fieldName string) *Validator {
	if value != "" && pattern != "" {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			v.AddError(fmt.Sprintf("%s 正则表达式错误: %s", fieldName, err.Error()))
		} else if !matched {
			v.AddError(fmt.Sprintf("%s 格式不正确", fieldName))
		}
	}
	return v
}

// Custom 自定义验证
func (v *Validator) Custom(value interface{}, validator func(interface{}) bool, fieldName string) *Validator {
	if !validator(value) {
		v.AddError(fmt.Sprintf("%s 验证失败", fieldName))
	}
	return v
}

// Password 验证密码强度
func (v *Validator) Password(value string, fieldName string) *Validator {
	if value != "" {
		if len(value) < 8 {
			v.AddError(fmt.Sprintf("%s 长度不能少于8个字符", fieldName))
		}

		hasUpper := false
		hasLower := false
		hasDigit := false
		hasSpecial := false

		for _, char := range value {
			switch {
			case unicode.IsUpper(char):
				hasUpper = true
			case unicode.IsLower(char):
				hasLower = true
			case unicode.IsDigit(char):
				hasDigit = true
			case unicode.IsPunct(char) || unicode.IsSymbol(char):
				hasSpecial = true
			}
		}

		if !hasUpper {
			v.AddError(fmt.Sprintf("%s 必须包含大写字母", fieldName))
		}
		if !hasLower {
			v.AddError(fmt.Sprintf("%s 必须包含小写字母", fieldName))
		}
		if !hasDigit {
			v.AddError(fmt.Sprintf("%s 必须包含数字", fieldName))
		}
		if !hasSpecial {
			v.AddError(fmt.Sprintf("%s 必须包含特殊字符", fieldName))
		}
	}
	return v
}

// 全局验证器实例
var (
	ValidatorUtil = NewValidator()
)

// ValidateStruct 验证结构体
func ValidateStruct(data interface{}) error {
	validator := NewValidator()

	// 这里应该使用反射来验证结构体字段
	// 简化实现，只返回nil
	return validator.Validate()
}

// ValidateMap 验证Map
func ValidateMap(data map[string]interface{}, rules map[string][]ValidationRule) error {
	validator := NewValidator()

	for field, rules := range rules {
		value, exists := data[field]
		if !exists {
			continue
		}

		for _, rule := range rules {
			rule.Validate(validator, field, value)
		}
	}

	return validator.Validate()
}

// ValidationRule 验证规则接口
type ValidationRule interface {
	Validate(validator *Validator, fieldName string, value interface{})
}

// RequiredRule 必填规则
type RequiredRule struct{}

func (r RequiredRule) Validate(validator *Validator, fieldName string, value interface{}) {
	validator.Required(value, fieldName)
}

// MinLengthRule 最小长度规则
type MinLengthRule struct {
	MinLength int
}

func (r MinLengthRule) Validate(validator *Validator, fieldName string, value interface{}) {
	if str, ok := value.(string); ok {
		validator.MinLength(str, r.MinLength, fieldName)
	}
}

// MaxLengthRule 最大长度规则
type MaxLengthRule struct {
	MaxLength int
}

func (r MaxLengthRule) Validate(validator *Validator, fieldName string, value interface{}) {
	if str, ok := value.(string); ok {
		validator.MaxLength(str, r.MaxLength, fieldName)
	}
}

// EmailRule 邮箱规则
type EmailRule struct{}

func (r EmailRule) Validate(validator *Validator, fieldName string, value interface{}) {
	if str, ok := value.(string); ok {
		validator.Email(str, fieldName)
	}
}

// PhoneRule 手机号规则
type PhoneRule struct{}

func (r PhoneRule) Validate(validator *Validator, fieldName string, value interface{}) {
	if str, ok := value.(string); ok {
		validator.Phone(str, fieldName)
	}
}
