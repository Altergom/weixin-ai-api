package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/app"
)

const (
	defaultFirstByteTimeout = 60 * time.Second
	defaultSystemPrompt     = "You are a helpful assistant."
	completionsPath         = "/v1/chat/completions"
)

// ErrFirstByteTimeout 表示模型在首字节截止时间前没有返回响应头，
// 本次调用没有产生可用回复。
var ErrFirstByteTimeout = errors.New("model response timed out before first byte")

// Client 通过 SSE 调用 OpenAI 兼容的聊天补全接口。客户端无状态，
// API Key 按次传入，既不保存也不记录。
type Client struct {
	endpoint         string
	modelName        string
	systemPrompt     string
	firstByteTimeout time.Duration
	httpClient       *http.Client
	log              *slog.Logger
}

// NewClient 根据非敏感配置创建模型客户端。httpClient 不应设置全局 Timeout，
// 避免流式正文中途被截断；首字节截止时间由每次调用单独控制。
func NewClient(cfg app.ModelConfig, httpClient *http.Client, logger *slog.Logger) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, errors.New("model base URL is required")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return nil, errors.New("model name is required")
	}
	timeout := cfg.RequestTimeout.Duration()
	if timeout <= 0 {
		timeout = defaultFirstByteTimeout
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	systemPrompt := strings.TrimSpace(cfg.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}
	return &Client{
		endpoint:         base + completionsPath,
		modelName:        strings.TrimSpace(cfg.Name),
		systemPrompt:     systemPrompt,
		firstByteTimeout: timeout,
		httpClient:       httpClient,
		log:              logger,
	}, nil
}

func (c *Client) newRequest(ctx context.Context, apiKey string, prompt Prompt) (*http.Request, error) {
	data, err := json.Marshal(c.buildRequest(prompt))
	if err != nil {
		return nil, fmt.Errorf("encode model request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create model request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if key := strings.TrimSpace(apiKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return req, nil
}
