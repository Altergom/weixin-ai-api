package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Altergom/weixin-ai-api/internal/config"
)

func TestHandlerLoginFlow(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/ilink/bot/get_bot_qrcode":
			_, _ = io.WriteString(w, `{"ret":0,"qrcode":"qr-code","qrcode_url":"https://example.com/qr"}`)
		case "/ilink/bot/get_qrcode_status":
			_, _ = io.WriteString(w, `{"ret":0,"status":"confirmed","bot_token":"private-token","account_id":"account-1"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	handler := newHandler(config.Config{ILinkBaseURL: upstream.URL, ILinkAppID: "bot", ILinkClientVersion: "1.0.0"})
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/wechat/qrcode", nil))
	if create.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		SessionID string `json:"session_id"`
		QRCode    struct {
			Code string `json:"qrcode"`
		} `json:"qrcode"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.SessionID == "" || created.QRCode.Code != "qr-code" {
		t.Fatalf("unexpected create response: %s", create.Body.String())
	}

	status := httptest.NewRecorder()
	handler.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/wechat/qrcode/status?session_id="+created.SessionID, nil))
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"status":"confirmed"`) {
		t.Fatalf("status = %d, body = %s", status.Code, status.Body.String())
	}
	if strings.Contains(status.Body.String(), "private-token") {
		t.Fatal("bot token leaked in response")
	}
}

func TestHandlerRejectsMissingSessionID(t *testing.T) {
	handler := newHandler(config.Config{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/wechat/qrcode/status", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("got status %d", response.Code)
	}
}
