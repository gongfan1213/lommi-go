package utils

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
)

// LoomiReferenceResolver Loomi智能引用解析器
type LoomiReferenceResolver struct {
	logger         log.Logger
	contextManager *LoomiContextManager
}

// NewLoomiReferenceResolver 创建智能引用解析器
func NewLoomiReferenceResolver(logger log.Logger, contextManager *LoomiContextManager) *LoomiReferenceResolver {
	return &LoomiReferenceResolver{
		logger:         logger,
		contextManager: contextManager,
	}
}

// ResolveReference 解析自然语言引用
func (lrr *LoomiReferenceResolver) ResolveReference(ctx context.Context, userID, sessionID, reference string) ([]string, error) {
	lrr.logger.Info(ctx, "解析自然语言引用",
		"user_id", userID,
		"session_id", sessionID,
		"reference", reference)

	// 清理引用文本
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return []string{}, nil
	}

	// 检查是否已经是标准格式
	if lrr.isStandardFormat(reference) {
		return []string{reference}, nil
	}

	// 解析不同类型的引用
	var resolvedRefs []string

	// 1. 解析序号引用（如"第三个洞察"、"profile3"等）
	ordinalRefs, err := lrr.resolveOrdinalReferences(ctx, userID, sessionID, reference)
	if err != nil {
		lrr.logger.Error(ctx, "解析序号引用失败", "error", err)
	} else {
		resolvedRefs = append(resolvedRefs, ordinalRefs...)
	}

	// 2. 解析相对引用（如"最新的"、"上一个"等）
	relativeRefs, err := lrr.resolveRelativeReferences(ctx, userID, sessionID, reference)
	if err != nil {
		lrr.logger.Error(ctx, "解析相对引用失败", "error", err)
	} else {
		resolvedRefs = append(resolvedRefs, relativeRefs...)
	}

	// 3. 解析文件引用（如"@file1"、"@image1"等）
	fileRefs, err := lrr.resolveFileReferences(ctx, userID, sessionID, reference)
	if err != nil {
		lrr.logger.Error(ctx, "解析文件引用失败", "error", err)
	} else {
		resolvedRefs = append(resolvedRefs, fileRefs...)
	}

	// 4. 解析内容类型引用（如"洞察"、"画像"等）
	contentTypeRefs, err := lrr.resolveContentTypeReferences(ctx, userID, sessionID, reference)
	if err != nil {
		lrr.logger.Error(ctx, "解析内容类型引用失败", "error", err)
	} else {
		resolvedRefs = append(resolvedRefs, contentTypeRefs...)
	}

	// 去重
	resolvedRefs = lrr.removeDuplicates(resolvedRefs)

	lrr.logger.Info(ctx, "自然语言引用解析完成",
		"user_id", userID,
		"session_id", sessionID,
		"original_reference", reference,
		"resolved_count", len(resolvedRefs),
		"resolved_refs", resolvedRefs)

	return resolvedRefs, nil
}

// isStandardFormat 检查是否已经是标准格式
func (lrr *LoomiReferenceResolver) isStandardFormat(reference string) bool {
	// 检查是否是@开头的标准格式
	return strings.HasPrefix(reference, "@")
}

