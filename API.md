# 微信 iLink 网关 API

这是可被宿主 Go 项目挂载的 `http.Handler`，网关本身不启动 HTTP 服务。

```go
gateway, err := ilink.NewGateway(ilink.Config{
    DataDir: "./data/weixin",
    ILinkBaseURL: "https://ilinkai.weixin.qq.com",
    ILinkAppID: "bot",
    ClientVersion: 65547,
}, nil)
```

宿主挂载 `gateway.Handler()` 后，接口路径如下。

## 登录和连接

```http
GET /healthz
POST /api/scan
GET /api/scan/status
POST /api/connection/start
POST /api/connection/stop
POST /api/connection/reconnect
```

## 读取消息

```http
GET /api/messages/next
```

按接收顺序返回一条消息并从内存队列移除。接口最长等待 30 秒：有消息返回 `200`，超时返回 `204`。

文本：

```json
{"type":"text","from_user_id":"user-id","context_token":"opaque-token","text":"你好"}
```

图片和语音使用相同结构，通过 `type` 区分，并在 `media` 中携带 iLink CDN 元数据。媒体识别、模型调用和业务回复由宿主项目负责。
