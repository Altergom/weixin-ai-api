# 模型 SSE 契约

## 范围

第一阶段使用 OpenAI Chat Completions 兼容接口。`model.base_url` 为服务根地址，客户端请求路径固定为 `/v1/chat/completions`。

## 请求

```http
POST {model.base_url}/v1/chat/completions
Authorization: Bearer {model.api_key}
Content-Type: application/json
Accept: text/event-stream
```

```json
{
  "model": "{model.name}",
  "stream": true,
  "messages": [
    {"role": "system", "content": "{model.system_prompt}"},
    {"role": "user", "content": "{wechat text}"}
  ]
}
```

`model.base_url` 不得包含 `/v1/chat/completions`，避免重复拼接。`model.system_prompt` 规定 AI 扮演的角色，为空时回退到通用默认值。API Key 从 `model.api_key` 配置读取，属敏感信息，禁止返回给 GUI 和写入日志（`app/logging.go` 已按 `api_key` 键脱敏）。

## 入站上下文

模型请求由 Gin 根据 iLink 文本消息构造，并保留以下内部元数据，不发送给第三方模型：

- iLink account ID
- WeChat peer ID
- iLink context token
- iLink message ID

第一阶段不维护模型侧历史记忆；每条微信文本独立请求模型。

## SSE 响应

响应的每行 `data:` 为一个 JSON 数据块：

```json
{"choices":[{"delta":{"content":"你好"},"finish_reason":null}]}
```

正文由 `choices[0].delta.content` 按到达顺序拼接。`data: [DONE]` 是正常结束标志。空 delta 可忽略。

## 失败语义

- 连接、首包、读取和解析失败均视为本次模型调用失败。
- 已收到但未完成的文本不得发送给微信，防止重复或半截回复。
- 成功完成后，协调器将完整文本按 iLink 文本长度限制切分后发送。
- 用户消息、API Key、Authorization 和完整模型响应不得写入日志。
