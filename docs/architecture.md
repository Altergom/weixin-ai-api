# 架构决策

## 桌面应用

桌面 GUI 使用 Wails。Wails 负责窗口、应用启动、退出和打包；它不承载业务协议。

Gin 服务在同一进程中启动，只监听 `127.0.0.1`。GUI 通过本地 HTTP API 获取二维码、连接状态和故障信息。第一阶段不使用 WebSocket。

## 消息链路

```text
WeChat -> iLink HTTP getupdates -> Gin message coordinator
    -> model SSE client -> Gin message coordinator
    -> iLink HTTP sendmessage -> WeChat
```

iLink 连接由客户端维护：扫码后保存账号绑定和同步状态，进程启动时自动恢复，启动与停止分别调用 `notifystart`、`notifystop`。

## 第一阶段范围

- 单个微信账号
- 私聊
- 纯文本消息
- iLink HTTP 长轮询
- 模型 SSE 响应

不包含群聊、媒体、多账号、WebSocket 或远程部署。
