package ilink

import (
	"context"
	"fmt"
	"net/http"
)

// NotifyStart 通知 iLink 本地连接器将开始为 token 对应账号执行长轮询。
// baseURL 是登录后得到的账号地址；为空时回退到配置地址。
func (c *Client) NotifyStart(ctx context.Context, baseURL, token string) error {
	if err := c.notify(ctx, baseURL, token, "ilink/bot/msg/notifystart"); err != nil {
		return fmt.Errorf("ilink notifystart: %w", err)
	}
	return nil
}

// NotifyStop 通知 iLink 连接器正在停止。该调用是尽力而为，即使退出上下文
// 很短，也应尝试执行。
func (c *Client) NotifyStop(ctx context.Context, baseURL, token string) error {
	if err := c.notify(ctx, baseURL, token, "ilink/bot/msg/notifystop"); err != nil {
		return fmt.Errorf("ilink notifystop: %w", err)
	}
	return nil
}

func (c *Client) notify(ctx context.Context, baseURL, token, endpoint string) error {
	target, err := c.resolveBaseURL(baseURL)
	if err != nil {
		return err
	}
	body := map[string]any{"base_info": c.baseInfo()}
	return c.doJSONAt(ctx, target, http.MethodPost, endpoint, token, body, nil)
}
