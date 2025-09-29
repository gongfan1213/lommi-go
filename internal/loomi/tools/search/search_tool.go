package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// SearchTool 搜索工具统一接口
type SearchTool struct {
	logger            *logx.Logger
	jinaClient        *JinaClient
	socialMediaSearch *SocialMediaSearch
	zhipuClient       *ZhipuClient
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query      string   `json:"query"`
	SearchType string   `json:"search_type"` // "web", "social", "all", "zhipu"
	Platforms  []string `json:"platforms"`   // 社交媒体平台列表
	MaxResults int      `json:"max_results"`
	Timeout    int      `json:"timeout"`
	Language   string   `json:"language"`
	DateRange  string   `json:"date_range"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Query        string                 `json:"query"`
	SearchType   string                 `json:"search_type"`
	TotalResults int                    `json:"total_results"`
	Results      []SearchResult         `json:"results"`
	Platforms    []string               `json:"platforms"`
	SearchTime   time.Duration          `json:"search_time"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// UnifiedSearchResult 统一搜索结果
type UnifiedSearchResult struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	Platform    string    `json:"platform"`
	Source      string    `json:"source"` // "web", "douyin", "xiaohongshu", "zhipu"
	Likes       int       `json:"likes"`
	Comments    int       `json:"comments"`
	Shares      int       `json:"shares"`
	Views       int       `json:"views"`
	Tags        []string  `json:"tags"`
	ImageURL    string    `json:"image_url"`
	VideoURL    string    `json:"video_url"`
	Score       float64   `json:"score"`
	Relevance   float64   `json:"relevance"`
}

// NewSearchTool 创建搜索工具
func NewSearchTool(logger *logx.Logger) *SearchTool {
	return &SearchTool{
		logger:            logger,
		jinaClient:        NewJinaClient(logger),
		socialMediaSearch: NewSocialMediaSearch(logger),
		zhipuClient:       NewZhipuClient(logger),
	}
}

// SetAPIKeys 设置API密钥
func (st *SearchTool) SetAPIKeys(keys map[string]string) {
	if jinaKey, ok := keys["jina"]; ok {
		st.jinaClient.SetAPIKey(jinaKey)
	}

	if douyinKey, ok := keys["douyin"]; ok {
		st.socialMediaSearch.SetAPIKey("douyin", douyinKey)
	}

	if xhsKey, ok := keys["xiaohongshu"]; ok {
		st.socialMediaSearch.SetAPIKey("xiaohongshu", xhsKey)
	}

	if zhipuKey, ok := keys["zhipu"]; ok {
		st.zhipuClient.SetAPIKey(zhipuKey)
	}
}

// Search 执行搜索
func (st *SearchTool) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	startTime := time.Now()
	st.logger.Info(ctx, "开始执行搜索",
		logx.KV("query", req.Query),
		logx.KV("search_type", req.SearchType),
		logx.KV("platforms", req.Platforms))

	response := &SearchResponse{
		Query:      req.Query,
		SearchType: req.SearchType,
		Results:    []SearchResult{},
		Platforms:  req.Platforms,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// 设置默认值
	if req.MaxResults <= 0 {
		req.MaxResults = 20
	}
	if req.Timeout <= 0 {
		req.Timeout = 30
	}

	// 根据搜索类型执行不同的搜索
	switch strings.ToLower(req.SearchType) {
	case "web":
		results, err := st.searchWeb(ctx, req)
		if err != nil {
			st.logger.Error(ctx, "网页搜索失败", logx.KV("error", err))
		} else {
			response.Results = append(response.Results, results...)
		}

	case "social":
		results, err := st.searchSocialMedia(ctx, req)
		if err != nil {
			st.logger.Error(ctx, "社交媒体搜索失败", logx.KV("error", err))
		} else {
			response.Results = append(response.Results, results...)
		}

	case "zhipu":
		results, err := st.searchWithZhipu(ctx, req)
		if err != nil {
			st.logger.Error(ctx, "智谱搜索失败", logx.KV("error", err))
		} else {
			response.Results = append(response.Results, results...)
		}

	case "all":
		// 并行执行所有类型的搜索
		webResults, webErr := st.searchWeb(ctx, req)
		socialResults, socialErr := st.searchSocialMedia(ctx, req)
		zhipuResults, zhipuErr := st.searchWithZhipu(ctx, req)

		if webErr != nil {
			st.logger.Error(ctx, "网页搜索失败", logx.KV("error", webErr))
		} else {
			response.Results = append(response.Results, webResults...)
		}

		if socialErr != nil {
			st.logger.Error(ctx, "社交媒体搜索失败", logx.KV("error", socialErr))
		} else {
			response.Results = append(response.Results, socialResults...)
		}

		if zhipuErr != nil {
			st.logger.Error(ctx, "智谱搜索失败", logx.KV("error", zhipuErr))
		} else {
			response.Results = append(response.Results, zhipuResults...)
		}

	default:
		return nil, fmt.Errorf("不支持的搜索类型: %s", req.SearchType)
	}

	// 限制结果数量
	if len(response.Results) > req.MaxResults {
		response.Results = response.Results[:req.MaxResults]
	}

	response.TotalResults = len(response.Results)
	response.SearchTime = time.Since(startTime)
	response.Metadata["search_duration_ms"] = response.SearchTime.Milliseconds()

	st.logger.Info(ctx, "搜索完成",
		logx.KV("total_results", response.TotalResults),
		logx.KV("search_time", response.SearchTime))

	return response, nil
}

