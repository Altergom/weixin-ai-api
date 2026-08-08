package sse

import (
	"errors"
	"strings"
	"testing"
)

func TestParseChatCompletion(t *testing.T) {
	stream := strings.NewReader("event: message\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n" +
		"data: [DONE]\n\n")

	var deltas []string
	text, err := ParseChatCompletion(stream, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseChatCompletion() error = %v", err)
	}
	if text != "你好" {
		t.Fatalf("ParseChatCompletion() text = %q, want %q", text, "你好")
	}
	if got := strings.Join(deltas, ""); got != "你好" {
		t.Fatalf("received deltas = %q, want %q", got, "你好")
	}
}

func TestParseChatCompletionRejectsIncompleteStream(t *testing.T) {
	_, err := ParseChatCompletion(strings.NewReader("data: {\"choices\":[]}\n\n"), nil)
	if !errors.Is(err, ErrStreamIncomplete) {
		t.Fatalf("ParseChatCompletion() error = %v, want ErrStreamIncomplete", err)
	}
}

func TestParseChatCompletionRejectsMalformedEvent(t *testing.T) {
	_, err := ParseChatCompletion(strings.NewReader("data: not-json\n\n"), nil)
	if err == nil || !strings.Contains(err.Error(), "decode model SSE event") {
		t.Fatalf("ParseChatCompletion() error = %v, want decode error", err)
	}
}
