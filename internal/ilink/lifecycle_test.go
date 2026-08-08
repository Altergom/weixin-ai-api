package ilink

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestNotifyStartAndStop(t *testing.T) {
	var seen []string
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Fatalf("Authorization = %q", got)
		}
		var body struct {
			BaseInfo map[string]any `json:"base_info"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.BaseInfo["channel_version"] != "1" {
			t.Fatalf("base_info = %#v", body.BaseInfo)
		}
		_, _ = w.Write([]byte(`{}`))
	})

	if err := client.NotifyStart(context.Background(), "", "tok"); err != nil {
		t.Fatalf("NotifyStart() error = %v", err)
	}
	if err := client.NotifyStop(context.Background(), "", "tok"); err != nil {
		t.Fatalf("NotifyStop() error = %v", err)
	}
	if len(seen) != 2 || seen[0] != "/ilink/bot/msg/notifystart" || seen[1] != "/ilink/bot/msg/notifystop" {
		t.Fatalf("paths = %#v", seen)
	}
}

func TestNotifyStartUsesPerAccountBaseURL(t *testing.T) {
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	// 显式传入的 base URL 无效时，必须在发起请求前拒绝。
	if err := client.NotifyStart(context.Background(), "://bad", "tok"); err == nil {
		t.Fatal("NotifyStart() error = nil, want invalid base URL error")
	}
}
