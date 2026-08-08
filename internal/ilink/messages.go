package ilink

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrSessionExpired 表示 getupdates 返回 iLink errcode -14：Bot 会话已失效，
// 账号必须重新认证。
var ErrSessionExpired = errors.New("ilink session expired")

const (
	sessionExpiredCode = -14
	textMessageType    = 1
	textItemType       = 1
)

// TextMessage 是一条发给 Bot 的入站文本消息。非文本消息在到达调用方前已过滤。
type TextMessage struct {
	Kind         string     `json:"type"`
	PeerID       string     `json:"from_user_id"`
	ContextToken string     `json:"context_token,omitempty"`
	Text         string     `json:"text,omitempty"`
	Media        *MediaInfo `json:"media,omitempty"`
}

// MediaInfo 保存媒体消息的 iLink CDN 引用。
type MediaInfo struct {
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AESKey            string `json:"aes_key,omitempty"`
	FullURL           string `json:"full_url,omitempty"`
	EncodeType        int    `json:"encode_type,omitempty"`
}

// UpdatesResult 是一次 getupdates 响应。Cursor 必须持久化并作为下一次请求的
// cursor；LongPollTimeout 是服务端建议的等待时间。
type UpdatesResult struct {
	Cursor          string
	LongPollTimeout time.Duration
	Messages        []TextMessage
}

type updatesResponse struct {
	Code            int    `json:"errcode"`
	Cursor          string `json:"get_updates_buf"`
	LongPollTimeout int64  `json:"longpolling_timeout_ms"`
	MessageList     []struct {
		MessageType  int    `json:"message_type"`
		FromUserID   string `json:"from_user_id"`
		ContextToken string `json:"context_token"`
		ItemList     []struct {
			Type     int `json:"type"`
			TextItem struct {
				Text string `json:"text"`
			} `json:"text_item"`
			ImageItem struct {
				Media struct {
					EncryptQueryParam string `json:"encrypt_query_param"`
					AESKey            string `json:"aes_key"`
					FullURL           string `json:"full_url"`
				} `json:"media"`
			} `json:"image_item"`
			VoiceItem struct {
				Media struct {
					EncryptQueryParam string `json:"encrypt_query_param"`
					AESKey            string `json:"aes_key"`
					FullURL           string `json:"full_url"`
				} `json:"media"`
				EncodeType int    `json:"encode_type"`
				Text       string `json:"text"`
			} `json:"voice_item"`
		} `json:"item_list"`
	} `json:"msgs"`
}

// GetUpdates 使用已保存的 cursor 长轮询新消息。
// 当 iLink 报告会话失效时返回 ErrSessionExpired。
func (c *Client) GetUpdates(ctx context.Context, baseURL, token, cursor string) (*UpdatesResult, error) {
	target, err := c.resolveBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"get_updates_buf": cursor, "base_info": c.baseInfo()}
	var response updatesResponse
	if err := c.doJSONAt(ctx, target, http.MethodPost, "ilink/bot/getupdates", token, body, &response); err != nil {
		return nil, fmt.Errorf("ilink getupdates: %w", err)
	}
	if response.Code == sessionExpiredCode {
		return nil, ErrSessionExpired
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("ilink getupdates: errcode %d", response.Code)
	}
	return response.result(), nil
}

func (r updatesResponse) result() *UpdatesResult {
	result := &UpdatesResult{
		Cursor:          r.Cursor,
		LongPollTimeout: time.Duration(r.LongPollTimeout) * time.Millisecond,
	}
	for _, message := range r.MessageList {
		if message.MessageType != textMessageType {
			continue
		}
		var text strings.Builder
		kind := ""
		var media *MediaInfo
		for _, item := range message.ItemList {
			if item.Type == textItemType {
				kind = "text"
				text.WriteString(item.TextItem.Text)
			} else if item.Type == 2 {
				kind = "image"
				media = &MediaInfo{
					EncryptQueryParam: item.ImageItem.Media.EncryptQueryParam,
					AESKey:            item.ImageItem.Media.AESKey,
					FullURL:           item.ImageItem.Media.FullURL,
				}
			} else if item.Type == 3 {
				kind = "voice"
				media = &MediaInfo{
					EncryptQueryParam: item.VoiceItem.Media.EncryptQueryParam,
					AESKey:            item.VoiceItem.Media.AESKey,
					FullURL:           item.VoiceItem.Media.FullURL,
					EncodeType:        item.VoiceItem.EncodeType,
				}
				if item.VoiceItem.Text != "" {
					text.WriteString(item.VoiceItem.Text)
				}
			}
		}
		if kind == "" || (kind == "text" && text.Len() == 0) {
			continue
		}
		result.Messages = append(result.Messages, TextMessage{
			Kind:         kind,
			PeerID:       message.FromUserID,
			ContextToken: message.ContextToken,
			Text:         text.String(),
			Media:        media,
		})
	}
	return result
}
