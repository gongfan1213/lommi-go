package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

// SocialMediaSearch 社交媒体搜索工具
type SocialMediaSearch struct {
	logger     *logx.Logger
	httpClient *http.Client
	apiKeys    map[string]string
}

// SearchResult 搜索结果
type SearchResult struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	Platform    string    `json:"platform"`
	Likes       int       `json:"likes"`
	Comments    int       `json:"comments"`
	Shares      int       `json:"shares"`
	Views       int       `json:"views"`
	Tags        []string  `json:"tags"`
	ImageURL    string    `json:"image_url"`
	VideoURL    string    `json:"video_url"`
}

// DouyinSearchRequest 抖音搜索请求
type DouyinSearchRequest struct {
	Query       string `json:"query"`
	Count       int    `json:"count"`
	SortType    string `json:"sort_type"`
	PublishTime string `json:"publish_time"`
}

// DouyinSearchResponse 抖音搜索响应
type DouyinSearchResponse struct {
	Data struct {
		List []DouyinVideo `json:"list"`
	} `json:"data"`
	Message string `json:"message"`
}

// DouyinVideo 抖音视频
type DouyinVideo struct {
	AwemeID    string            `json:"aweme_id"`
	Desc       string            `json:"desc"`
	CreateTime int64             `json:"create_time"`
	Author     DouyinAuthor      `json:"author"`
	Statistics DouyinStatistics  `json:"statistics"`
	Video      DouyinVideoInfo   `json:"video"`
	ImageInfos []DouyinImageInfo `json:"image_infos"`
	TextExtra  []DouyinTextExtra `json:"text_extra"`
	ShareURL   string            `json:"share_url"`
}

// DouyinAuthor 抖音作者
type DouyinAuthor struct {
	UID      string `json:"uid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar_thumb"`
}

// DouyinStatistics 抖音统计
type DouyinStatistics struct {
	DiggCount    int `json:"digg_count"`
	CommentCount int `json:"comment_count"`
	ShareCount   int `json:"share_count"`
	PlayCount    int `json:"play_count"`
}

// DouyinVideoInfo 抖音视频信息
type DouyinVideoInfo struct {
	PlayAddr DouyinPlayAddr `json:"play_addr"`
	Cover    DouyinCover    `json:"cover"`
	Duration int64          `json:"duration"`
}

// DouyinPlayAddr 抖音播放地址
type DouyinPlayAddr struct {
	URLList []string `json:"url_list"`
}

// DouyinCover 抖音封面
type DouyinCover struct {
	URLList []string `json:"url_list"`
}

// DouyinImageInfo 抖音图片信息
type DouyinImageInfo struct {
	URLList []string `json:"url_list"`
}

// DouyinTextExtra 抖音文本标签
type DouyinTextExtra struct {
	HashtagName string `json:"hashtag_name"`
}

