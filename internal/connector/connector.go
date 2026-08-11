package connector

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

const (
	minBackoff       = 1 * time.Second
	maxBackoff       = 60 * time.Second
	notifyStopBudget = 5 * time.Second
	// pollTimeout 限制一次 getupdates 调用时长，必须大于服务端的
	// longpolling_timeout_ms，避免正常长轮询被提前截断。
	pollTimeout = 120 * time.Second
)

// ILinkClient 是连接器所需的 iLink 客户端子集，收窄接口后无需真实服务即可测试消息循环。
type ILinkClient interface {
	NotifyStart(ctx context.Context, baseURL, token string) error
	NotifyStop(ctx context.Context, baseURL, token string) error
	GetUpdates(ctx context.Context, baseURL, token, cursor string) (*ilink.UpdatesResult, error)
	SendMessage(ctx context.Context, baseURL, token, peerID, contextToken, text string) error
}

// MediaClient 是可选的媒体能力接口。文本连接器无需实现这些方法。
type MediaClient interface {
	UploadMedia(context.Context, string, string, string, string, ilink.MediaType, []byte) (ilink.UploadedMedia, error)
	SendImageMessage(context.Context, string, string, string, string, string, ilink.UploadedMedia) error
	SendVoiceMessage(context.Context, string, string, string, string, string, ilink.UploadedMedia, int, int) error
}

// Connector 把 iLink 长轮询与模型连接起来。第一阶段一个 Connector 只运行一个账号，
// Start、Stop、Status 可以并发调用。
type Connector struct {
	ilink      ILinkClient
	store      *store.FileStore
	queue      *MessageQueue
	log        *slog.Logger
	cdnBaseURL string

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// New 创建连接器。模型 API Key 按项目约定从配置依赖中取得，按次传入且绝不记录。
func New(ilinkClient ILinkClient, fileStore *store.FileStore, queue *MessageQueue, logger *slog.Logger) *Connector {
	return NewWithCDN(ilinkClient, fileStore, queue, logger, ilink.DefaultCDNBaseURL)
}

// NewWithCDN 创建连接器并指定微信媒体 CDN 地址。
func NewWithCDN(ilinkClient ILinkClient, fileStore *store.FileStore, queue *MessageQueue, logger *slog.Logger, cdnBaseURL string) *Connector {
	if logger == nil {
		logger = slog.Default()
	}
	return &Connector{
		ilink:      ilinkClient,
		store:      fileStore,
		queue:      queue,
		log:        logger,
		cdnBaseURL: strings.TrimRight(strings.TrimSpace(cdnBaseURL), "/"),
	}
}

// Start 恢复已保存的绑定，并在后台启动轮询循环。
// 没有绑定或连接器已经运行时返回错误。
func (c *Connector) Start(ctx context.Context) error {
	binding, err := c.store.LoadBinding()
	if err != nil {
		return err
	}
	if binding == nil {
		return errors.New("no iLink account is bound; scan a QR code first")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		return errors.New("connector already running")
	}
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	c.cancel = cancel
	c.done = make(chan struct{})
	go c.run(runCtx, *binding)
	return nil
}

// Stop 取消循环，使用新上下文尽力调用 notifystop，并等待循环退出。
// 未运行时也可以安全调用。
func (c *Connector) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	done := c.done
	c.cancel = nil
	c.done = nil
	c.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done

	if binding, err := c.store.LoadBinding(); err == nil && binding != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), notifyStopBudget)
		defer stopCancel()
		if err := c.ilink.NotifyStop(stopCtx, binding.BaseURL, binding.BotToken); err != nil {
			c.log.Warn("notifystop failed", "err", err)
		}
		c.persistStatus(*binding, store.ConnectionStatusDisconnected, "")
	}
}

// Restart 停止当前循环并启动新的循环。
func (c *Connector) Restart(ctx context.Context) error {
	c.Stop()
	return c.Start(ctx)
}

