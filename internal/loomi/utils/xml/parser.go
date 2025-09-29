package xmlx

import (
	"regexp"
)

type ParseResult struct {
	ID        string
	Title     string
	Content   string
	CoverText string
	Hook      string
	Type      string
}

type Config struct {
	TagName string
}

// 简化占位：按 <tagN>...</tagN> 提取，后续补齐与 Python 一致的统一配置
func Parse(text string, cfg Config, startID int) []ParseResult {
	tag := regexp.MustCompile(`<` + cfg.TagName + `\d+>` + `([\s\S]*?)` + `</` + cfg.TagName + `\d+>`)
	matches := tag.FindAllStringSubmatch(text, -1)
	results := make([]ParseResult, 0, len(matches))
	for i, m := range matches {
		results = append(results, ParseResult{
			ID:      cfg.TagName + string(rune(startID+i)),
			Title:   "",
			Content: m[1],
			Type:    cfg.TagName,
		})
	}
	return results
}
