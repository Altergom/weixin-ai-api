# iLink HTTP 协议记录

## 服务地址

默认服务地址为 `https://ilinkai.weixin.qq.com`。二维码确认响应中的 `baseurl` 是该账号后续请求的优先地址。

二维码状态响应为 `scaned_but_redirect` 且提供 `redirect_host` 时，仅后续扫码状态轮询改用 `https://{redirect_host}`。

## 公共请求头

带认证的 POST 请求使用：

```http
Content-Type: application/json
AuthorizationType: ilink_bot_token
Authorization: Bearer {bot_token}
X-WECHAT-UIN: {base64-encoded random uint32 decimal string}
iLink-App-Id: {configured app id}
iLink-App-ClientVersion: {configured uint32 client version}
```

每个业务请求都带有：

```json
{
  "base_info": {
    "channel_version": "{client version}",
    "bot_agent": "{configured bot agent}"
  }
}
```

`bot_token`、Authorization 和二维码原始值属于敏感信息。

## 登录

### 创建二维码

```http
POST /ilink/bot/get_bot_qrcode?bot_type=3
```

请求正文：

```json
{"local_token_list": []}
```

响应使用 `qrcode` 作为状态轮询参数，使用 `qrcode_img_content` 供 GUI 展示。

### 轮询扫码状态

```http
GET /ilink/bot/get_qrcode_status?qrcode={qrcode}
```

状态包括：`wait`、`scaned`、`confirmed`、`expired`、`scaned_but_redirect`、`need_verifycode`、`verify_code_blocked`、`binded_redirect`。

`confirmed` 响应必须包含 `bot_token`、`ilink_bot_id`、`baseurl` 和 `ilink_user_id` 才能建立绑定。

## 连接生命周期

```http
POST /ilink/bot/notifystart
POST /ilink/bot/notifystop
```

二者请求正文为 `{"base_info": {...}}`。`notifystart` 在长轮询前调用；`notifystop` 在连接停止或应用退出时尽力调用。

## 消息

### 获取更新

```http
POST /ilink/bot/getupdates
```

请求正文：

```json
{
  "get_updates_buf": "{saved cursor or empty string}",
  "base_info": {}
}
```

响应中的 `get_updates_buf` 必须持久化，并作为下一次请求的 cursor。`longpolling_timeout_ms` 是服务端建议的下次等待时长。`errcode = -14` 表示会话已过期。

只接收 `message_type = 1`、`item_list[].type = 1` 的文本消息。回复时保存并携带每个发送者最新的 `context_token`。

### 发送文本

```http
POST /ilink/bot/sendmessage
```

```json
{
  "msg": {
    "to_user_id": "{peer id}",
    "context_token": "{saved context token}",
    "item_list": [
      {
        "type": 1,
        "text_item": {"text": "{reply}"}
      }
    ]
  },
  "base_info": {}
}
```

响应 `ret = 0` 代表成功。非零 `ret` 或 HTTP 非 2xx 是发送失败。
