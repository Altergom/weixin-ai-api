# 微信 iLink API 库

本仓库是可被其他 Go 项目引用的基础库，不包含可执行入口、HTTP 监听进程或模型调用。

## 使用

```go
gateway, err := ilink.NewGateway(ilink.Config{
	DataDir:       "./data/weixin",
	ILinkBaseURL:  "https://ilinkai.weixin.qq.com",
	ILinkAppID:    "bot",
	ClientVersion: 65547,
	BotAgent:      "my-app/1.0.0",
}, nil)
if err != nil {
	return err
}

mux := http.NewServeMux()
mux.Handle("/", gateway.Handler())
return http.ListenAndServe("127.0.0.1:8722", mux)
```

宿主项目负责启动 HTTP 服务、业务处理、模型调用和微信回复策略。网关负责二维码登录、iLink 长轮询和统一入站消息队列。

## 接口

- `GET /healthz`
- `POST /api/scan`
- `GET /api/scan/status`
- `POST /api/connection/start`
- `POST /api/connection/stop`
- `POST /api/connection/reconnect`
- `GET /api/messages/next`

完整请求和响应格式见根目录 `API.md`。
