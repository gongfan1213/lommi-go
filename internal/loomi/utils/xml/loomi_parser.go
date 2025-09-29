package xmlx

import (
	"fmt"
	"regexp"
	"strings"
)

// LoomiXMLParser provides enhanced XML parsing capabilities for Loomi agents
type LoomiXMLParser struct{}

// NewLoomiXMLParser creates a new Loomi XML parser
func NewLoomiXMLParser() *LoomiXMLParser {
	return &LoomiXMLParser{}
}

// ParseConfig represents the configuration for parsing
type ParseConfig struct {
	TagName      string
	TitleTag     string
	ContentTag   string
	CoverTextTag string
	HookTag      string
	Type         string
}

// UnifiedConfigs provides predefined configurations for different agent types
var UnifiedConfigs = map[string]ParseConfig{
	"resonant": {
		TagName:    "resonant",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "resonant",
	},
	"persona": {
		TagName:    "persona",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "persona",
	},
	"hitpoint": {
		TagName:    "hitpoint",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "hitpoint",
	},
	"brand_analysis": {
		TagName:    "brand_analysis",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "brand_analysis",
	},
	"content_analysis": {
		TagName:    "content_analysis",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "content_analysis",
	},
	"knowledge": {
		TagName:    "knowledge",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "knowledge",
	},
	"websearch": {
		TagName:    "websearch",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "websearch",
	},
	"orchestrator": {
		TagName:    "orchestrator",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "orchestrator",
	},
}

// ContentConfigs provides configurations for content creation agents
var ContentConfigs = map[string]ParseConfig{
	"xhs_post": {
		TagName:      "xhs_post",
		TitleTag:     "title",
		ContentTag:   "content",
		CoverTextTag: "cover_text",
		Type:         "xhs_post",
	},
	"wechat_article": {
		TagName:    "wechat_article",
		TitleTag:   "title",
		ContentTag: "content",
		Type:       "wechat_article",
	},
	"tiktok_script": {
		TagName:      "tiktok_script",
		TitleTag:     "title",
		ContentTag:   "content",
		CoverTextTag: "cover_text",
		HookTag:      "hook",
		Type:         "tiktok_script",
	},
}

// ParseEnhanced provides enhanced XML parsing with title extraction
func (p *LoomiXMLParser) ParseEnhanced(text string, config ParseConfig, startID int) []ParseResult {
	// Pattern for extracting complete XML blocks
	pattern := fmt.Sprintf(`<%s\d+>([\s\S]*?)</%s\d+>`, config.TagName, config.TagName)
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)

	results := make([]ParseResult, 0, len(matches))

	for i, match := range matches {
		if len(match) < 2 {
			continue
		}

		blockContent := match[1]

		// Extract title if available
		title := p.extractTitle(blockContent, config)

		// Extract content (remove title if present)
		content := p.extractContent(blockContent, config, title)

		// Extract optional cover_text and hook if configured
		cover := ""
		if config.CoverTextTag != "" {
			cover = p.extractTag(blockContent, config.CoverTextTag)
		}
		hook := ""
		if config.HookTag != "" {
			hook = p.extractTag(blockContent, config.HookTag)
		}

		results = append(results, ParseResult{
			ID:        fmt.Sprintf("%s%d", config.TagName, startID+i),
			Title:     title,
			Content:   content,
			CoverText: strings.TrimSpace(cover),
			Hook:      strings.TrimSpace(hook),
			Type:      config.Type,
		})
	}

	return results
}

// extractTitle extracts title from XML block
func (p *LoomiXMLParser) extractTitle(blockContent string, config ParseConfig) string {
	// Try to extract title from <title> tag
	titlePattern := fmt.Sprintf(`<%s>([\s\S]*?)</%s>`, config.TitleTag, config.TitleTag)
	titleRe := regexp.MustCompile(titlePattern)
	titleMatch := titleRe.FindStringSubmatch(blockContent)

	if len(titleMatch) > 1 {
		return strings.TrimSpace(titleMatch[1])
	}

	// Try to extract from the first line if it looks like a title
	lines := strings.Split(strings.TrimSpace(blockContent), "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		// Check if first line looks like a title (not too long, no special formatting)
		if len(firstLine) <= 100 && !strings.Contains(firstLine, "：") && !strings.Contains(firstLine, ":") {
			return firstLine
		}
	}

	return ""
}

// extractContent extracts content from XML block
func (p *LoomiXMLParser) extractContent(blockContent string, config ParseConfig, title string) string {
	// Remove title tag if present
	content := blockContent
	titlePattern := fmt.Sprintf(`<%s>[\s\S]*?</%s>`, config.TitleTag, config.TitleTag)
	content = regexp.MustCompile(titlePattern).ReplaceAllString(content, "")

	// If we found a title in the first line, remove it
	if title != "" {
		lines := strings.Split(content, "\n")
		if len(lines) > 0 && strings.TrimSpace(lines[0]) == title {
			content = strings.Join(lines[1:], "\n")
		}
	}

	// Clean up the content
	content = strings.TrimSpace(content)

	// Remove extra blank lines
	content = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(content, "\n\n")

	return content
}

// extractTag extracts the inner text of a simple tag like <tag>...</tag>
func (p *LoomiXMLParser) extractTag(blockContent string, tag string) string {
	if tag == "" {
		return ""
	}
	pattern := fmt.Sprintf(`<%s>([\s\S]*?)</%s>`, tag, tag)
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(blockContent)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// CleanXMLTags removes XML tags from text, leaving only the thinking process
func (p *LoomiXMLParser) CleanXMLTags(text, tagName string) string {
	// Remove complete XML blocks
	pattern := fmt.Sprintf(`<%s\d+>[\s\S]*?</%s\d+>`, tagName, tagName)
	cleaned := regexp.MustCompile(pattern).ReplaceAllString(text, "")

	// Remove partial tags
	cleaned = regexp.MustCompile(fmt.Sprintf(`</?%s\d*>`, tagName)).ReplaceAllString(cleaned, "")

	// Clean up extra whitespace
	cleaned = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(cleaned, "\n\n")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// ExtractOtherContent extracts content outside of specified XML tags
func (p *LoomiXMLParser) ExtractOtherContent(text string, tagPatterns []string) string {
	cleaned := text

	for _, pattern := range tagPatterns {
		// Remove complete tag blocks
		fullPattern := fmt.Sprintf(`%s[\s\S]*?%s`, pattern, strings.Replace(pattern, "<", "</", 1))
		re := regexp.MustCompile(fullPattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Clean up extra whitespace
	cleaned = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(cleaned, "\n\n")
	cleaned = strings.TrimSpace(cleaned)

	// Return if there's substantial content
	if len(cleaned) > 10 {
		return cleaned
	}

	return ""
}