// Status 返回供本地 GUI 使用的非敏感连接状态。
func (c *Connector) Status() (store.PublicStatus, error) {
	binding, err := c.store.LoadBinding()
	if err != nil {
		return store.PublicStatus{}, err
	}
	if binding == nil {
		return store.PublicStatus{Status: store.ConnectionStatusDisconnected}, nil
	}
	return binding.Public(), nil
}

// NextMessage 按接收顺序返回下一条微信入站消息。
func (c *Connector) NextMessage(ctx context.Context) (ilink.TextMessage, error) {
	if c.queue == nil {
		return ilink.TextMessage{}, errors.New("message queue is unavailable")
	}
	return c.queue.Next(ctx)
}

// SendMessage 向当前已绑定的微信用户发送文本回复。
func (c *Connector) SendMessage(ctx context.Context, peerID, contextToken, text string) error {
	binding, err := c.store.LoadBinding()
	if err != nil {
		return err
	}
	if binding == nil {
		return errors.New("no iLink account is bound")
	}
	if strings.TrimSpace(contextToken) == "" {
		contextToken, err = c.store.LoadContextToken(binding.AccountID, peerID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(contextToken) == "" {
			return errors.New("no context token for peer")
		}
	}
	return c.ilink.SendMessage(ctx, binding.BaseURL, binding.BotToken, peerID, contextToken, text)
}

// SendImage 上传并发送图片消息。
func (c *Connector) SendImage(ctx context.Context, peerID, contextToken, caption string, data []byte) error {
	binding, token, err := c.mediaBinding(peerID, contextToken)
	if err != nil {
		return err
	}
	client, ok := c.ilink.(MediaClient)
	if !ok {
		return errors.New("media sending is not supported by iLink client")
	}
	media, err := client.UploadMedia(ctx, binding.BaseURL, binding.BotToken, c.cdnBaseURL, peerID, ilink.MediaTypeImage, data)
	if err != nil {
		return err
	}
	return client.SendImageMessage(ctx, binding.BaseURL, binding.BotToken, peerID, token, caption, media)
}

// SendVoice 上传并发送语音消息。encodeType=6 表示 SILK，7 表示 MP3。
func (c *Connector) SendVoice(ctx context.Context, peerID, contextToken, text string, data []byte, encodeType, playtimeMs int) error {
	binding, token, err := c.mediaBinding(peerID, contextToken)
	if err != nil {
		return err
	}
	client, ok := c.ilink.(MediaClient)
	if !ok {
		return errors.New("media sending is not supported by iLink client")
	}
	media, err := client.UploadMedia(ctx, binding.BaseURL, binding.BotToken, c.cdnBaseURL, peerID, ilink.MediaTypeVoice, data)
	if err != nil {
		return err
	}
	return client.SendVoiceMessage(ctx, binding.BaseURL, binding.BotToken, peerID, token, text, media, encodeType, playtimeMs)
}

func (c *Connector) mediaBinding(peerID, contextToken string) (store.Binding, string, error) {
	if strings.TrimSpace(peerID) == "" {
		return store.Binding{}, "", errors.New("peer ID is required")
	}
	binding, err := c.store.LoadBinding()
	if err != nil {
		return store.Binding{}, "", err
	}
	if binding == nil {
		return store.Binding{}, "", errors.New("no iLink account is bound")
	}
	if strings.TrimSpace(contextToken) == "" {
		contextToken, err = c.store.LoadContextToken(binding.AccountID, peerID)
		if err != nil {
			return store.Binding{}, "", err
		}
		if strings.TrimSpace(contextToken) == "" {
			return store.Binding{}, "", errors.New("no context token for peer")
		}
	}
	return *binding, contextToken, nil
}

// persistStatus 更新可变状态字段，同时保留 token、cursor 和 baseurl。
// 状态仅用于提示，错误只记录日志而不向上层传播。
func (c *Connector) persistStatus(binding store.Binding, status store.ConnectionStatus, lastErr string) {
	binding.Status = status
	binding.LastError = lastErr
	if status == store.ConnectionStatusConnected {
		binding.ConnectedAt = time.Now().UTC()
	}
	if err := c.store.SaveBinding(binding); err != nil {
		c.log.Warn("persist connection status failed", "status", status, "err", err)
	}
}
