package model

import (
	"testing"

	"github.com/Altergom/weixin-ai-api/internal/app"
)

func TestBuildRequestOmitsMetadata(t *testing.T) {
	client, err := NewClient(app.ModelConfig{BaseURL: "https://model.test", Name: "gpt-x"}, nil, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	req := client.buildRequest(Prompt{
		AccountID:    "acct",
		PeerID:       "peer",
		ContextToken: "ctx",
		MessageID:    "msg",
		Text:         "hi there",
	})
	if req.Model != "gpt-x" || !req.Stream {
		t.Fatalf("request = %+v", req)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("messages = %#v", req.Messages)
	}
	if req.Messages[0].Role != "system" || req.Messages[0].Content != defaultSystemPrompt {
		t.Fatalf("messages[0] = %+v", req.Messages[0])
	}
	if req.Messages[1].Role != "user" || req.Messages[1].Content != "hi there" {
		t.Fatalf("messages[1] = %+v", req.Messages[1])
	}
}

func TestBuildRequestUsesConfiguredSystemPrompt(t *testing.T) {
	role := "你是一个只用中文回答的客服助手。"
	client, err := NewClient(app.ModelConfig{BaseURL: "https://model.test", Name: "m", SystemPrompt: "  " + role + "  "}, nil, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	req := client.buildRequest(Prompt{Text: "hi"})
	if req.Messages[0].Content != role {
		t.Fatalf("system prompt = %q, want %q", req.Messages[0].Content, role)
	}
}

func TestNewClientTrimsCompletionsPath(t *testing.T) {
	client, err := NewClient(app.ModelConfig{BaseURL: "https://model.test/", Name: "m"}, nil, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.endpoint != "https://model.test/v1/chat/completions" {
		t.Fatalf("endpoint = %q", client.endpoint)
	}
}