// resolveOrdinalReferences 解析序号引用
func (lrr *LoomiReferenceResolver) resolveOrdinalReferences(ctx context.Context, userID, sessionID, reference string) ([]string, error) {
	var resolvedRefs []string

	// 序号词汇映射
	ordinalMapping := map[string]int{
		"第一个": 1, "第二个": 2, "第三个": 3, "第四个": 4, "第五个": 5,
		"第一个": 1, "第二个": 2, "第三个": 3, "第四个": 4, "第五个": 5,
		"第一": 1, "第二": 2, "第三": 3, "第四": 4, "第五": 5,
		"1": 1, "2": 2, "3": 3, "4": 4, "5": 5,
	}

	// 内容类型映射
	contentTypes := map[string]string{
		"洞察": "insight", "画像": "profile", "打点": "hitpoint",
		"事实": "facts", "帖子": "xhs_post", "文案": "xhs_post",
		"思考": "fake_think", "抖音": "tiktok_script", "小红书": "xhs_post",
		"微信": "wechat_article", "文章": "wechat_article",
	}

	// 匹配模式：序号 + 内容类型
	patterns := []string{
		`(第一个|第二个|第三个|第四个|第五个|第一|第二|第三|第四|第五|\d+)(个)?(洞察|画像|打点|事实|帖子|文案|思考|抖音|小红书|微信|文章)`,
		`(洞察|画像|打点|事实|帖子|文案|思考|抖音|小红书|微信|文章)(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(reference, -1)

		for _, match := range matches {
			if len(match) >= 2 {
				var ordinal int
				var contentType string

				if len(match) == 4 {
					// 模式1：序号 + 内容类型
					ordinalStr := match[1]
					contentTypeStr := match[3]

					if num, exists := ordinalMapping[ordinalStr]; exists {
						ordinal = num
					} else if num, err := strconv.Atoi(ordinalStr); err == nil {
						ordinal = num
					}

					if ct, exists := contentTypes[contentTypeStr]; exists {
						contentType = ct
					}
				} else if len(match) == 3 {
					// 模式2：内容类型 + 序号
					contentTypeStr := match[1]
					ordinalStr := match[2]

					if ct, exists := contentTypes[contentTypeStr]; exists {
						contentType = ct
					}

					if num, err := strconv.Atoi(ordinalStr); err == nil {
						ordinal = num
					}
				}

				if ordinal > 0 && contentType != "" {
					// 构建标准引用格式
					standardRef := fmt.Sprintf("@%s%d", contentType, ordinal)
					resolvedRefs = append(resolvedRefs, standardRef)
				}
			}
		}
	}

	return resolvedRefs, nil
}

// resolveRelativeReferences 解析相对引用
func (lrr *LoomiReferenceResolver) resolveRelativeReferences(ctx context.Context, userID, sessionID, reference string) ([]string, error) {
	var resolvedRefs []string

	// 获取上下文状态
	contextState, err := lrr.contextManager.GetContext(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取上下文状态失败: %w", err)
	}

	// 相对引用模式
	relativePatterns := map[string]string{
		"最新的": "latest",
		"上一个": "previous",
		"最后的": "last",
		"最近":  "recent",
		"最新":  "latest",
	}

	for pattern, relativeType := range relativePatterns {
		if strings.Contains(reference, pattern) {
			// 根据相对类型获取对应的引用
			ref := lrr.getRelativeReference(contextState, relativeType)
			if ref != "" {
				resolvedRefs = append(resolvedRefs, ref)
			}
		}
	}

	return resolvedRefs, nil
}

// resolveFileReferences 解析文件引用
func (lrr *LoomiReferenceResolver) resolveFileReferences(ctx context.Context, userID, sessionID, reference string) ([]string, error) {
	var resolvedRefs []string

	// 文件引用模式
	filePatterns := []string{
		`@file(\d+)`,
		`@image(\d+)`,
		`@document(\d+)`,
		`@pdf(\d+)`,
		`@word(\d+)`,
		`@excel(\d+)`,
	}

	for _, pattern := range filePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(reference, -1)

		for _, match := range matches {
			if len(match) >= 2 {
				// 直接返回标准格式
				resolvedRefs = append(resolvedRefs, match[0])
			}
		}
	}

	return resolvedRefs, nil
}

// resolveContentTypeReferences 解析内容类型引用
func (lrr *LoomiReferenceResolver) resolveContentTypeReferences(ctx context.Context, userID, sessionID, reference string) ([]string, error) {
	var resolvedRefs []string

	// 内容类型映射
	contentTypes := map[string]string{
		"洞察": "insight", "画像": "profile", "打点": "hitpoint",
		"事实": "facts", "帖子": "xhs_post", "文案": "xhs_post",
		"思考": "fake_think", "抖音": "tiktok_script", "小红书": "xhs_post",
		"微信": "wechat_article", "文章": "wechat_article",
		"文件": "file", "图片": "image", "图像": "image",
		"照片": "image", "文档": "document", "PDF": "pdf",
		"Word": "word", "Excel": "excel",
	}

	// 检查是否包含内容类型关键词
	for keyword, contentType := range contentTypes {
		if strings.Contains(reference, keyword) {
			// 尝试获取该类型的最新引用
			ref := lrr.getLatestReferenceByType(ctx, userID, sessionID, contentType)
			if ref != "" {
				resolvedRefs = append(resolvedRefs, ref)
			}
		}
	}

	return resolvedRefs, nil
}

// getRelativeReference 获取相对引用
func (lrr *LoomiReferenceResolver) getRelativeReference(contextState *LoomiContextState, relativeType string) string {
	switch relativeType {
	case "latest", "last":
		// 返回最新的引用
		if len(contextState.CreatedNotes) > 0 {
			latestNote := contextState.CreatedNotes[len(contextState.CreatedNotes)-1]
			if noteID, ok := latestNote["id"].(string); ok {
				return "@" + noteID
			}
		}
	case "previous":
		// 返回上一个引用
		if len(contextState.CreatedNotes) > 1 {
			previousNote := contextState.CreatedNotes[len(contextState.CreatedNotes)-2]
			if noteID, ok := previousNote["id"].(string); ok {
				return "@" + noteID
			}
		}
	case "recent":
		// 返回最近的引用（最近3个）
		if len(contextState.CreatedNotes) > 0 {
			start := len(contextState.CreatedNotes) - 3
			if start < 0 {
				start = 0
			}
			recentNote := contextState.CreatedNotes[start]
			if noteID, ok := recentNote["id"].(string); ok {
				return "@" + noteID
			}
		}
	}
	return ""
}

// getLatestReferenceByType 根据类型获取最新引用
func (lrr *LoomiReferenceResolver) getLatestReferenceByType(ctx context.Context, userID, sessionID, contentType string) string {
	// 这里需要根据实际的存储实现来获取最新引用
	// 目前返回一个模拟的引用
	return fmt.Sprintf("@%s1", contentType)
}

// removeDuplicates 去除重复的引用
func (lrr *LoomiReferenceResolver) removeDuplicates(refs []string) []string {
	seen := make(map[string]bool)
	var uniqueRefs []string

	for _, ref := range refs {
		if !seen[ref] {
			seen[ref] = true
			uniqueRefs = append(uniqueRefs, ref)
		}
	}

	return uniqueRefs
}

// ValidateReference 验证引用有效性
func (lrr *LoomiReferenceResolver) ValidateReference(ctx context.Context, userID, sessionID, reference string) (bool, error) {
	lrr.logger.Info(ctx, "验证引用有效性",
		"user_id", userID,
		"session_id", sessionID,
		"reference", reference)

	// 解析引用
	resolvedRefs, err := lrr.ResolveReference(ctx, userID, sessionID, reference)
	if err != nil {
		return false, fmt.Errorf("解析引用失败: %w", err)
	}

	// 检查是否有有效解析结果
	if len(resolvedRefs) == 0 {
		return false, nil
	}

	// 获取上下文状态
	contextState, err := lrr.contextManager.GetContext(ctx, userID, sessionID)
	if err != nil {
		return false, fmt.Errorf("获取上下文状态失败: %w", err)
	}

	// 验证每个解析后的引用是否存在
	for _, resolvedRef := range resolvedRefs {
		if !lrr.referenceExists(contextState, resolvedRef) {
			lrr.logger.Warn(ctx, "引用不存在", "reference", resolvedRef)
			return false, nil
		}
	}

	lrr.logger.Info(ctx, "引用验证成功", "reference", reference, "resolved_refs", resolvedRefs)
	return true, nil
}

// referenceExists 检查引用是否存在
func (lrr *LoomiReferenceResolver) referenceExists(contextState *LoomiContextState, reference string) bool {
	// 检查是否在创建的notes中
	for _, note := range contextState.CreatedNotes {
		if noteID, ok := note["id"].(string); ok {
			if "@"+noteID == reference {
				return true
			}
		}
	}

	// 检查是否在全局上下文中
	if _, exists := contextState.GlobalContext[reference]; exists {
		return true
	}

	// 检查是否在共享内存中
	if _, exists := contextState.SharedMemory[reference]; exists {
		return true
	}

	return false
}

// ConvertToStandardFormat 转换为标准@引用格式
func (lrr *LoomiReferenceResolver) ConvertToStandardFormat(ctx context.Context, userID, sessionID, reference string) ([]string, error) {
	lrr.logger.Info(ctx, "转换为标准引用格式",
		"user_id", userID,
		"session_id", sessionID,
		"reference", reference)

	// 如果已经是标准格式，直接返回
	if lrr.isStandardFormat(reference) {
		return []string{reference}, nil
	}

	// 解析引用
	resolvedRefs, err := lrr.ResolveReference(ctx, userID, sessionID, reference)
	if err != nil {
		return nil, fmt.Errorf("解析引用失败: %w", err)
	}

	lrr.logger.Info(ctx, "标准格式转换完成",
		"original_reference", reference,
		"standard_refs", resolvedRefs)

	return resolvedRefs, nil
}

// GetReferenceSuggestions 获取引用建议
func (lrr *LoomiReferenceResolver) GetReferenceSuggestions(ctx context.Context, userID, sessionID, partialReference string) ([]string, error) {
	lrr.logger.Info(ctx, "获取引用建议",
		"user_id", userID,
		"session_id", sessionID,
		"partial_reference", partialReference)

	var suggestions []string

	// 获取上下文状态
	contextState, err := lrr.contextManager.GetContext(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取上下文状态失败: %w", err)
	}

	// 从创建的notes中获取建议
	for _, note := range contextState.CreatedNotes {
		if noteID, ok := note["id"].(string); ok {
			if strings.Contains(noteID, partialReference) {
				suggestions = append(suggestions, "@"+noteID)
			}
		}
	}

	// 从全局上下文中获取建议
	for key := range contextState.GlobalContext {
		if strings.Contains(key, partialReference) {
			suggestions = append(suggestions, "@"+key)
		}
	}

	// 从共享内存中获取建议
	for key := range contextState.SharedMemory {
		if strings.Contains(key, partialReference) {
			suggestions = append(suggestions, "@"+key)
		}
	}

	lrr.logger.Info(ctx, "引用建议获取完成",
		"partial_reference", partialReference,
		"suggestions_count", len(suggestions),
		"suggestions", suggestions)

	return suggestions, nil
}
