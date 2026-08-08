package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/app"
	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

// Coordinator 是 API 所需的连接器接口。使用接口后，控制端点无需真实连接
// 也可以进行测试。
type Coordinator interface {
	Status() (store.PublicStatus, error)
	Start(ctx context.Context) error
	Stop()
	Restart(ctx context.Context) error
}

// ILinkLogin 是扫码端点所需的 iLink 接口。
type ILinkLogin interface {
	CreateQRCode(ctx context.Context, localTokens []string) (*ilink.QRCode, error)
	PollQRCodeStatus(ctx context.Context, baseURL, qrcode, verifyCode string) (*ilink.QRCodeStatus, error)
}

// MessageSource 提供按接收顺序读取微信入站消息的能力。
type MessageSource interface {
	Next(context.Context) (ilink.TextMessage, error)
}

// Deps 是本地 API 委托调用的依赖。
type Deps struct {
	Coordinator Coordinator
	ILink       ILinkLogin
	Store       *store.FileStore
	Messages    MessageSource
}

// Server 封装仅监听回环地址、供桌面 GUI 调用的 Gin HTTP 服务。
type Server struct {
	cfg    app.ServerConfig
	log    *slog.Logger
	engine *http.ServeMux
	http   *http.Server
	deps   Deps

	mu      sync.Mutex
	session *loginSession
}

// NewServer 创建本地 API 服务并注册路由。
func NewServer(cfg app.ServerConfig, log *slog.Logger, deps Deps) *Server {
	engine := http.NewServeMux()
	engine.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	s := &Server{cfg: cfg, log: log, engine: engine, deps: deps}
	s.registerRoutes()
	return s
}

// Engine 暴露路由器，供后续阶段注册 API 路由。
func (s *Server) Engine() http.Handler { return s.engine }

// Start 绑定回环监听地址并持续服务，直到上下文被取消。
func (s *Server) Start(ctx context.Context) error {
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	s.http = &http.Server{Addr: addr, Handler: s.engine}

	errc := make(chan error, 1)
	go func() {
		s.log.Info("local api listening", "addr", addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errc <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.http.Shutdown(shutdownCtx)
	case err := <-errc:
		return err
	}
}
