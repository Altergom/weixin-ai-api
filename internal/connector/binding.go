package connector

import (
	"fmt"

	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

// NewBinding 将已确认的 iLink 二维码登录结果转换为持久化绑定。
// 它连接登录流程（§6）与存储：扫码确认后，调用方持久化返回的 Binding，
// 才能开始轮询账号消息。
func NewBinding(status *ilink.QRCodeStatus) (store.Binding, error) {
	if status == nil {
		return store.Binding{}, fmt.Errorf("qr code status is required")
	}
	binding := store.Binding{
		AccountID:    status.AccountID,
		WeixinUserID: status.WeixinUserID,
		BaseURL:      status.BaseURL,
		BotToken:     status.BotToken,
		Status:       store.ConnectionStatusDisconnected,
	}
	if err := binding.Validate(); err != nil {
		return store.Binding{}, fmt.Errorf("build binding from qr status: %w", err)
	}
	return binding, nil
}
