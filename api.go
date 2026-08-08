package ilink

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Altergom/weixin-ai-api/internal/app"
	"github.com/Altergom/weixin-ai-api/internal/connector"
	"github.com/Altergom/weixin-ai-api/internal/httpapi"
	protocol "github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

// Message 是网关输出的统一入站消息类型。
type Message = protocol.TextMessage

// Config 是网关库的非敏感运行配置。
type Config struct {
	Host          string
	Port          int
	DataDir       string
	ILinkBaseURL  string
	ILinkAppID    string
	ClientVersion uint32
	BotAgent      string
}

// Gateway 封装消息队列、iLink 连接器和本地 HTTP Handler。
type Gateway struct {
	connector *connector.Connector
	handler   http.Handler
}

// NewGateway 创建可被宿主项目挂载的网关，不启动 HTTP 监听。
func NewGateway(cfg Config, logger *slog.Logger) (*Gateway, error) {
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 8722
	}
	if logger == nil {
		logger = slog.Default()
	}
	fileStore, err := store.NewFileStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	ilinkClient, err := protocol.NewClient(app.ILinkConfig{
		BaseURL: cfg.ILinkBaseURL, AppID: cfg.ILinkAppID,
		ClientVersion: cfg.ClientVersion, BotAgent: cfg.BotAgent,
	}, &http.Client{}, logger)
	if err != nil {
		return nil, err
	}
	queue := connector.NewMessageQueue()
	coordinator := connector.New(ilinkClient, fileStore, queue, logger)
	server := httpapi.NewServer(app.ServerConfig{Host: cfg.Host, Port: cfg.Port}, logger, httpapi.Deps{
		Coordinator: coordinator,
		ILink:       ilinkClient,
		Store:       fileStore,
		Messages:    queue,
	})
	return &Gateway{connector: coordinator, handler: server.Engine()}, nil
}

// Handler 返回宿主项目挂载的 HTTP Handler。
func (g *Gateway) Handler() http.Handler { return g.handler }

// StartConnection 启动已绑定账号的 iLink 长轮询。
func (g *Gateway) StartConnection(ctx context.Context) error { return g.connector.Start(ctx) }

// StopConnection 停止 iLink 长轮询。
func (g *Gateway) StopConnection() { g.connector.Stop() }
