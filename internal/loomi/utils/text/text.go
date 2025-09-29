package textx

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// TextUtils 文本处理工具类
type TextUtils struct {
	logger log.Logger
}

// NewTextUtils 创建文本处理工具
func NewTextUtils(logger log.Logger) *TextUtils {
	return &TextUtils{
		logger: logger,
	}
}

// ObserveThinkData 观察和思考数据结构
type ObserveThinkData struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Data    string `json:"data,omitempty"`
}

// RemoveFileAnalysisReferences 清理文件分析引用
func (tu *TextUtils) RemoveFileAnalysisReferences(observeThinkData []ObserveThinkData) []ObserveThinkData {
	tu.logger.Debug(context.Background(), "开始清理文件分析引用", "count", len(observeThinkData))

	if len(observeThinkData) == 0 {
		return observeThinkData
	}

	cleanedData := make([]ObserveThinkData, 0, len(observeThinkData))
	for _, item := range observeThinkData {
		cleanedItem := item
		cleanedItem.Content = tu.cleanText(item.Content)
		cleanedItem.Data = tu.cleanText(item.Data)
		cleanedData = append(cleanedData, cleanedItem)
	}

	tu.logger.Debug(context.Background(), "文件分析引用清理完成", "original_count", len(observeThinkData), "cleaned_count", len(cleanedData))
	return cleanedData
}

// cleanText 清理单个文本中的file_analysis相关内容
func (tu *TextUtils) cleanText(text string) string {
	if text == "" {
		return text
	}

	// 先处理带括号的引用格式：(@file_analysis数字) 和 （@file_analysis数字）
	// 支持英文括号和中文括号
	patternEn := regexp.MustCompile(`\(@file_analysis[^\)]*\)`) // 英文括号
	patternCn := regexp.MustCompile(`（@file_analysis[^）]*）`)    // 中文括号
	cleanedText := patternEn.ReplaceAllString(text, "")
	cleanedText = patternCn.ReplaceAllString(cleanedText, "")

	// 处理没有括号但以@开头的情况：@file_analysis数字
	pattern2 := regexp.MustCompile(`@file_analysis\w+`)
	cleanedText = pattern2.ReplaceAllString(cleanedText, "")

	// 处理file_analysis字符串本身
	pattern3 := regexp.MustCompile(`file_analysis`)
	cleanedText = pattern3.ReplaceAllString(cleanedText, "")

	// 清理多余的空白字符
	cleanedText = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedText, " ")
	cleanedText = strings.TrimSpace(cleanedText)

	return cleanedText
}

// ExtractFileAnalysisReferences 提取文件分析引用
func (tu *TextUtils) ExtractFileAnalysisReferences(text string) []string {
	tu.logger.Debug(context.Background(), "开始提取文件分析引用", "text_length", len(text))

	var references []string

	// 提取带括号的引用格式
	patternEn := regexp.MustCompile(`\(@file_analysis[^\)]*\)`)
	patternCn := regexp.MustCompile(`（@file_analysis[^）]*）`)

	matchesEn := patternEn.FindAllString(text, -1)
	matchesCn := patternCn.FindAllString(text, -1)

	references = append(references, matchesEn...)
	references = append(references, matchesCn...)

	// 提取没有括号的引用格式
	pattern2 := regexp.MustCompile(`@file_analysis\w+`)
	matches2 := pattern2.FindAllString(text, -1)
	references = append(references, matches2...)

	// 去重
	uniqueRefs := make(map[string]bool)
	var uniqueReferences []string
	for _, ref := range references {
		if !uniqueRefs[ref] {
			uniqueRefs[ref] = true
			uniqueReferences = append(uniqueReferences, ref)
		}
	}

	tu.logger.Debug(context.Background(), "文件分析引用提取完成", "count", len(uniqueReferences))
	return uniqueReferences
}

