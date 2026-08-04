# 微信 iLink 基础服务

将 `openclaw-weixin` 使用的 iLink 协议抽离为可被其他程序调用的 HTTP 基础服务。服务不承载模型推理，只保存外接模型所需的连接配置，并负责微信二维码登录会话。

## 启动

```bash
cd weixin-ilink-service
go run ./cmd/server
```

可通过 `LISTEN_ADDR`、`ILINK_BASE_URL`、`ILINK_APP_ID`、`ILINK_CLIENT_VERSION` 配置监听地址和 iLink 请求头。

## 接口

设置模型（key 只保存在内存中）：

```bash
curl -X PUT http://localhost:8080/api/v1/model \
  -H 'content-type: application/json' \
  -d '{"model":"deepseek-chat","baseurl":"https://api.deepseek.com","key":"sk-..."}'
```

`GET /api/v1/model` 返回模型名、baseurl 和脱敏 key。

创建微信二维码：`POST /api/v1/wechat/qrcode`，返回 `session_id`、`qrcode`、`qrcode_url` 或 `qrcode_image`。轮询 `GET /api/v1/wechat/qrcode/status?session_id=...`，状态包括 `wait`、`scaned`、`confirmed`、`expired` 等 iLink 原始状态。

所有错误响应统一为 `{ "error": "..." }`。服务不会记录请求体、模型 key、bot token 或二维码凭据。
