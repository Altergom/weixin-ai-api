package ilink

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
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
			"from_user_id":  "",
			"to_user_id":    peerID,
			"client_id":     newClientID(),
			"message_type":  2,
			"message_state": 2,
			"item_list": []map[string]any{{
				"type":      textItemType,
				"text_item": map[string]any{"text": text},
			}},
		},
		"base_info": c.baseInfo(),
	}
	if strings.TrimSpace(contextToken) != "" {
		body["msg"].(map[string]any)["context_token"] = contextToken
	}
	c.log.Info("发送微信消息请求", "peer_id", peerID, "has_context_token", strings.TrimSpace(contextToken) != "", "text_length", len([]rune(text)))
	var response struct {
		Ret int `json:"ret"`
	}
	if err := c.doJSONAt(ctx, target, http.MethodPost, "ilink/bot/sendmessage", token, body, &response); err != nil {
		return fmt.Errorf("ilink sendmessage: %w", err)
	}
	if response.Ret != 0 {
		return fmt.Errorf("ilink sendmessage: ret %d", response.Ret)
	}
	c.log.Info("微信消息接口返回成功", "peer_id", peerID, "ret", response.Ret)
	return nil
}

func newClientID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return "kawa-weixin-" + hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("kawa-weixin-%d", time.Now().UnixNano())
}
