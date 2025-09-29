package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// HTTPClient HTTP客户端
type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
	timeout time.Duration
}

// HTTPRequest HTTP请求配置
type HTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
	Params  map[string]string `json:"params"`
	Timeout time.Duration     `json:"timeout"`
	Context context.Context   `json:"-"`
}

// HTTPResponse HTTP响应
type HTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	Error      error               `json:"error,omitempty"`
}

// NewHTTPClient 创建新的HTTP客户端
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		headers: make(map[string]string),
		timeout: timeout,
	}
}

// SetHeader 设置默认请求头
func (c *HTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetHeaders 设置多个请求头
func (c *HTTPClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, path string, params map[string]string) (*HTTPResponse, error) {
	req := &HTTPRequest{
		Method:  "GET",
		URL:     c.buildURL(path),
		Params:  params,
		Context: ctx,
	}
	return c.Do(req)
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}) (*HTTPResponse, error) {
	req := &HTTPRequest{
		Method:  "POST",
		URL:     c.buildURL(path),
		Body:    body,
		Context: ctx,
	}
	return c.Do(req)
}

// Put 发送PUT请求
func (c *HTTPClient) Put(ctx context.Context, path string, body interface{}) (*HTTPResponse, error) {
	req := &HTTPRequest{
		Method:  "PUT",
		URL:     c.buildURL(path),
		Body:    body,
		Context: ctx,
	}
	return c.Do(req)
}

// Delete 发送DELETE请求
func (c *HTTPClient) Delete(ctx context.Context, path string) (*HTTPResponse, error) {
	req := &HTTPRequest{
		Method:  "DELETE",
		URL:     c.buildURL(path),
		Context: ctx,
	}
	return c.Do(req)
}

