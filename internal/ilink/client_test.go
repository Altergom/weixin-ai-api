package ilink

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/Altergom/weixin-ai-api/internal/app"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(app.ILinkConfig{
		BaseURL:       "https://ilink.example.test",
		AppID:         "test-app",
		ClientVersion: 65547,
	}, http.DefaultClient, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.randomUIN = func() (string, error) { return base64.StdEncoding.EncodeToString([]byte("42")), nil }
	return client
}

func TestNewRequestAddsRequiredHeaders(t *testing.T) {
	client := newTestClient(t)
	req, err := client.newRequest(context.Background(), http.MethodPost, "ilink/bot/getupdates", "secret", map[string]string{"x": "y"})
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer secret" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := req.Header.Get("AuthorizationType"); got != "ilink_bot_token" {
		t.Fatalf("AuthorizationType = %q", got)
	}
	if got := req.Header.Get("X-WECHAT-UIN"); got != "NDI=" {
		t.Fatalf("X-WECHAT-UIN = %q", got)
	}
	if got := req.Header.Get("iLink-App-Id"); got != "test-app" {
		t.Fatalf("iLink-App-Id = %q", got)
	}
	if got := req.Header.Get("iLink-App-ClientVersion"); got != "65547" {
		t.Fatalf("iLink-App-ClientVersion = %q", got)
	}
}

func TestNewRequestOmitsAuthorizationWithoutToken(t *testing.T) {
	client := newTestClient(t)
	req, err := client.newRequest(context.Background(), http.MethodGet, "ilink/bot/get_qrcode_status?qrcode=x", "", nil)
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, want empty", got)
	}
	if got := req.Header.Get("X-WECHAT-UIN"); got != "" {
		t.Fatalf("X-WECHAT-UIN = %q, want empty for GET", got)
	}
}
