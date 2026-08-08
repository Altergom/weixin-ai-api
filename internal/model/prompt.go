package model

// Prompt 是一条路由到模型的微信文本消息。路由元数据用于连接器识别会话，
// 不会发送给模型。
type Prompt struct {
	AccountID    string // iLink 账号（Bot）ID
	PeerID       string // 微信发送者 ID
	ContextToken string // 该联系人的 iLink 回复上下文 token
	MessageID    string // iLink 入站消息 ID
	Text         string // 用户文本，唯一发送给模型的消息内容
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
	Messages []chatMessage `json:"messages"`
}

// buildRequest 将 Prompt 转换为 OpenAI 兼容请求体。
// 只有用户文本和系统提示词会离开进程，路由元数据不会外发。
func (c *Client) buildRequest(prompt Prompt) chatRequest {
	return chatRequest{
		Model:  c.modelName,
		Stream: true,
		Messages: []chatMessage{
			{Role: "system", Content: c.systemPrompt},
			{Role: "user", Content: prompt.Text},
		},
	}
}