// Do 执行HTTP请求
func (c *HTTPClient) Do(req *HTTPRequest) (*HTTPResponse, error) {
	// 构建完整URL
	fullURL := req.URL
	if len(req.Params) > 0 {
		values := url.Values{}
		for k, v := range req.Params {
			values.Add(k, v)
		}
		fullURL += "?" + values.Encode()
	}

	// 准备请求体
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		case io.Reader:
			bodyReader = v
		default:
			// JSON序列化
			jsonData, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("序列化请求体失败: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(req.Context, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 设置Content-Type
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}

// buildURL 构建完整URL
func (c *HTTPClient) buildURL(path string) string {
	if c.baseURL == "" {
		return path
	}

	baseURL := strings.TrimSuffix(c.baseURL, "/")
	path = strings.TrimPrefix(path, "/")

	return baseURL + "/" + path
}

// ParseJSONResponse 解析JSON响应
func (resp *HTTPResponse) ParseJSONResponse(v interface{}) error {
	if resp.Error != nil {
		return resp.Error
	}

	return json.Unmarshal(resp.Body, v)
}

// GetHeader 获取响应头
func (resp *HTTPResponse) GetHeader(key string) string {
	if values := resp.Headers[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// IsSuccess 检查请求是否成功
func (resp *HTTPResponse) IsSuccess() bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// IsClientError 检查是否为客户端错误
func (resp *HTTPResponse) IsClientError() bool {
	return resp.StatusCode >= 400 && resp.StatusCode < 500
}

// IsServerError 检查是否为服务器错误
func (resp *HTTPResponse) IsServerError() bool {
	return resp.StatusCode >= 500
}

// String 返回响应体字符串
func (resp *HTTPResponse) String() string {
	return string(resp.Body)
}

// HTTPClientBuilder HTTP客户端构建器
type HTTPClientBuilder struct {
	client *HTTPClient
}

// NewHTTPClientBuilder 创建HTTP客户端构建器
func NewHTTPClientBuilder() *HTTPClientBuilder {
	return &HTTPClientBuilder{
		client: &HTTPClient{
			client:  &http.Client{},
			headers: make(map[string]string),
		},
	}
}

// SetBaseURL 设置基础URL
func (b *HTTPClientBuilder) SetBaseURL(baseURL string) *HTTPClientBuilder {
	b.client.baseURL = baseURL
	return b
}

// SetTimeout 设置超时时间
func (b *HTTPClientBuilder) SetTimeout(timeout time.Duration) *HTTPClientBuilder {
	b.client.timeout = timeout
	b.client.client.Timeout = timeout
	return b
}

// SetHeader 设置请求头
func (b *HTTPClientBuilder) SetHeader(key, value string) *HTTPClientBuilder {
	b.client.headers[key] = value
	return b
}

// SetHeaders 设置多个请求头
func (b *HTTPClientBuilder) SetHeaders(headers map[string]string) *HTTPClientBuilder {
	for k, v := range headers {
		b.client.headers[k] = v
	}
	return b
}

// SetUserAgent 设置User-Agent
func (b *HTTPClientBuilder) SetUserAgent(userAgent string) *HTTPClientBuilder {
	b.client.headers["User-Agent"] = userAgent
	return b
}

// SetAuthorization 设置Authorization头
func (b *HTTPClientBuilder) SetAuthorization(auth string) *HTTPClientBuilder {
	b.client.headers["Authorization"] = auth
	return b
}

// SetBearerToken 设置Bearer Token
func (b *HTTPClientBuilder) SetBearerToken(token string) *HTTPClientBuilder {
	b.client.headers["Authorization"] = "Bearer " + token
	return b
}

// SetBasicAuth 设置Basic Auth
func (b *HTTPClientBuilder) SetBasicAuth(username, password string) *HTTPClientBuilder {
	b.client.headers["Authorization"] = "Basic " +
		base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	return b
}

// SetContentType 设置Content-Type
func (b *HTTPClientBuilder) SetContentType(contentType string) *HTTPClientBuilder {
	b.client.headers["Content-Type"] = contentType
	return b
}

// Build 构建HTTP客户端
func (b *HTTPClientBuilder) Build() *HTTPClient {
	return b.client
}

// HTTPClientPool HTTP客户端池
type HTTPClientPool struct {
	clients []*HTTPClient
	current int
	mu      sync.RWMutex
}

// NewHTTPClientPool 创建HTTP客户端池
func NewHTTPClientPool(size int, baseURL string, timeout time.Duration) *HTTPClientPool {
	pool := &HTTPClientPool{
		clients: make([]*HTTPClient, size),
		current: 0,
	}

	for i := 0; i < size; i++ {
		pool.clients[i] = NewHTTPClient(baseURL, timeout)
	}

	return pool
}

// Get 获取客户端
func (p *HTTPClientPool) Get() *HTTPClient {
	p.mu.Lock()
	defer p.mu.Unlock()

	client := p.clients[p.current]
	p.current = (p.current + 1) % len(p.clients)
	return client
}

// SetHeaders 设置所有客户端的请求头
func (p *HTTPClientPool) SetHeaders(headers map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.SetHeaders(headers)
	}
}

// HTTPRequestBuilder HTTP请求构建器
type HTTPRequestBuilder struct {
	request *HTTPRequest
}

// NewHTTPRequestBuilder 创建HTTP请求构建器
func NewHTTPRequestBuilder() *HTTPRequestBuilder {
	return &HTTPRequestBuilder{
		request: &HTTPRequest{
			Headers: make(map[string]string),
			Params:  make(map[string]string),
		},
	}
}

// SetMethod 设置请求方法
func (b *HTTPRequestBuilder) SetMethod(method string) *HTTPRequestBuilder {
	b.request.Method = method
	return b
}

// SetURL 设置URL
func (b *HTTPRequestBuilder) SetURL(url string) *HTTPRequestBuilder {
	b.request.URL = url
	return b
}

// SetBody 设置请求体
func (b *HTTPRequestBuilder) SetBody(body interface{}) *HTTPRequestBuilder {
	b.request.Body = body
	return b
}

// SetHeader 设置请求头
func (b *HTTPRequestBuilder) SetHeader(key, value string) *HTTPRequestBuilder {
	b.request.Headers[key] = value
	return b
}

// SetHeaders 设置多个请求头
func (b *HTTPRequestBuilder) SetHeaders(headers map[string]string) *HTTPRequestBuilder {
	for k, v := range headers {
		b.request.Headers[k] = v
	}
	return b
}

// SetParam 设置URL参数
func (b *HTTPRequestBuilder) SetParam(key, value string) *HTTPRequestBuilder {
	b.request.Params[key] = value
	return b
}

// SetParams 设置多个URL参数
func (b *HTTPRequestBuilder) SetParams(params map[string]string) *HTTPRequestBuilder {
	for k, v := range params {
		b.request.Params[k] = v
	}
	return b
}

// SetTimeout 设置超时时间
func (b *HTTPRequestBuilder) SetTimeout(timeout time.Duration) *HTTPRequestBuilder {
	b.request.Timeout = timeout
	return b
}

// SetContext 设置上下文
func (b *HTTPRequestBuilder) SetContext(ctx context.Context) *HTTPRequestBuilder {
	b.request.Context = ctx
	return b
}

// Build 构建HTTP请求
func (b *HTTPRequestBuilder) Build() *HTTPRequest {
	return b.request
}
