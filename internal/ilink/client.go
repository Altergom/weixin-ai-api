package ilink

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/app"
)

const defaultRequestTimeout = 15 * time.Second

// Client 是无状态的轻量 iLink HTTP 客户端。账号凭据按请求传入，
// 持续保存在本地存储中。
type Client struct {
	baseURL       *url.URL
	appID         string
	clientVersion uint32
	botAgent      string
	httpClient    *http.Client
	log           *slog.Logger
	randomUIN     func() (string, error)
}

// APIError 刻意不包含原始响应正文，因为上游错误可能包含不适合记录日志的
// 请求标识或其他数据。
type APIError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("ilink API error: HTTP %d, code %d: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("ilink API error: HTTP %d: %s", e.StatusCode, e.Message)
}

// NewClient 根据非敏感应用配置创建客户端。
func NewClient(cfg app.ILinkConfig, httpClient *http.Client, logger *slog.Logger) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("invalid ilink base URL")
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return nil, errors.New("ilink app ID is required")
	}
	if cfg.ClientVersion == 0 {
		return nil, errors.New("ilink client version is required")
	}
	if httpClient == nil {
		timeout := cfg.RequestTimeout.Duration()
		if timeout <= 0 {
			timeout = defaultRequestTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		baseURL:       baseURL,
		appID:         strings.TrimSpace(cfg.AppID),
		clientVersion: cfg.ClientVersion,
		botAgent:      strings.TrimSpace(cfg.BotAgent),
		httpClient:    httpClient,
		log:           logger,
		randomUIN:     newRandomUIN,
	}, nil
}

func endpointURL(baseURL *url.URL, endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse ilink endpoint: %w", err)
	}
	base := *baseURL
	base.Path = path.Join(base.Path, parsed.Path)
	base.RawQuery = parsed.RawQuery
	return base.String(), nil
}

func (c *Client) newRequest(ctx context.Context, method, endpoint, token string, body any) (*http.Request, error) {
	return c.newRequestAt(ctx, c.baseURL, method, endpoint, token, body)
}

func (c *Client) newRequestAt(ctx context.Context, baseURL *url.URL, method, endpoint, token string, body any) (*http.Request, error) {
	urlString, err := endpointURL(baseURL, endpoint)
	if err != nil {
		return nil, err
	}

	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode ilink request: %w", err)
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlString, payload)
	if err != nil {
		return nil, fmt.Errorf("create ilink request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("iLink-App-Id", c.appID)
	req.Header.Set("iLink-App-ClientVersion", strconv.FormatUint(uint64(c.clientVersion), 10))

	if method != http.MethodGet {
		uin, err := c.randomUIN()
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-WECHAT-UIN", uin)
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("AuthorizationType", "ilink_bot_token")
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	return req, nil
}

func (c *Client) doJSON(ctx context.Context, method, endpoint, token string, body, output any) error {
	return c.doJSONAt(ctx, c.baseURL, method, endpoint, token, body, output)
}

func (c *Client) doJSONAt(ctx context.Context, baseURL *url.URL, method, endpoint, token string, body, output any) error {
	req, err := c.newRequestAt(ctx, baseURL, method, endpoint, token, body)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform ilink request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return fmt.Errorf("read ilink response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return newAPIError(resp.StatusCode, data)
	}
	if output == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode ilink response: %w", err)
	}
	return nil
}

// resolveBaseURL 在 raw 为空时返回配置中的地址，否则返回登录后或 IDC 重定向
// 得到的账号专属 baseurl。
func (c *Client) resolveBaseURL(raw string) (*url.URL, error) {
	if raw == "" {
		return c.baseURL, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid ilink base URL %q", raw)
	}
	return parsed, nil
}

func (c *Client) baseInfo() map[string]string {
	info := map[string]string{
		"channel_version": strconv.FormatUint(uint64(c.clientVersion), 10),
	}
	if c.botAgent != "" {
		info["bot_agent"] = c.botAgent
	}
	return info
}

func newAPIError(statusCode int, data []byte) error {
	var payload struct {
		Code    int    `json:"errcode"`
		Message string `json:"errmsg"`
	}
	_ = json.Unmarshal(data, &payload)
	if payload.Message == "" {
		payload.Message = http.StatusText(statusCode)
	}
	return &APIError{StatusCode: statusCode, Code: payload.Code, Message: payload.Message}
}

func newRandomUIN() (string, error) {
	var data [4]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("generate X-WECHAT-UIN: %w", err)
	}
	value := binary.BigEndian.Uint32(data[:])
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(value), 10))), nil
}
