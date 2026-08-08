package ilink

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// SendMessage 向 peerID 发送纯文本回复。contextToken 必须是该联系人最新保存的
// token，iLink 才能保持会话上下文。
func (c *Client) SendMessage(ctx context.Context, baseURL, token, peerID, contextToken, text string) error {
	if strings.TrimSpace(peerID) == "" {
		return fmt.Errorf("ilink sendmessage: peer ID is required")
	}
	target, err := c.resolveBaseURL(baseURL)
	if err != nil {
		return err
	}
	body := map[string]any{
		"msg": map[string]any{
			"to_user_id":    peerID,
			"context_token": contextToken,
			"item_list": []map[string]any{{
				"type":      textItemType,
				"text_item": map[string]any{"text": text},
			}},
		},
		"base_info": c.baseInfo(),
	}
	var response struct {
		Ret int `json:"ret"`
	}
	if err := c.doJSONAt(ctx, target, http.MethodPost, "ilink/bot/sendmessage", token, body, &response); err != nil {
		return fmt.Errorf("ilink sendmessage: %w", err)
	}
	if response.Ret != 0 {
		return fmt.Errorf("ilink sendmessage: ret %d", response.Ret)
	}
	return nil
}
