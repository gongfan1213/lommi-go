package markdown

import (
	"context"
	"log"
	"regexp"
	"strings"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// MarkdownProcessor Markdown格式处理器
type MarkdownProcessor struct {
	logger log.Logger
}

// NewMarkdownProcessor 创建Markdown处理器
func NewMarkdownProcessor(logger log.Logger) *MarkdownProcessor {
	return &MarkdownProcessor{
		logger: logger,
	}
}

// EnsureCompatibility 确保markdown格式在JSON传输和前端渲染中的兼容性
func (mp *MarkdownProcessor) EnsureCompatibility(text string) string {
	if text == "" {
		return ""
	}

	mp.logger.Debug(context.Background(), "开始处理markdown兼容性", "text_length", len(text))

	// 阶段1：预处理 - 修复JSON传输中被转义的markdown格式
	text = mp.fixEscapedMarkdown(text)

	// 阶段2：规范化 - 修复不匹配和异常的星号标记
	text = mp.normalizeMarkdownMarks(text)

	// 阶段3：优化 - 清理内部空格并优化格式
	text = mp.optimizeMarkdownFormat(text)

	// 阶段4：验证 - 确保输出的一致性
	text = mp.validateMarkdownSyntax(text)

	mp.logger.Debug(context.Background(), "markdown兼容性处理完成", "result_length", len(text))
	return text
}

// fixEscapedMarkdown 修复JSON传输中被转义的markdown格式
func (mp *MarkdownProcessor) fixEscapedMarkdown(text string) string {
	// 修复被转义的星号
	text = strings.ReplaceAll(text, "\\*", "*")
	text = strings.ReplaceAll(text, "\\_", "_")
	text = strings.ReplaceAll(text, "\\`", "`")
	text = strings.ReplaceAll(text, "\\#", "#")
	text = strings.ReplaceAll(text, "\\[", "[")
	text = strings.ReplaceAll(text, "\\]", "]")
	text = strings.ReplaceAll(text, "\\(", "(")
	text = strings.ReplaceAll(text, "\\)", ")")

	return text
}

// normalizeMarkdownMarks 规范化markdown标记
func (mp *MarkdownProcessor) normalizeMarkdownMarks(text string) string {
	// 修复不匹配的星号标记
	// 处理奇数个星号的情况
	starPattern := regexp.MustCompile(`\*+`)
	text = starPattern.ReplaceAllStringFunc(text, func(match string) string {
		count := len(match)
		if count%2 == 1 {
			// 奇数个星号，去掉最后一个
			return strings.Repeat("*", count-1)
		}
		return match
	})

	// 修复不匹配的下划线标记
	underscorePattern := regexp.MustCompile(`_+`)
	text = underscorePattern.ReplaceAllStringFunc(text, func(match string) string {
		count := len(match)
		if count%2 == 1 {
			// 奇数个下划线，去掉最后一个
			return strings.Repeat("_", count-1)
		}
		return match
	})

	// 修复不匹配的反引号标记
	backtickPattern := regexp.MustCompile("`+")
	text = backtickPattern.ReplaceAllStringFunc(text, func(match string) string {
		count := len(match)
		if count%2 == 1 {
			// 奇数个反引号，去掉最后一个
			return strings.Repeat("`", count-1)
		}
		return match
	})

	return text
}

// optimizeMarkdownFormat 优化markdown格式
func (mp *MarkdownProcessor) optimizeMarkdownFormat(text string) string {
	// 规范化Windows换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 清理每行末尾的空格
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	text = strings.Join(lines, "\n")

	// 压缩连续的空白行（最多保留2个）
	text = regexp.MustCompile("\n{3,}").ReplaceAllString(text, "\n\n")

	// 清理行首的额外空格
	text = regexp.MustCompile(`\n\s+\n`).ReplaceAllString(text, "\n\n")

	return text
}

// validateMarkdownSyntax 验证markdown语法
func (mp *MarkdownProcessor) validateMarkdownSyntax(text string) string {
	// 确保标题格式正确
	text = regexp.MustCompile(`^#{1,6}\s+`).ReplaceAllStringFunc(text, func(match string) string {
		return strings.TrimSpace(match) + " "
	})

	// 确保列表格式正确
	text = regexp.MustCompile(`^\s*[-*+]\s+`).ReplaceAllStringFunc(text, func(match string) string {
		return strings.TrimSpace(match) + " "
	})

	// 确保代码块格式正确
	text = regexp.MustCompile("```[^`]*```").ReplaceAllStringFunc(text, func(match string) string {
		// 确保代码块有正确的换行
		if !strings.HasPrefix(match, "```\n") {
			match = "```\n" + strings.TrimPrefix(match, "```")
		}
		if !strings.HasSuffix(match, "\n```") {
			match = strings.TrimSuffix(match, "```") + "\n```"
		}
		return match
	})

	return text
}

// LogAnalysis 记录markdown分析日志
func (mp *MarkdownProcessor) LogAnalysis(content, label string) {
	if len(content) > 0 {
		preview := content
		if len(preview) > 160 {
			preview = preview[:160] + "..."
		}
		mp.logger.Debug(context.Background(), "markdown分析", "label", label, "length", len(content), "preview", preview)
	} else {
		mp.logger.Debug(context.Background(), "markdown分析", "label", label, "status", "empty")
	}
}

// ExtractMarkdownElements 提取markdown元素
func (mp *MarkdownProcessor) ExtractMarkdownElements(text string) map[string][]string {
	elements := make(map[string][]string)

	// 提取标题
	headers := regexp.MustCompile(`^#{1,6}\s+(.+)$`).FindAllStringSubmatch(text, -1)
	for _, header := range headers {
		if len(header) > 1 {
			elements["headers"] = append(elements["headers"], header[1])
		}
	}

	// 提取列表项
	listItems := regexp.MustCompile(`^\s*[-*+]\s+(.+)$`).FindAllStringSubmatch(text, -1)
	for _, item := range listItems {
		if len(item) > 1 {
			elements["list_items"] = append(elements["list_items"], item[1])
		}
	}

	// 提取代码块
	codeBlocks := regexp.MustCompile("```[^`]*```").FindAllString(text, -1)
	elements["code_blocks"] = codeBlocks

	// 提取内联代码
	inlineCode := regexp.MustCompile("`[^`]+`").FindAllString(text, -1)
	elements["inline_code"] = inlineCode

	// 提取链接
	links := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).FindAllStringSubmatch(text, -1)
	for _, link := range links {
		if len(link) > 2 {
			elements["links"] = append(elements["links"], link[1]+" -> "+link[2])
		}
	}

	return elements
}

