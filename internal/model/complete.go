package model

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/model/sse"
)

// Complete 向模型发送 prompt，仅在流正常结束后返回完整回复。
// 半截流会返回错误且不返回文本，调用方不会把半条回复发出去。
// onDelta 非空时，会在每个内容增量到达时收到回调。apiKey 只用于本次请求鉴权。
func (c *Client) Complete(ctx context.Context, apiKey string, prompt Prompt, onDelta func(string) error) (string, error) {
	req, err := c.newRequest(ctx, apiKey, prompt)
	if err != nil {
		return "", err
	}

	// 设置首字节截止时间，但不限制流式正文时长：若响应头未及时到达，
	// 定时器取消请求上下文。httpClient 没有全局 Timeout，健康流可以持续更久。
	reqCtx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(reqCtx)

	timedOut := make(chan struct{})
	timer := time.AfterFunc(c.firstByteTimeout, func() {
		close(timedOut)
		cancel()
	})

	resp, err := c.httpClient.Do(req)
	firstByte := timer.Stop()
	if err != nil {
		return "", c.classify(ctx, timedOut, firstByte, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4*1024))
		return "", fmt.Errorf("model request failed: HTTP %d", resp.StatusCode)
	}

	reply, err := sse.ParseChatCompletion(resp.Body, onDelta)
	if err != nil {
		return "", c.classify(ctx, timedOut, true, err)
	}
	return reply, nil
}

// classify 将底层错误转换为调用方可读的错误，优先处理调用方取消、
// 首字节超时，最后才返回原始错误。
func (c *Client) classify(ctx context.Context, timedOut <-chan struct{}, gotFirstByte bool, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("model request canceled: %w", ctx.Err())
	}
	if !gotFirstByte {
		select {
		case <-timedOut:
			return ErrFirstByteTimeout
		default:
		}
	}
	return fmt.Errorf("model request failed: %w", err)
}
