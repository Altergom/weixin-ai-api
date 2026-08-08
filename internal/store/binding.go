package store

import (
	"fmt"
	"strings"
	"time"
)

// ConnectionStatus 表示本地 iLink 连接器状态。
type ConnectionStatus string

const (
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusConnecting   ConnectionStatus = "connecting"
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusFailed       ConnectionStatus = "failed"
)

// Binding 是一个 iLink Bot 账号的全部持久化状态。BotToken 属于凭据，
// 调用方不得把 Binding 写入日志或 HTTP 响应。
type Binding struct {
	AccountID    string           `json:"account_id"`
	WeixinUserID string           `json:"weixin_user_id"`
	BaseURL      string           `json:"base_url"`
	BotToken     string           `json:"bot_token"`
	Cursor       string           `json:"cursor,omitempty"`
	Status       ConnectionStatus `json:"status"`
	LastError    string           `json:"last_error,omitempty"`
	ConnectedAt  time.Time        `json:"connected_at,omitempty"`
	LastEventAt  time.Time        `json:"last_event_at,omitempty"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// Validate 检查账号开始轮询前必须满足的字段约束。
func (b Binding) Validate() error {
	if strings.TrimSpace(b.AccountID) == "" {
		return fmt.Errorf("binding account ID is required")
	}
	if strings.TrimSpace(b.BaseURL) == "" {
		return fmt.Errorf("binding base URL is required")
	}
	if strings.TrimSpace(b.BotToken) == "" {
		return fmt.Errorf("binding bot token is required")
	}
	return nil
}

// PublicStatus 可安全返回给本地 GUI，刻意排除 BotToken、Cursor 和 context token。
type PublicStatus struct {
	AccountID    string           `json:"account_id,omitempty"`
	WeixinUserID string           `json:"weixin_user_id,omitempty"`
	Status       ConnectionStatus `json:"status"`
	LastError    string           `json:"last_error,omitempty"`
	ConnectedAt  time.Time        `json:"connected_at,omitempty"`
	LastEventAt  time.Time        `json:"last_event_at,omitempty"`
	UpdatedAt    time.Time        `json:"updated_at,omitempty"`
}

// Public 返回绑定中的非敏感部分。
func (b Binding) Public() PublicStatus {
	return PublicStatus{
		AccountID:    b.AccountID,
		WeixinUserID: b.WeixinUserID,
		Status:       b.Status,
		LastError:    b.LastError,
		ConnectedAt:  b.ConnectedAt,
		LastEventAt:  b.LastEventAt,
		UpdatedAt:    b.UpdatedAt,
	}
}