// FormatFileAnalysisSummary 格式化文件分析摘要
func (tu *TextUtils) FormatFileAnalysisSummary(references []string) string {
	tu.logger.Debug(context.Background(), "开始格式化文件分析摘要", "references_count", len(references))

	if len(references) == 0 {
		return "暂无文件分析引用"
	}

	var summary strings.Builder
	summary.WriteString("检测到以下文件分析引用：\n")

	for i, ref := range references {
		summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, ref))
	}

	result := summary.String()
	tu.logger.Debug(context.Background(), "文件分析摘要格式化完成", "summary_length", len(result))
	return result
}

// CleanObserveThinkData 清理观察和思考数据
func (tu *TextUtils) CleanObserveThinkData(data []ObserveThinkData) []ObserveThinkData {
	tu.logger.Debug(context.Background(), "开始清理观察和思考数据", "count", len(data))

	if len(data) == 0 {
		return data
	}

	cleanedData := make([]ObserveThinkData, 0, len(data))
	for _, item := range data {
		cleanedItem := item
		cleanedItem.Content = tu.cleanText(item.Content)
		cleanedItem.Data = tu.cleanText(item.Data)
		cleanedData = append(cleanedData, cleanedItem)
	}

	tu.logger.Debug(context.Background(), "观察和思考数据清理完成", "original_count", len(data), "cleaned_count", len(cleanedData))
	return cleanedData
}

// RemoveFileAnalysisReferences: 清理文件分析引用（兼容性函数）
func RemoveFileAnalysisReferences(s string) string {
	tu := NewTextUtils(nil)
	return tu.cleanText(s)
}

// CleanText 清理文本中的特殊字符和格式
func (tu *TextUtils) CleanText(text string) string {
	if text == "" {
		return text
	}

	// 移除HTML标签
	htmlPattern := regexp.MustCompile(`<[^>]*>`)
	cleanedText := htmlPattern.ReplaceAllString(text, "")

	// 移除多余的空白字符
	cleanedText = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedText, " ")
	cleanedText = strings.TrimSpace(cleanedText)

	// 移除特殊字符
	specialPattern := regexp.MustCompile(`[^\w\s\u4e00-\u9fff\u3000-\u303f\uff00-\uffef]`)
	cleanedText = specialPattern.ReplaceAllString(cleanedText, "")

	return cleanedText
}

// ExtractKeywords 提取关键词
func (tu *TextUtils) ExtractKeywords(text string, maxKeywords int) []string {
	if text == "" {
		return []string{}
	}

	// 简单的关键词提取逻辑
	words := regexp.MustCompile(`\b\w+\b`).FindAllString(text, -1)

	// 统计词频
	wordCount := make(map[string]int)
	for _, word := range words {
		if len(word) > 1 { // 过滤单字符
			wordCount[strings.ToLower(word)]++
		}
	}

	// 按频率排序
	type wordFreq struct {
		word  string
		count int
	}

	var wordFreqs []wordFreq
	for word, count := range wordCount {
		wordFreqs = append(wordFreqs, wordFreq{word, count})
	}

	// 简单的排序（按频率降序）
	for i := 0; i < len(wordFreqs)-1; i++ {
		for j := i + 1; j < len(wordFreqs); j++ {
			if wordFreqs[i].count < wordFreqs[j].count {
				wordFreqs[i], wordFreqs[j] = wordFreqs[j], wordFreqs[i]
			}
		}
	}

	// 提取前N个关键词
	var keywords []string
	for i, wf := range wordFreqs {
		if i >= maxKeywords {
			break
		}
		keywords = append(keywords, wf.word)
	}

	return keywords
}

// TruncateText 截断文本
func (tu *TextUtils) TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// 尝试在单词边界截断
	if maxLength > 0 && maxLength < len(text) {
		truncated := text[:maxLength]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > maxLength/2 {
			return truncated[:lastSpace] + "..."
		}
		return truncated + "..."
	}

	return text
}
