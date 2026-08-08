package ilink

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/app"
)

func newServerClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(app.ILinkConfig{
		BaseURL:       server.URL,
		AppID:         "test-app",
		ClientVersion: 1,
		BotAgent:      "test-agent",
	}, server.Client(), nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func TestGetUpdatesFiltersTextMessages(t *testing.T) {
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/getupdates" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Fatalf("Authorization = %q", got)
		}
		var body struct {
			Cursor   string         `json:"get_updates_buf"`
			BaseInfo map[string]any `json:"base_info"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Cursor != "cur-1" {
			t.Fatalf("cursor = %q", body.Cursor)
		}
		if body.BaseInfo["channel_version"] != "1" || body.BaseInfo["bot_agent"] != "test-agent" {
			t.Fatalf("base_info = %#v", body.BaseInfo)
		}
		_, _ = w.Write([]byte(`{"errcode":0,"get_updates_buf":"cur-2","longpolling_timeout_ms":5000,
			"msgs":[
				{"message_type":1,"from_user_id":"peer","context_token":"ctx","item_list":[{"type":1,"text_item":{"text":"he"}},{"type":1,"text_item":{"text":"llo"}}]},
				{"message_type":2,"from_user_id":"other","item_list":[{"type":1,"text_item":{"text":"skip"}}]}
			]}`))
	})

	result, err := client.GetUpdates(context.Background(), "", "tok", "cur-1")
	if err != nil {
		t.Fatalf("GetUpdates() error = %v", err)
	}
	if result.Cursor != "cur-2" || result.LongPollTimeout != 5*time.Second {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("messages = %#v", result.Messages)
	}
	got := result.Messages[0]
	if got.PeerID != "peer" || got.ContextToken != "ctx" || got.Text != "hello" {
		t.Fatalf("message = %+v", got)
	}
}

func TestGetUpdatesSessionExpired(t *testing.T) {
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":-14}`))
	})
	_, err := client.GetUpdates(context.Background(), "", "tok", "")
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("GetUpdates() error = %v, want ErrSessionExpired", err)
	}
}

func TestGetUpdatesQueuesImageAndVoice(t *testing.T) {
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0,"get_updates_buf":"cur-media","msgs":[
			{"message_type":1,"from_user_id":"peer-image","context_token":"ctx-image","item_list":[{"type":2,"image_item":{"media":{"encrypt_query_param":"img-param","aes_key":"img-key"}}}]},
			{"message_type":1,"from_user_id":"peer-voice","context_token":"ctx-voice","item_list":[{"type":3,"voice_item":{"encode_type":6,"text":"语音文字","media":{"encrypt_query_param":"voice-param","aes_key":"voice-key"}}}]}
		]}`))
	})
	result, err := client.GetUpdates(context.Background(), "", "tok", "")
	if err != nil {
		t.Fatalf("GetUpdates() error = %v", err)
	}
	if len(result.Messages) != 2 {
		t.Fatalf("messages = %#v", result.Messages)
	}
	if result.Messages[0].Kind != "image" || result.Messages[0].Media == nil || result.Messages[0].Media.AESKey != "img-key" {
		t.Fatalf("image = %+v", result.Messages[0])
	}
	voice := result.Messages[1]
	if voice.Kind != "voice" || voice.Text != "语音文字" || voice.Media == nil || voice.Media.EncodeType != 6 {
		t.Fatalf("voice = %+v", voice)
	}
}
