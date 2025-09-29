package utils

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// StreamingTagParser 流式XML标签解析器
type StreamingTagParser struct {
	buffer                string
	processedTags         map[string]bool
	lastProcessedPosition int
	mu                    sync.RWMutex
	logger                log.Logger
}

// TagInfo 标签信息
type TagInfo struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// NewStreamingTagParser 创建流式XML标签解析器
func NewStreamingTagParser(logger log.Logger) *StreamingTagParser {
	return &StreamingTagParser{
		buffer:        "",
		processedTags: make(map[string]bool),
		logger:        logger,
	}
}

// Reset 重置解析器状态，用于新的流式解析会话
func (stp *StreamingTagParser) Reset() {
	stp.mu.Lock()
	defer stp.mu.Unlock()

	stp.buffer = ""
	stp.processedTags = make(map[string]bool)
	stp.lastProcessedPosition = 0

	stp.logger.Debug(context.Background(), "StreamingTagParser已重置")
}

// AddChunk 添加新的chunk并检查是否有完整的标签
func (stp *StreamingTagParser) AddChunk(chunk string) []TagInfo {
	stp.mu.Lock()
	defer stp.mu.Unlock()

	if chunk == "" {
		return nil
	}

	// 添加到缓冲区
	stp.buffer += chunk

	// 检查完整标签
	newTags := stp.extractCompleteTags()

	stp.logger.Debug(context.Background(), "添加chunk到解析器",
		"chunk_length", len(chunk),
		"buffer_length", len(stp.buffer),
		"new_tags_count", len(newTags))

	return newTags
}

// extractCompleteTags 提取完整的标签
func (stp *StreamingTagParser) extractCompleteTags() []TagInfo {
	var tags []TagInfo

	// 支持的标签类型
	tagTypes := []string{"think", "Observe", "observe"}

	for _, tagType := range tagTypes {
		// 构建正则表达式模式
		pattern := regexp.MustCompile(`<` + tagType + `[^>]*>([^<]*)</` + tagType + `>`)
		matches := pattern.FindAllStringSubmatch(stp.buffer, -1)

		for _, match := range matches {
			if len(match) > 1 {
				content := strings.TrimSpace(match[1])
				if content != "" {
					// 生成标签的唯一标识符
					tagID := tagType + ":" + content

					// 检查是否已经处理过这个标签
					if !stp.processedTags[tagID] {
						stp.processedTags[tagID] = true
						tags = append(tags, TagInfo{
							Type:    tagType,
							Content: content,
						})
					}
				}
			}
		}
	}

	return tags
}

// GetBuffer 获取当前缓冲区内容
func (stp *StreamingTagParser) GetBuffer() string {
	stp.mu.RLock()
	defer stp.mu.RUnlock()
	return stp.buffer
}

// GetProcessedTags 获取已处理的标签
func (stp *StreamingTagParser) GetProcessedTags() map[string]bool {
	stp.mu.RLock()
	defer stp.mu.RUnlock()

	// 返回副本
	result := make(map[string]bool)
	for k, v := range stp.processedTags {
		result[k] = v
	}
	return result
}

// GetLastProcessedPosition 获取最后处理的位置
func (stp *StreamingTagParser) GetLastProcessedPosition() int {
	stp.mu.RLock()
	defer stp.mu.RUnlock()
	return stp.lastProcessedPosition
}

// SetLastProcessedPosition 设置最后处理的位置
func (stp *StreamingTagParser) SetLastProcessedPosition(position int) {
	stp.mu.Lock()
	defer stp.mu.Unlock()
	stp.lastProcessedPosition = position
}

// ExtractAllTags 提取缓冲区中的所有标签（不重复）
func (stp *StreamingTagParser) ExtractAllTags() []TagInfo {
	stp.mu.Lock()
	defer stp.mu.Unlock()

	var allTags []TagInfo
	tagTypes := []string{"think", "Observe", "observe"}

	for _, tagType := range tagTypes {
		pattern := regexp.MustCompile(`<` + tagType + `[^>]*>([^<]*)</` + tagType + `>`)
		matches := pattern.FindAllStringSubmatch(stp.buffer, -1)

		for _, match := range matches {
			if len(match) > 1 {
				content := strings.TrimSpace(match[1])
				if content != "" {
					tagID := tagType + ":" + content
					if !stp.processedTags[tagID] {
						stp.processedTags[tagID] = true
						allTags = append(allTags, TagInfo{
							Type:    tagType,
							Content: content,
						})
					}
				}
			}
		}
	}

	return allTags
}

