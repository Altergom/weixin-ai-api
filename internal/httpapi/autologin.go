package httpapi

import (
	"context"
	"time"
)

const autoLoginPollInterval = 2 * time.Second

// AutoLogin 在没有 GUI 时推进二维码登录：没有绑定账号时创建二维码，
// 将图片 URL 交给 onQR（例如终端渲染器），然后轮询到确认。二维码过期时重新创建。
// 账号确认、已绑定或上下文取消时返回。它可以与共享会话状态的 HTTP 服务并行运行。
func (s *Server) AutoLogin(ctx context.Context, onQR func(imageURL string)) {
	if s.isBound() {
		s.log.Info("account already bound; skipping auto-login")
		return
	}
	for ctx.Err() == nil {
		if err := s.presentQR(ctx, onQR); err != nil {
			s.log.Warn("create login qr failed; retrying", "err", err)
			if !sleepCtx(ctx, autoLoginPollInterval) {
				return
			}
			continue
		}
		if s.pollUntilResolved(ctx) {
			return
		}
	}
}

func (s *Server) presentQR(ctx context.Context, onQR func(string)) error {
	createCtx, cancel := context.WithTimeout(ctx, loginRequestTimeout)
	defer cancel()
	imageURL, err := s.createLogin(createCtx)
	if err != nil {
		return err
	}
	if onQR != nil {
		onQR(imageURL)
	}
	return nil
}

// pollUntilResolved 轮询当前二维码直到确认（返回 true）或过期（返回 false，
// 调用方随后创建新二维码）。上下文取消时也返回 true 以停止循环。
func (s *Server) pollUntilResolved(ctx context.Context) bool {
	for {
		if !sleepCtx(ctx, autoLoginPollInterval) {
			return true
		}
		pollCtx, cancel := context.WithTimeout(ctx, loginRequestTimeout)
		status, err := s.pollLogin(pollCtx)
		cancel()
		if err != nil {
			s.log.Warn("poll login status failed", "err", err)
			continue
		}
		switch status {
		case "confirmed":
			s.log.Info("wechat account confirmed; connection starting")
			return true
		case "expired":
			s.log.Info("login qr expired; regenerating")
			return false
		}
	}
}

func (s *Server) isBound() bool {
	binding, err := s.deps.Store.LoadBinding()
	return err == nil && binding != nil
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
