package alerts

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/config"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// EmailAlert 邮件告警客户端
type EmailAlert struct {
	logger    log.Logger
	config    *config.Config
	smtpHost  string
	smtpPort  string
	username  string
	password  string
	fromEmail string
	toEmails  []string
	enabled   bool
}

// NewEmailAlert 创建邮件告警客户端
func NewEmailAlert(logger log.Logger, config *config.Config) *EmailAlert {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM_EMAIL")
	toEmailsStr := os.Getenv("SMTP_TO_EMAILS")

	var toEmails []string
	if toEmailsStr != "" {
		toEmails = strings.Split(toEmailsStr, ",")
		for i, email := range toEmails {
			toEmails[i] = strings.TrimSpace(email)
		}
	}

	enabled := smtpHost != "" && smtpPort != "" && username != "" && password != "" && fromEmail != "" && len(toEmails) > 0 && os.Getenv("EMAIL_ALERT_ENABLED") != "false"

	return &EmailAlert{
		logger:    logger,
		config:    config,
		smtpHost:  smtpHost,
		smtpPort:  smtpPort,
		username:  username,
		password:  password,
		fromEmail: fromEmail,
		toEmails:  toEmails,
		enabled:   enabled,
	}
}

// Send 发送邮件告警
func (ea *EmailAlert) Send(ctx context.Context, message *AlertMessage) error {
	if !ea.enabled {
		ea.logger.Debug(ctx, "邮件告警已禁用，跳过发送")
		return nil
	}

	ea.logger.Info(ctx, "发送邮件告警",
		"alert_id", message.ID,
		"title", message.Title,
		"level", message.Level,
		"to_emails", ea.toEmails)

	// 构建邮件内容
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(string(message.Level)), message.Title)
	body := ea.buildEmailBody(message)

	// 构建邮件消息
	emailMessage := ea.buildEmailMessage(subject, body, ea.fromEmail, ea.toEmails)

	// 发送邮件
	err := ea.sendEmail(emailMessage)
	if err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	ea.logger.Info(ctx, "邮件告警发送成功", "alert_id", message.ID)
	return nil
}

// GetName 获取客户端名称
func (ea *EmailAlert) GetName() string {
	return "email"
}

// IsEnabled 检查是否启用
func (ea *EmailAlert) IsEnabled() bool {
	return ea.enabled
}

// GetConfig 获取配置
func (ea *EmailAlert) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"smtp_host":  ea.smtpHost,
		"smtp_port":  ea.smtpPort,
		"username":   ea.username,
		"from_email": ea.fromEmail,
		"to_emails":  ea.toEmails,
		"enabled":    ea.enabled,
	}
}

// buildEmailBody 构建邮件正文
func (ea *EmailAlert) buildEmailBody(message *AlertMessage) string {
	var body strings.Builder

	// 邮件头部
	body.WriteString(fmt.Sprintf("告警ID: %s\n", message.ID))
	body.WriteString(fmt.Sprintf("告警级别: %s\n", message.Level))
	body.WriteString(fmt.Sprintf("告警类型: %s\n", message.Type))
	body.WriteString(fmt.Sprintf("告警来源: %s\n", message.Source))
	body.WriteString(fmt.Sprintf("优先级: %d\n", message.Priority))
	body.WriteString(fmt.Sprintf("时间: %s\n", message.Timestamp.Format("2006-01-02 15:04:05")))
	body.WriteString("\n")

	// 告警内容
	body.WriteString("告警内容:\n")
	body.WriteString("=" + strings.Repeat("=", 50) + "\n")
	body.WriteString(message.Content)
	body.WriteString("\n")
	body.WriteString("=" + strings.Repeat("=", 50) + "\n\n")

	// 元数据
	if len(message.Metadata) > 0 {
		body.WriteString("详细信息:\n")
		body.WriteString("-" + strings.Repeat("-", 30) + "\n")
		for key, value := range message.Metadata {
			body.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
		body.WriteString("-" + strings.Repeat("-", 30) + "\n\n")
	}

	// 标签
	if len(message.Tags) > 0 {
		body.WriteString(fmt.Sprintf("标签: %s\n\n", strings.Join(message.Tags, ", ")))
	}

	// 邮件尾部
	body.WriteString("---\n")
	body.WriteString("此邮件由系统自动发送，请勿回复。\n")
	body.WriteString(fmt.Sprintf("发送时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return body.String()
}

// buildEmailMessage 构建邮件消息
func (ea *EmailAlert) buildEmailMessage(subject, body, from string, to []string) []byte {
	var message strings.Builder

	// 邮件头部
	message.WriteString(fmt.Sprintf("From: %s\n", from))
	message.WriteString(fmt.Sprintf("To: %s\n", strings.Join(to, ", ")))
	message.WriteString(fmt.Sprintf("Subject: %s\n", subject))
	message.WriteString("MIME-Version: 1.0\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\n")
	message.WriteString("\n")

	// 邮件正文
	message.WriteString(body)

	return []byte(message.String())
}

// sendEmail 发送邮件
func (ea *EmailAlert) sendEmail(message []byte) error {
	// 构建SMTP地址
	addr := fmt.Sprintf("%s:%s", ea.smtpHost, ea.smtpPort)

	// 创建认证
	auth := smtp.PlainAuth("", ea.username, ea.password, ea.smtpHost)

	// 发送邮件
	err := smtp.SendMail(addr, auth, ea.fromEmail, ea.toEmails, message)
	if err != nil {
		return fmt.Errorf("SMTP发送失败: %w", err)
	}

	return nil
}

// SendTestEmail 发送测试邮件
func (ea *EmailAlert) SendTestEmail(ctx context.Context) error {
	if !ea.enabled {
		return fmt.Errorf("邮件告警已禁用")
	}

	testMessage := &AlertMessage{
		ID:      "test_email",
		Title:   "测试邮件",
		Content: "这是一封测试邮件，用于验证邮件告警功能是否正常工作。",
		Level:   AlertLevelInfo,
		Type:    AlertTypeSystem,
		Source:  "email_test",
		Metadata: map[string]interface{}{
			"test": true,
		},
		Tags: []string{"test", "email"},
	}

	return ea.Send(ctx, testMessage)
}

// AddRecipient 添加收件人
func (ea *EmailAlert) AddRecipient(email string) {
	email = strings.TrimSpace(email)
	if email != "" {
		// 检查是否已存在
		for _, existingEmail := range ea.toEmails {
			if existingEmail == email {
				return
			}
		}
		ea.toEmails = append(ea.toEmails, email)
		ea.logger.Info(context.Background(), "添加邮件收件人", "email", email)
	}
}

// RemoveRecipient 移除收件人
func (ea *EmailAlert) RemoveRecipient(email string) {
	email = strings.TrimSpace(email)
	if email != "" {
		for i, existingEmail := range ea.toEmails {
			if existingEmail == email {
				ea.toEmails = append(ea.toEmails[:i], ea.toEmails[i+1:]...)
				ea.logger.Info(context.Background(), "移除邮件收件人", "email", email)
				return
			}
		}
	}
}

// GetRecipients 获取收件人列表
func (ea *EmailAlert) GetRecipients() []string {
	return ea.toEmails
}

// SetRecipients 设置收件人列表
func (ea *EmailAlert) SetRecipients(emails []string) {
	ea.toEmails = emails
	ea.logger.Info(context.Background(), "设置邮件收件人列表", "count", len(emails))
}