// ClearProcessedTags 清理已处理的标签记录
func (stp *StreamingTagParser) ClearProcessedTags() {
	stp.mu.Lock()
	defer stp.mu.Unlock()
	stp.processedTags = make(map[string]bool)
	stp.logger.Debug(context.Background(), "已清理处理过的标签记录")
}

// GetUnprocessedContent 获取未处理的内容
func (stp *StreamingTagParser) GetUnprocessedContent() string {
	stp.mu.RLock()
	defer stp.mu.RUnlock()

	// 移除所有标签，只保留纯文本内容
	content := stp.buffer

	// 移除think标签
	content = regexp.MustCompile(`<think[^>]*>.*?</think>`).ReplaceAllString(content, "")

	// 移除Observe标签
	content = regexp.MustCompile(`<Observe[^>]*>.*?</Observe>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<observe[^>]*>.*?</observe>`).ReplaceAllString(content, "")

	// 清理多余的空白字符
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	return content
}

// GetTagStatistics 获取标签统计信息
func (stp *StreamingTagParser) GetTagStatistics() map[string]int {
	stp.mu.RLock()
	defer stp.mu.RUnlock()

	stats := make(map[string]int)
	tagTypes := []string{"think", "Observe", "observe"}

	for _, tagType := range tagTypes {
		pattern := regexp.MustCompile(`<` + tagType + `[^>]*>([^<]*)</` + tagType + `>`)
		matches := pattern.FindAllString(stp.buffer, -1)
		stats[tagType] = len(matches)
	}

	return stats
}

// IsTagComplete 检查标签是否完整
func (stp *StreamingTagParser) IsTagComplete(tagType string) bool {
	stp.mu.RLock()
	defer stp.mu.RUnlock()

	// 检查是否有完整的开始和结束标签
	startPattern := regexp.MustCompile(`<` + tagType + `[^>]*>`)
	endPattern := regexp.MustCompile(`</` + tagType + `>`)

	startMatches := startPattern.FindAllString(stp.buffer, -1)
	endMatches := endPattern.FindAllString(stp.buffer, -1)

	return len(startMatches) > 0 && len(endMatches) > 0 && len(startMatches) == len(endMatches)
}

// GetIncompleteTags 获取不完整的标签
func (stp *StreamingTagParser) GetIncompleteTags() []string {
	stp.mu.RLock()
	defer stp.mu.RUnlock()

	var incompleteTags []string
	tagTypes := []string{"think", "Observe", "observe"}

	for _, tagType := range tagTypes {
		startPattern := regexp.MustCompile(`<` + tagType + `[^>]*>`)
		endPattern := regexp.MustCompile(`</` + tagType + `>`)

		startMatches := startPattern.FindAllString(stp.buffer, -1)
		endMatches := endPattern.FindAllString(stp.buffer, -1)

		if len(startMatches) > len(endMatches) {
			incompleteTags = append(incompleteTags, tagType)
		}
	}

	return incompleteTags
}

// CleanupBuffer 清理缓冲区中的已处理内容
func (stp *StreamingTagParser) CleanupBuffer() {
	stp.mu.Lock()
	defer stp.mu.Unlock()

	// 保留未处理的内容
	unprocessedContent := stp.GetUnprocessedContent()
	stp.buffer = unprocessedContent

	stp.logger.Debug(context.Background(), "缓冲区已清理", "remaining_length", len(stp.buffer))
}

// GetBufferLength 获取缓冲区长度
func (stp *StreamingTagParser) GetBufferLength() int {
	stp.mu.RLock()
	defer stp.mu.RUnlock()
	return len(stp.buffer)
}

// HasUnprocessedContent 检查是否有未处理的内容
func (stp *StreamingTagParser) HasUnprocessedContent() bool {
	stp.mu.RLock()
	defer stp.mu.RUnlock()

	unprocessedContent := stp.GetUnprocessedContent()
	return len(unprocessedContent) > 0
}
