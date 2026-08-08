package model

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/app"
)

func newModelClient(t *testing.T, server *httptest.Server, timeoutMS int) *Client {
	t.Helper()
	client, err := NewClient(app.ModelConfig{
		BaseURL:        server.URL,
		Name:           "test-model",
		RequestTimeout: app.Millis(time.Duration(timeoutMS) * time.Millisecond),
	}, server.Client(), nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func TestCompleteStreamsReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != completionsPath {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n" +
			"data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := newModelClient(t, server, 0)
	var deltas []string
	reply, err := client.Complete(context.Background(), "key", Prompt{Text: "hi"}, func(d string) error {
		deltas = append(deltas, d)
		return nil
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if reply != "你好" {
		t.Fatalf("reply = %q", reply)
	}
	if len(deltas) != 2 {
		t.Fatalf("deltas = %#v", deltas)
	}
}

func TestCompleteRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newModelClient(t, server, 0)
	if _, err := client.Complete(context.Background(), "key", Prompt{Text: "hi"}, nil); err == nil {
		t.Fatal("Complete() error = nil, want HTTP failure")
	}
}

func TestCompletePropagatesCancel(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	client := newModelClient(t, server, 0)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := client.Complete(ctx, "key", Prompt{Text: "hi"}, nil)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Complete() error = %v, want context.Canceled", err)
	}
}

func TestCompleteFirstByteTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	client := newModelClient(t, server, 40)
	_, err := client.Complete(context.Background(), "key", Prompt{Text: "hi"}, nil)
	if !errors.Is(err, ErrFirstByteTimeout) {
		t.Fatalf("Complete() error = %v, want ErrFirstByteTimeout", err)
	}
}
