package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Altergom/weixin-ai-api/internal/connector"
	"github.com/Altergom/weixin-ai-api/internal/ilink"
)

// loginSession 保存进行中的扫码会话。qrcode 值属于敏感信息，只留在进程内；
// GUI 只能获得图片 URL 和状态。
type loginSession struct {
	qrcode  string
	baseURL string
}

// errNoScan 表示在创建二维码前就请求了扫码状态。
var errNoScan = errors.New("no scan in progress")

// createLogin 请求新的二维码并保存为当前会话。它返回供展示的图片 URL，
// qrcode 值留在服务内部。
func (s *Server) createLogin(ctx context.Context) (string, error) {
	var localTokens []string
	if binding, err := s.deps.Store.LoadBinding(); err == nil && binding != nil {
		localTokens = append(localTokens, binding.BotToken)
	}
	qr, err := s.deps.ILink.CreateQRCode(ctx, localTokens)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.session = &loginSession{qrcode: qr.Value}
	s.mu.Unlock()
	return qr.ImageURL, nil
}

// pollLogin 推进当前会话一步并返回 iLink 状态字符串。
// confirmed 时持久化绑定并启动连接器；重定向时更新会话 base URL。
// errNoScan 表示没有活动会话。
func (s *Server) pollLogin(ctx context.Context) (string, error) {
	s.mu.Lock()
	session := s.session
	s.mu.Unlock()
	if session == nil {
		return "", errNoScan
	}

	status, err := s.deps.ILink.PollQRCodeStatus(ctx, session.baseURL, session.qrcode, "")
	if err != nil {
		return "", err
	}

	switch status.Status {
	case "scaned_but_redirect":
		s.mu.Lock()
		if s.session != nil && status.RedirectHost != "" {
			s.session.baseURL = "https://" + status.RedirectHost
		}
		s.mu.Unlock()
	case "confirmed":
		if err := s.confirm(ctx, status); err != nil {
			return "", err
		}
	}
	return status.Status, nil
}

func (s *Server) handleScanCreate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), loginRequestTimeout)
	defer cancel()
	imageURL, err := s.createLogin(ctx)
	if err != nil {
		s.fail(w, http.StatusBadGateway, "create qr code failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"image_url": imageURL})
}

func (s *Server) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), loginRequestTimeout)
	defer cancel()
	status, err := s.pollLogin(ctx)
	switch {
	case errors.Is(err, errNoScan):
		s.fail(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		s.fail(w, http.StatusBadGateway, "poll qr status failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

// confirm 持久化新绑定并启动连接器，同时清除会话，避免后续轮询重复确认。
func (s *Server) confirm(ctx context.Context, status *ilink.QRCodeStatus) error {
	binding, err := connector.NewBinding(status)
	if err != nil {
		return err
	}
	if err := s.deps.Store.SaveBinding(binding); err != nil {
		return err
	}
	s.mu.Lock()
	s.session = nil
	s.mu.Unlock()
	if err := s.deps.Coordinator.Start(ctx); err != nil {
		s.log.Warn("auto-start after scan failed", "err", err)
	}
	return nil
}
