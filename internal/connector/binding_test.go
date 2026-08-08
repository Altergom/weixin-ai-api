package connector

import (
	"testing"

	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

func TestNewBindingFromConfirmedStatus(t *testing.T) {
	binding, err := NewBinding(&ilink.QRCodeStatus{
		Status:       "confirmed",
		BotToken:     "secret",
		AccountID:    "bot@im.bot",
		BaseURL:      "https://route.example.test",
		WeixinUserID: "user@im.wechat",
	})
	if err != nil {
		t.Fatalf("NewBinding() error = %v", err)
	}
	if binding.AccountID != "bot@im.bot" || binding.BotToken != "secret" {
		t.Fatalf("binding = %+v", binding)
	}
	if binding.BaseURL != "https://route.example.test" || binding.WeixinUserID != "user@im.wechat" {
		t.Fatalf("binding = %+v", binding)
	}
	if binding.Status != store.ConnectionStatusDisconnected {
		t.Fatalf("status = %q", binding.Status)
	}
}

func TestNewBindingRejectsIncomplete(t *testing.T) {
	if _, err := NewBinding(&ilink.QRCodeStatus{Status: "confirmed", AccountID: "bot"}); err == nil {
		t.Fatal("NewBinding() error = nil, want validation error for missing token/baseurl")
	}
	if _, err := NewBinding(nil); err == nil {
		t.Fatal("NewBinding(nil) error = nil, want error")
	}
}