// ConvertToHTML 将markdown转换为HTML（简化版）
func (mp *MarkdownProcessor) ConvertToHTML(text string) string {
	// 这是一个简化的转换，实际项目中应该使用专门的markdown库
	html := text

	// 转换标题
	html = regexp.MustCompile(`^#{6}\s+(.+)$`).ReplaceAllString(html, "<h6>$1</h6>")
	html = regexp.MustCompile(`^#{5}\s+(.+)$`).ReplaceAllString(html, "<h5>$1</h5>")
	html = regexp.MustCompile(`^#{4}\s+(.+)$`).ReplaceAllString(html, "<h4>$1</h4>")
	html = regexp.MustCompile(`^#{3}\s+(.+)$`).ReplaceAllString(html, "<h3>$1</h3>")
	html = regexp.MustCompile(`^#{2}\s+(.+)$`).ReplaceAllString(html, "<h2>$1</h2>")
	html = regexp.MustCompile(`^#{1}\s+(.+)$`).ReplaceAllString(html, "<h1>$1</h1>")

	// 转换粗体
	html = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(html, "<strong>$1</strong>")

	// 转换斜体
	html = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(html, "<em>$1</em>")

	// 转换代码块
	html = regexp.MustCompile("```([^`]*)```").ReplaceAllString(html, "<pre><code>$1</code></pre>")

	// 转换内联代码
	html = regexp.MustCompile("`([^`]+)`").ReplaceAllString(html, "<code>$1</code>")

	// 转换链接
	html = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(html, "<a href=\"$2\">$1</a>")

	return html
}

// EnsureCompatibility 兼容性函数（保持向后兼容）
func EnsureCompatibility(text string) string {
	mp := NewMarkdownProcessor(nil)
	return mp.EnsureCompatibility(text)
}

// LogAnalysis 兼容性函数（保持向后兼容）
func LogAnalysis(content, label string) {
	if len(content) > 0 {
		preview := content
		if len(preview) > 160 {
			preview = preview[:160] + "..."
		}
		log.Printf("[markdown][%s] len=%d preview=%q", label, len(content), preview)
	} else {
		log.Printf("[markdown][%s] empty", label)
	}
}
