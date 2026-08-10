package connector

import (
	"context"
	"errors"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

// run 使用指数退避重连，直到上下文取消或 iLink 会话过期。
// 会话过期需要重新扫码，不能自动重试。
func (c *Connector) run(ctx context.Context, binding store.Binding) {
	defer close(c.done)

	backoff := minBackoff
	for {
		err := c.connectAndPoll(ctx, &binding, func() { backoff = minBackoff })
		switch {
		case err == nil, errors.Is(err, context.Canceled):
			return
		case errors.Is(err, ilink.ErrSessionExpired):
			c.log.Warn("iLink session expired; stopping until re-scan", "account", binding.AccountID)
			c.persistStatus(binding, store.ConnectionStatusFailed, "session expired, re-scan required")
			return
		default:
			c.log.Warn("connection dropped; will retry", "err", err, "backoff", backoff.String())
			c.persistStatus(binding, store.ConnectionStatusFailed, err.Error())
			if !sleep(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
		}
	}
}

// connectAndPoll 运行一次连接会话：先 notifystart，再进入 getupdates 循环。
// 每次轮询成功后通过 onProgress 重置调用方的退避时间。
func (c *Connector) connectAndPoll(ctx context.Context, binding *store.Binding, onProgress func()) error {
	c.persistStatus(*binding, store.ConnectionStatusConnecting, "")
	if err := c.ilink.NotifyStart(ctx, binding.BaseURL, binding.BotToken); err != nil {
		return err
	}
	c.persistStatus(*binding, store.ConnectionStatusConnected, "")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.log.Info("开始轮询微信消息", "cursor_present", binding.Cursor != "")
		result, err := c.getUpdates(ctx, binding)
		if err != nil {
			c.log.Error("微信消息轮询失败", "error", err)
			return err
		}
		c.log.Info("微信消息轮询完成", "message_count", len(result.Messages), "cursor_changed", result.Cursor != binding.Cursor, "long_poll_timeout", result.LongPollTimeout)
		onProgress()
		if result.Cursor != binding.Cursor {
			binding.Cursor = result.Cursor
			c.persistStatus(*binding, store.ConnectionStatusConnected, "")
		}
		for _, message := range result.Messages {
			binding.LastEventAt = time.Now().UTC()
			c.persistStatus(*binding, store.ConnectionStatusConnected, "")
			c.handleMessage(ctx, *binding, message)
		}
		if result.LongPollTimeout > 0 && len(result.Messages) == 0 {
			if !sleep(ctx, result.LongPollTimeout) {
				return ctx.Err()
			}
		}
	}
}

// handleMessage 将一条消息交给模型并发送回复。发送失败只记录并跳过，
// 不让一条异常回复拆掉整个循环。
func (c *Connector) handleMessage(ctx context.Context, binding store.Binding, message ilink.TextMessage) {
	c.log.Info("微信消息入队", "peer_id", message.PeerID, "kind", message.Kind, "has_context_token", message.ContextToken != "")
	if err := c.store.SaveContextToken(binding.AccountID, message.PeerID, message.ContextToken); err != nil {
		c.log.Warn("save context token failed", "err", err)
	}
	// MessageID 暂为空：ilink.TextMessage 当前还没有该字段。
	c.queue.Enqueue(message)
}

// getUpdates 为一次长轮询设置上限，避免死连接永久阻塞循环。
// iLink HTTP 客户端不能设置全局超时（长轮询需要保持请求），因此每次轮询
// 单独设置一个高于服务端 longpolling_timeout_ms 的上限。
func (c *Connector) getUpdates(ctx context.Context, binding *store.Binding) (*ilink.UpdatesResult, error) {
	pollCtx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()
	return c.ilink.GetUpdates(pollCtx, binding.BaseURL, binding.BotToken, binding.Cursor)
}

func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

// sleep 等待 d 或上下文取消；被取消时返回 false。
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