// searchWeb 网页搜索
func (st *SearchTool) searchWeb(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	st.logger.Info(ctx, "执行网页搜索", logx.KV("query", req.Query))

	// 使用Jina AI进行网页搜索
	results, err := st.jinaClient.SearchWeb(ctx, req.Query)
	if err != nil {
		return nil, err
	}

	// 转换为统一格式
	var searchResults []SearchResult
	for _, result := range results {
		searchResults = append(searchResults, SearchResult{
			Title:       result.Title,
			Content:     result.Content,
			URL:         result.URL,
			Author:      result.Author,
			PublishedAt: result.PublishedAt,
			Platform:    "web",
			Tags:        []string{"网页搜索"},
		})
	}

	return searchResults, nil
}

// searchSocialMedia 社交媒体搜索
func (st *SearchTool) searchSocialMedia(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	st.logger.Info(ctx, "执行社交媒体搜索",
		logx.KV("query", req.Query),
		logx.KV("platforms", req.Platforms))

	var allResults []SearchResult

	// 如果没有指定平台，搜索所有平台
	platforms := req.Platforms
	if len(platforms) == 0 {
		platforms = []string{"douyin", "xiaohongshu"}
	}

	for _, platform := range platforms {
		results, err := st.socialMediaSearch.Search(ctx, req.Query, platform)
		if err != nil {
			st.logger.Error(ctx, "平台搜索失败",
				logx.KV("platform", platform),
				logx.KV("error", err))
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// searchWithZhipu 智谱搜索
func (st *SearchTool) searchWithZhipu(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	st.logger.Info(ctx, "执行智谱搜索", logx.KV("query", req.Query))

	// 使用智谱AI进行搜索
	results, err := st.zhipuClient.Search(ctx, req.Query)
	if err != nil {
		return nil, err
	}

	// 转换为统一格式
	var searchResults []SearchResult
	for _, result := range results {
		searchResults = append(searchResults, SearchResult{
			Title:       result.Title,
			Content:     result.Content,
			URL:         result.URL,
			Author:      result.Author,
			PublishedAt: result.PublishedAt,
			Platform:    "zhipu",
			Tags:        []string{"智谱AI"},
		})
	}

	return searchResults, nil
}

// SearchAndProcessPosts 搜索并处理帖子
func (st *SearchTool) SearchAndProcessPosts(ctx context.Context, query, platform string) ([]ProcessedPost, error) {
	st.logger.Info(ctx, "搜索并处理帖子",
		logx.KV("query", query),
		logx.KV("platform", platform))

	// 搜索帖子
	searchResults, err := st.socialMediaSearch.Search(ctx, query, platform)
	if err != nil {
		return nil, err
	}

	// 处理搜索结果
	var processedPosts []ProcessedPost
	for _, result := range searchResults {
		post := ProcessedPost{
			ID:          generatePostID(result.URL),
			Title:       result.Title,
			Content:     result.Content,
			URL:         result.URL,
			Author:      result.Author,
			Platform:    result.Platform,
			PublishedAt: result.PublishedAt,
			Engagement: PostEngagement{
				Likes:    result.Likes,
				Comments: result.Comments,
				Shares:   result.Shares,
				Views:    result.Views,
			},
			Media: PostMedia{
				ImageURL: result.ImageURL,
				VideoURL: result.VideoURL,
			},
			Tags:        result.Tags,
			ProcessedAt: time.Now(),
		}

		processedPosts = append(processedPosts, post)
	}

	st.logger.Info(ctx, "帖子处理完成", logx.KV("processed_count", len(processedPosts)))
	return processedPosts, nil
}

// ProcessedPost 处理后的帖子
type ProcessedPost struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	URL         string         `json:"url"`
	Author      string         `json:"author"`
	Platform    string         `json:"platform"`
	PublishedAt time.Time      `json:"published_at"`
	Engagement  PostEngagement `json:"engagement"`
	Media       PostMedia      `json:"media"`
	Tags        []string       `json:"tags"`
	ProcessedAt time.Time      `json:"processed_at"`
}

// PostEngagement 帖子互动数据
type PostEngagement struct {
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
	Views    int `json:"views"`
}

// PostMedia 帖子媒体
type PostMedia struct {
	ImageURL string `json:"image_url"`
	VideoURL string `json:"video_url"`
}

// generatePostID 生成帖子ID
func generatePostID(url string) string {
	// 简单的ID生成逻辑，实际应用中应该更复杂
	return strings.ReplaceAll(url, "/", "_")
}

// GetSearchStats 获取搜索统计
func (st *SearchTool) GetSearchStats(ctx context.Context) map[string]interface{} {
	stats := map[string]interface{}{
		"jina_client":         st.jinaClient != nil,
		"social_media":        st.socialMediaSearch != nil,
		"zhipu_client":        st.zhipuClient != nil,
		"available_platforms": []string{"web", "douyin", "xiaohongshu", "zhipu"},
	}

	return stats
}

// HealthCheck 健康检查
func (st *SearchTool) HealthCheck(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"overall_status": "healthy",
		"components": map[string]interface{}{
			"jina_client":  "available",
			"social_media": "available",
			"zhipu_client": "available",
		},
	}

	return health
}
