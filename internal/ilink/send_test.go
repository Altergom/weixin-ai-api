package ilink

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSendMessage(t *testing.T) {
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/sendmessage" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Msg struct {
				ToUserID     string `json:"to_user_id"`
				ContextToken string `json:"context_token"`
				ItemList     []struct {
					Type     int `json:"type"`
					TextItem struct {
						Text string `json:"text"`
					} `json:"text_item"`
				} `json:"item_list"`
			} `json:"msg"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Msg.ToUserID != "peer" || body.Msg.ContextToken != "ctx" {
			t.Fatalf("msg = %+v", body.Msg)
		}
		if len(body.Msg.ItemList) != 1 || body.Msg.ItemList[0].TextItem.Text != "hi" {
			t.Fatalf("item_list = %#v", body.Msg.ItemList)
		}
		_, _ = w.Write([]byte(`{"ret":0}`))
	})

	if err := client.SendMessage(context.Background(), "", "tok", "peer", "ctx", "hi"); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
}

func TestSendMessageNonZeroRet(t *testing.T) {
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ret":1}`))
	})
	if err := client.SendMessage(context.Background(), "", "tok", "peer", "ctx", "hi"); err == nil {
		t.Fatal("SendMessage() error = nil, want failure")
	}
}

func TestSendMessageRequiresPeer(t *testing.T) {
	client := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be called without peer ID")
	})
	if err := client.SendMessage(context.Background(), "", "tok", "", "ctx", "hi"); err == nil {
		t.Fatal("SendMessage() error = nil, want peer validation error")
	}
}
