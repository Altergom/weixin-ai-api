# 微信消息接口（设计草案）

统一入站消息按接收顺序进入内存队列，通过公共 Handler 提供长轮询读取接口。

```http
GET /api/v1/wechat/messages/next?session_id=<session_id>
```

响应使用 `type` 区分 `text`、`image` 和 `voice`。模型、业务处理和媒体识别不属于本库职责。
