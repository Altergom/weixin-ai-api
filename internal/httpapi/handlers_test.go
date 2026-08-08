package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Altergom/weixin-ai-api/internal/app"
	"github.com/Altergom/weixin-ai-api/internal/connector"
	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

type fakeCoordinator struct {
	status    store.PublicStatus
	started   bool
	stopped   bool
	restarted bool
	startErr  error
}

func (f *fakeCoordinator) Status() (store.PublicStatus, error) { return f.status, nil }
func (f *fakeCoordinator) Start(context.Context) error         { f.started = true; return f.startErr }
func (f *fakeCoordinator) Stop()                               { f.stopped = true }
func (f *fakeCoordinator) Restart(context.Context) error       { f.restarted = true; return nil }

func newTestServer(t *testing.T, deps Deps) *Server {
	t.Helper()
	return NewServer(app.ServerConfig{Host: "127.0.0.1", Port: 0}, slog.Default(), deps)
}

func do(s *Server, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	s.Engine().ServeHTTP(rec, req)
	return rec
}

func TestStatusOmitsToken(t *testing.T) {
	fc := &fakeCoordinator{status: store.PublicStatus{
		AccountID: "bot@im.bot",
		Status:    store.ConnectionStatusConnected,
	}}
	s := newTestServer(t, Deps{Coordinator: fc})

	rec := do(s, http.MethodGet, "/api/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "bot_token") || strings.Contains(body, "secret") {
		t.Fatalf("status leaked sensitive data: %s", body)
	}
	var got store.PublicStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != store.ConnectionStatusConnected {
		t.Fatalf("status = %q", got.Status)
	}
}

func TestControlEndpoints(t *testing.T) {
	fc := &fakeCoordinator{}
	s := newTestServer(t, Deps{Coordinator: fc})

	if rec := do(s, http.MethodPost, "/api/connection/stop"); rec.Code != http.StatusOK || !fc.stopped {
		t.Fatalf("stop: code=%d stopped=%v", rec.Code, fc.stopped)
	}
	if rec := do(s, http.MethodPost, "/api/connection/reconnect"); rec.Code != http.StatusOK || !fc.restarted {
		t.Fatalf("reconnect: code=%d restarted=%v", rec.Code, fc.restarted)
	}
	if rec := do(s, http.MethodPost, "/api/connection/start"); rec.Code != http.StatusOK || !fc.started {
		t.Fatalf("start: code=%d started=%v", rec.Code, fc.started)
	}
}

func TestNextMessageEndpoint(t *testing.T) {
	queue := connector.NewMessageQueue()
	queue.Enqueue(ilink.TextMessage{Kind: "text", PeerID: "peer", ContextToken: "ctx", Text: "hello"})
	s := newTestServer(t, Deps{Coordinator: &fakeCoordinator{}, Messages: queue})
	rec := do(s, http.MethodGet, "/api/messages/next")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got ilink.TextMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Kind != "text" || got.PeerID != "peer" || got.Text != "hello" {
		t.Fatalf("message = %+v", got)
	}
}