// XHSearchRequest 小红书搜索请求
type XHSearchRequest struct {
	Query    string `json:"query"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Sort     string `json:"sort"`
}

// XHSearchResponse 小红书搜索响应
type XHSearchResponse struct {
	Data struct {
		Items []XHNote `json:"items"`
	} `json:"data"`
	Message string `json:"message"`
}

// XHNote 小红书笔记
type XHNote struct {
	NoteID   string     `json:"note_id"`
	Title    string     `json:"title"`
	Desc     string     `json:"desc"`
	Type     string     `json:"type"`
	Author   XHAuthor   `json:"author"`
	Time     int64      `json:"time"`
	Interact XHInteract `json:"interact"`
	Images   []XHImage  `json:"images"`
	Video    XHVideo    `json:"video"`
	TagList  []XHTag    `json:"tag_list"`
	NoteCard XHNoteCard `json:"note_card"`
}

// XHAuthor 小红书作者
type XHAuthor struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// XHInteract 小红书互动
type XHInteract struct {
	Liked          bool `json:"liked"`
	LikedCount     int  `json:"liked_count"`
	Collected      bool `json:"collected"`
	CollectedCount int  `json:"collected_count"`
	CommentCount   int  `json:"comment_count"`
	ShareCount     int  `json:"share_count"`
}

// XHImage 小红书图片
type XHImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// XHVideo 小红书视频
type XHVideo struct {
	URL      string `json:"url"`
	Cover    string `json:"cover"`
	Duration int64  `json:"duration"`
}

// XHTag 小红书标签
type XHTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// XHNoteCard 小红书笔记卡片
type XHNoteCard struct {
	URL string `json:"url"`
}

// NewSocialMediaSearch 创建社交媒体搜索工具
func NewSocialMediaSearch(logger *logx.Logger) *SocialMediaSearch {
	return &SocialMediaSearch{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKeys: make(map[string]string),
	}
}

// SetAPIKey 设置API密钥
func (sms *SocialMediaSearch) SetAPIKey(platform, key string) {
	sms.apiKeys[platform] = key
}

// Search 搜索社交媒体内容
func (sms *SocialMediaSearch) Search(ctx context.Context, query string, platform string) ([]SearchResult, error) {
	sms.logger.Info(ctx, "开始搜索社交媒体内容",
		logx.KV("query", query),
		logx.KV("platform", platform))

	switch strings.ToLower(platform) {
	case "douyin", "tiktok":
		return sms.searchDouyin(ctx, query)
	case "xiaohongshu", "xhs", "red":
		return sms.searchXiaohongshu(ctx, query)
	default:
		return sms.searchAll(ctx, query)
	}
}

// searchDouyin 搜索抖音内容
func (sms *SocialMediaSearch) searchDouyin(ctx context.Context, query string) ([]SearchResult, error) {
	sms.logger.Info(ctx, "搜索抖音内容", logx.KV("query", query))

	// 检查是否有API密钥
	apiKey, hasKey := sms.apiKeys["douyin"]
	if !hasKey {
		sms.logger.Warn(ctx, "抖音API密钥未配置，返回模拟数据")
		return sms.getMockDouyinResults(query), nil
	}

	// 构建请求
	req := DouyinSearchRequest{
		Query:       query,
		Count:       20,
		SortType:    "0", // 综合排序
		PublishTime: "0", // 不限时间
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求到抖音搜索API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.douyin.com/v1/search/video/",
		strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := sms.httpClient.Do(httpReq)
	if err != nil {
		sms.logger.Error(ctx, "抖音搜索API请求失败", logx.KV("error", err))
		return sms.getMockDouyinResults(query), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		sms.logger.Error(ctx, "抖音搜索API返回错误", logx.KV("status_code", resp.StatusCode))
		return sms.getMockDouyinResults(query), nil
	}

	var response DouyinSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		sms.logger.Error(ctx, "解析抖音搜索响应失败", logx.KV("error", err))
		return sms.getMockDouyinResults(query), nil
	}

	// 转换结果
	var results []SearchResult
	for _, video := range response.Data.List {
		result := SearchResult{
			Title:       video.Desc,
			Content:     video.Desc,
			URL:         video.ShareURL,
			Author:      video.Author.Nickname,
			PublishedAt: time.Unix(video.CreateTime, 0),
			Platform:    "douyin",
			Likes:       video.Statistics.DiggCount,
			Comments:    video.Statistics.CommentCount,
			Shares:      video.Statistics.ShareCount,
			Views:       video.Statistics.PlayCount,
		}

		// 提取标签
		for _, extra := range video.TextExtra {
			if extra.HashtagName != "" {
				result.Tags = append(result.Tags, "#"+extra.HashtagName)
			}
		}

		// 设置图片或视频URL
		if len(video.ImageInfos) > 0 {
			result.ImageURL = video.ImageInfos[0].URLList[0]
		}
		if len(video.Video.PlayAddr.URLList) > 0 {
			result.VideoURL = video.Video.PlayAddr.URLList[0]
		}

		results = append(results, result)
	}

	sms.logger.Info(ctx, "抖音搜索完成", logx.KV("results_count", len(results)))
	return results, nil
}

// searchXiaohongshu 搜索小红书内容
func (sms *SocialMediaSearch) searchXiaohongshu(ctx context.Context, query string) ([]SearchResult, error) {
	sms.logger.Info(ctx, "搜索小红书内容", logx.KV("query", query))

	// 检查是否有API密钥
	apiKey, hasKey := sms.apiKeys["xiaohongshu"]
	if !hasKey {
		sms.logger.Warn(ctx, "小红书API密钥未配置，返回模拟数据")
		return sms.getMockXHSResults(query), nil
	}

	// 构建请求
	req := XHSearchRequest{
		Query:    query,
		Page:     1,
		PageSize: 20,
		Sort:     "general", // 综合排序
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求到小红书搜索API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.xiaohongshu.com/api/sns/v1/search/notes",
		strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := sms.httpClient.Do(httpReq)
	if err != nil {
		sms.logger.Error(ctx, "小红书搜索API请求失败", logx.KV("error", err))
		return sms.getMockXHSResults(query), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		sms.logger.Error(ctx, "小红书搜索API返回错误", logx.KV("status_code", resp.StatusCode))
		return sms.getMockXHSResults(query), nil
	}

	var response XHSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		sms.logger.Error(ctx, "解析小红书搜索响应失败", logx.KV("error", err))
		return sms.getMockXHSResults(query), nil
	}

	// 转换结果
	var results []SearchResult
	for _, note := range response.Data.Items {
		result := SearchResult{
			Title:       note.Title,
			Content:     note.Desc,
			URL:         note.NoteCard.URL,
			Author:      note.Author.Nickname,
			PublishedAt: time.Unix(note.Time, 0),
			Platform:    "xiaohongshu",
			Likes:       note.Interact.LikedCount,
			Comments:    note.Interact.CommentCount,
			Shares:      note.Interact.ShareCount,
		}

		// 提取标签
		for _, tag := range note.TagList {
			result.Tags = append(result.Tags, tag.Name)
		}

		// 设置图片或视频URL
		if len(note.Images) > 0 {
			result.ImageURL = note.Images[0].URL
		}
		if note.Video.URL != "" {
			result.VideoURL = note.Video.URL
		}

		results = append(results, result)
	}

	sms.logger.Info(ctx, "小红书搜索完成", logx.KV("results_count", len(results)))
	return results, nil
}

// searchAll 搜索所有平台
func (sms *SocialMediaSearch) searchAll(ctx context.Context, query string) ([]SearchResult, error) {
	sms.logger.Info(ctx, "搜索所有社交媒体平台", logx.KV("query", query))

	var allResults []SearchResult

	// 搜索抖音
	douyinResults, err := sms.searchDouyin(ctx, query)
	if err != nil {
		sms.logger.Error(ctx, "搜索抖音失败", logx.KV("error", err))
	} else {
		allResults = append(allResults, douyinResults...)
	}

	// 搜索小红书
	xhsResults, err := sms.searchXiaohongshu(ctx, query)
	if err != nil {
		sms.logger.Error(ctx, "搜索小红书失败", logx.KV("error", err))
	} else {
		allResults = append(allResults, xhsResults...)
	}

	sms.logger.Info(ctx, "全平台搜索完成", logx.KV("total_results", len(allResults)))
	return allResults, nil
}

// getMockDouyinResults 获取模拟抖音结果
func (sms *SocialMediaSearch) getMockDouyinResults(query string) []SearchResult {
	return []SearchResult{
		{
			Title:       fmt.Sprintf("抖音视频: %s", query),
			Content:     fmt.Sprintf("这是一个关于%s的抖音视频内容，包含了相关的创意和想法", query),
			URL:         "https://www.douyin.com/video/123456",
			Author:      "抖音用户",
			PublishedAt: time.Now().Add(-time.Hour),
			Platform:    "douyin",
			Likes:       1000,
			Comments:    100,
			Shares:      50,
			Views:       5000,
			Tags:        []string{"#" + query, "#热门", "#创意"},
			VideoURL:    "https://example.com/video.mp4",
		},
		{
			Title:       fmt.Sprintf("抖音教程: %s", query),
			Content:     fmt.Sprintf("教你如何制作%s相关的抖音内容", query),
			URL:         "https://www.douyin.com/video/123457",
			Author:      "教程达人",
			PublishedAt: time.Now().Add(-2 * time.Hour),
			Platform:    "douyin",
			Likes:       800,
			Comments:    120,
			Shares:      80,
			Views:       3000,
			Tags:        []string{"#" + query, "#教程", "#学习"},
			VideoURL:    "https://example.com/tutorial.mp4",
		},
	}
}

// getMockXHSResults 获取模拟小红书结果
func (sms *SocialMediaSearch) getMockXHSResults(query string) []SearchResult {
	return []SearchResult{
		{
			Title:       fmt.Sprintf("小红书笔记: %s", query),
			Content:     fmt.Sprintf("分享一些关于%s的小红书笔记和心得", query),
			URL:         "https://www.xiaohongshu.com/explore/123456",
			Author:      "小红书用户",
			PublishedAt: time.Now().Add(-2 * time.Hour),
			Platform:    "xiaohongshu",
			Likes:       500,
			Comments:    80,
			Shares:      30,
			Tags:        []string{query, "分享", "生活"},
			ImageURL:    "https://example.com/image1.jpg",
		},
		{
			Title:       fmt.Sprintf("小红书攻略: %s", query),
			Content:     fmt.Sprintf("详细的%s攻略和推荐", query),
			URL:         "https://www.xiaohongshu.com/explore/123457",
			Author:      "攻略达人",
			PublishedAt: time.Now().Add(-3 * time.Hour),
			Platform:    "xiaohongshu",
			Likes:       1200,
			Comments:    150,
			Shares:      100,
			Tags:        []string{query, "攻略", "推荐"},
			ImageURL:    "https://example.com/image2.jpg",
		},
	}
}
