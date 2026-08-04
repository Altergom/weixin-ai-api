package ilink

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL, appID, channelVersion, clientVersion string
	httpClient                                    *http.Client
}

type QRCode struct {
	Code  string `json:"qrcode"`
	URL   string `json:"qrcode_url,omitempty"`
	Image string `json:"qrcode_image,omitempty"`
}

type LoginStatus struct {
	Status    string `json:"status"`
	AccountID string `json:"account_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	BotToken  string `json:"-"`
	Error     string `json:"error,omitempty"`
}

func NewClient(baseURL, appID, version string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://ilinkai.weixin.qq.com"
	}
	if appID == "" {
		appID = "bot"
	}
	if version == "" {
		version = "1.0.0"
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), appID: appID, channelVersion: version, clientVersion: versionHeader(version), httpClient: &http.Client{Timeout: 45 * time.Second}}
}

func (c *Client) QRCode(ctx context.Context) (*QRCode, error) {
	data, err := c.request(ctx, http.MethodPost, "/ilink/bot/get_bot_qrcode?bot_type=3", "", map[string]any{"local_token_list": []string{}}, false)
	if err != nil {
		return nil, err
	}
	image := normalizeImage(stringValue(data, "qrcode_img_content", "qrcodeImage"))
	qrURL := stringValue(data, "qrcode_url", "qrcodeUrl")
	if qrURL == "" && (strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://")) {
		qrURL, image = image, ""
	}
	return &QRCode{Code: stringValue(data, "qrcode"), URL: qrURL, Image: image}, nil
}

func (c *Client) QRCodeStatus(ctx context.Context, code string) (*LoginStatus, error) {
	query := url.Values{"qrcode": []string{code}}
	data, err := c.request(ctx, http.MethodGet, "/ilink/bot/get_qrcode_status?"+query.Encode(), "", nil, false)
	if err != nil {
		return nil, err
	}
	return &LoginStatus{Status: normalizeStatus(stringValue(data, "status")), AccountID: deepString(data, "account_id", "accountId", "bot_id", "botId"), UserID: deepString(data, "user_id", "userId", "ilink_user_id", "ilinkUserId"), BotToken: deepString(data, "bot_token", "botToken", "token"), Error: stringValue(data, "error", "errmsg", "message")}, nil
}

func (c *Client) request(ctx context.Context, method, path, token string, payload any, auth bool) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("iLink-App-Id", c.appID)
	req.Header.Set("iLink-App-ClientVersion", c.clientVersion)
	if auth {
		req.Header.Set("AuthorizationType", "ilink_bot_token")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-WECHAT-UIN", randomUIN())
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decode ilink response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ilink http %d: %s", resp.StatusCode, stringValue(data, "message", "msg", "error"))
	}
	if code, ok := data["ret"].(float64); ok && code != 0 {
		return nil, fmt.Errorf("ilink business error %v: %s", code, stringValue(data, "errmsg", "message", "error"))
	}
	if nested, ok := data["data"].(map[string]any); ok {
		for k, v := range nested {
			data[k] = v
		}
	}
	return data, nil
}

func stringValue(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
func deepString(v any, keys ...string) string {
	if m, ok := v.(map[string]any); ok {
		if s := stringValue(m, keys...); s != "" {
			return s
		}
		for _, child := range m {
			if s := deepString(child, keys...); s != "" {
				return s
			}
		}
	}
	return ""
}
func normalizeStatus(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "wait"
	}
	if s == "scanned" {
		return "scaned"
	}
	return s
}
func normalizeImage(s string) string {
	if s == "" || strings.HasPrefix(s, "data:image") || strings.HasPrefix(s, "http") {
		return s
	}
	if strings.Contains(s, "<svg") {
		return "data:image/svg+xml;charset=utf-8," + url.PathEscape(s)
	}
	return "data:image/png;base64," + strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)
}
func randomUIN() string {
	n := rand.New(rand.NewSource(time.Now().UnixNano())).Uint32()
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", n)))
}
func versionHeader(v string) string {
	var a, b, d int
	fmt.Sscanf(strings.TrimPrefix(v, "v"), "%d.%d.%d", &a, &b, &d)
	return fmt.Sprintf("%d", ((a&255)<<16)|((b&255)<<8)|(d&255))
}
func clientID() string { return "weixin-ilink-service-" + randomID() }

func randomID() string {
	b := make([]byte, 16)
	if _, err := cryptorand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}
