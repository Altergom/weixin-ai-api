package connector

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

var _ ILinkClient = (*ilink.Client)(nil)

type fakeILink struct {
	mu      sync.Mutex
	started bool
	stopped bool
	updates chan *ilink.UpdatesResult
}

func (f *fakeILink) NotifyStart(context.Context, string, string) error {
	f.mu.Lock()
	f.started = true
	f.mu.Unlock()
	return nil
}
func (f *fakeILink) NotifyStop(context.Context, string, string) error {
	f.mu.Lock()
	f.stopped = true
	f.mu.Unlock()
	return nil
}
func (f *fakeILink) GetUpdates(ctx context.Context, _, _, _ string) (*ilink.UpdatesResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-f.updates:
		return result, nil
	}
}
func (f *fakeILink) SendMessage(context.Context, string, string, string, string, string) error {
	return nil
}

func newTestStore(t *testing.T) *store.FileStore {
	t.Helper()
	fs, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := fs.SaveBinding(store.Binding{AccountID: "bot@im.bot", BaseURL: "https://route.example.test", BotToken: "secret"}); err != nil {
		t.Fatalf("SaveBinding() error = %v", err)
	}
	return fs
}

func TestConnectorQueuesMessage(t *testing.T) {
	fs := newTestStore(t)
	il := &fakeILink{updates: make(chan *ilink.UpdatesResult, 1)}
	queue := NewMessageQueue()
	conn := New(il, fs, queue, nil)
	il.updates <- &ilink.UpdatesResult{Cursor: "cur-2", Messages: []ilink.TextMessage{{Kind: "text", PeerID: "peer", ContextToken: "ctx", Text: "hi"}}}
	if err := conn.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	message, err := queue.Next(context.Background())
	if err != nil {
		t.Fatalf("queue.Next() error = %v", err)
	}
	conn.Stop()
	if message.Kind != "text" || message.Text != "hi" || message.PeerID != "peer" || message.ContextToken != "ctx" {
		t.Fatalf("message = %+v", message)
	}
	il.mu.Lock()
	defer il.mu.Unlock()
	if !il.started || !il.stopped {
		t.Fatalf("started=%v stopped=%v", il.started, il.stopped)
	}
}

type expiringILink struct{ fakeILink }

func (e *expiringILink) GetUpdates(context.Context, string, string, string) (*ilink.UpdatesResult, error) {
	return nil, ilink.ErrSessionExpired
}

func TestConnectorStopsOnSessionExpired(t *testing.T) {
	fs := newTestStore(t)
	il := &expiringILink{}
	conn := New(il, fs, NewMessageQueue(), nil)
	if err := conn.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitFor(t, func() bool {
		binding, _ := fs.LoadBinding()
		return binding.Status == store.ConnectionStatusFailed
	})
	binding, _ := fs.LoadBinding()
	if binding.LastError == "" {
		t.Fatal("expected last_error to be recorded on session expiry")
	}
	conn.Stop()
}

func TestStartWithoutBindingErrors(t *testing.T) {
	fs, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := New(&fakeILink{}, fs, NewMessageQueue(), nil).Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil, want error for unbound account")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
