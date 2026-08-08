package ilink

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Altergom/weixin-ai-api/internal/app"
)

func TestCreateQRCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/get_bot_qrcode" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("bot_type"); got != defaultBotType {
			t.Fatalf("bot_type = %q", got)
		}
		var body struct {
			LocalTokens []string `json:"local_token_list"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.LocalTokens) != 1 || body.LocalTokens[0] != "old-token" {
			t.Fatalf("local_token_list = %#v", body.LocalTokens)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"qrcode":"secret-qr","qrcode_img_content":"https://example.test/qr"}`))
	}))
	defer server.Close()

	client, err := NewClient(app.ILinkConfig{
		BaseURL:       server.URL,
		AppID:         "test-app",
		ClientVersion: 1,
	}, server.Client(), nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	qrcode, err := client.CreateQRCode(context.Background(), []string{"old-token"})
	if err != nil {
		t.Fatalf("CreateQRCode() error = %v", err)
	}
	if qrcode.Value != "secret-qr" || qrcode.ImageURL != "https://example.test/qr" {
		t.Fatalf("CreateQRCode() = %+v", qrcode)
	}
}

func TestPollQRCodeStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/get_qrcode_status" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("qrcode"); got != "qr-value" {
			t.Fatalf("qrcode = %q", got)
		}
		if got := r.URL.Query().Get("verify_code"); got != "1234" {
			t.Fatalf("verify_code = %q", got)
		}
		_, _ = w.Write([]byte(`{"status":"confirmed","bot_token":"secret","ilink_bot_id":"bot@im.bot","baseurl":"https://route.example.test","ilink_user_id":"user@im.wechat"}`))
	}))
	defer server.Close()

	client, err := NewClient(app.ILinkConfig{BaseURL: server.URL, AppID: "test", ClientVersion: 1}, server.Client(), nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	status, err := client.PollQRCodeStatus(context.Background(), "", "qr-value", "1234")
	if err != nil {
		t.Fatalf("PollQRCodeStatus() error = %v", err)
	}
	if status.Status != "confirmed" || status.AccountID != "bot@im.bot" || status.BotToken != "secret" {
		t.Fatalf("PollQRCodeStatus() = %+v", status)
	}
}
