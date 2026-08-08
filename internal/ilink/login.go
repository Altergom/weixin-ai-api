package ilink

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

const defaultBotType = "3"

// QRCode 是短时有效的 iLink 登录挑战。Value 不得写入日志。
type QRCode struct {
	Value    string
	ImageURL string
}

// QRCodeStatus 是登录挑战的当前状态。BotToken 只会在 confirmed 后出现，
// 持久化时不得记录日志。
type QRCodeStatus struct {
	Status       string
	BotToken     string
	AccountID    string
	BaseURL      string
	WeixinUserID string
	RedirectHost string
}

// CreateQRCode 向 iLink 请求二维码。localTokens 用于重复登录时让 iLink
// 识别本地已知账号。
func (c *Client) CreateQRCode(ctx context.Context, localTokens []string) (*QRCode, error) {
	var response struct {
		QRCode         string `json:"qrcode"`
		QRCodeImageURL string `json:"qrcode_img_content"`
	}
	body := map[string]any{"local_token_list": localTokens}
	if err := c.doJSON(ctx, http.MethodPost, "ilink/bot/get_bot_qrcode?bot_type="+defaultBotType, "", body, &response); err != nil {
		return nil, fmt.Errorf("create ilink QR code: %w", err)
	}
	if response.QRCode == "" || response.QRCodeImageURL == "" {
		return nil, fmt.Errorf("create ilink QR code: response missing QR code data")
	}
	return &QRCode{Value: response.QRCode, ImageURL: response.QRCodeImageURL}, nil
}

// PollQRCodeStatus 在 baseURL 上等待二维码状态变化。baseURL 为空时使用配置地址。
// 收到 scaned_but_redirect 后，调用方必须使用返回的 RedirectHost 继续轮询。
func (c *Client) PollQRCodeStatus(ctx context.Context, baseURL, qrcode, verifyCode string) (*QRCodeStatus, error) {
	if qrcode == "" {
		return nil, fmt.Errorf("QR code is required")
	}
	targetURL, err := c.resolveBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	query := url.Values{"qrcode": []string{qrcode}}
	if verifyCode != "" {
		query.Set("verify_code", verifyCode)
	}
	var response struct {
		Status       string `json:"status"`
		BotToken     string `json:"bot_token"`
		AccountID    string `json:"ilink_bot_id"`
		BaseURL      string `json:"baseurl"`
		WeixinUserID string `json:"ilink_user_id"`
		RedirectHost string `json:"redirect_host"`
	}
	endpoint := "ilink/bot/get_qrcode_status?" + query.Encode()
	if err := c.doJSONAt(ctx, targetURL, http.MethodGet, endpoint, "", nil, &response); err != nil {
		return nil, fmt.Errorf("poll ilink QR code: %w", err)
	}
	if response.Status == "" {
		return nil, fmt.Errorf("poll ilink QR code: response missing status")
	}
	return &QRCodeStatus{
		Status:       response.Status,
		BotToken:     response.BotToken,
		AccountID:    response.AccountID,
		BaseURL:      response.BaseURL,
		WeixinUserID: response.WeixinUserID,
		RedirectHost: response.RedirectHost,
	}, nil
}
