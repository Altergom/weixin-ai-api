# 微信 ClawBot 启动与使用说明

面向在本机运行后端服务的用户。第一阶段：单账号、私聊、纯文本，通过 iLink HTTP 长轮询收发，模型走 OpenAI 兼容 SSE。桌面 GUI 尚未接入，本文用命令行和 HTTP 接口演示完整流程。

## 1. 前置条件

- Go 1.25 或更新版本。
- 一个微信账号用于扫码登录。
- 一个 OpenAI 兼容的模型服务地址、模型名和 API Key。
- 仅支持本机运行：服务只监听 `127.0.0.1`，不对外暴露。

## 2. 准备配置

配置文件只存**非敏感参数**，从模板复制一份：

```bash
cd backend
cp config.example.json config.json
```

编辑 `config.json`：

```json
{
  "server":  { "host": "127.0.0.1", "port": 8722 },
  "ilink":   {
    "base_url": "https://ilinkai.weixin.qq.com",
    "app_id": "你的 app id",
    "client_version": 65547,
    "bot_agent": "你的 bot agent",
    "request_timeout_ms": 15000
  },
  "model":   {
    "base_url": "https://api.openai.com",
    "name": "gpt-4o-mini",
    "system_prompt": "你是一个友好、简洁的微信助手，用中文回答用户的问题。",
    "api_key": "sk-...",
    "request_timeout_ms": 60000
  }
}
```

字段说明：
- `server.host` 必须是回环地址（`127.0.0.1` / `localhost` / `::1`），否则启动即报错。
- `ilink.base_url` 为 iLink 服务根地址；`app_id`、`client_version`、`bot_agent` 由 iLink 侧提供。
- `model.base_url` **不要**带 `/v1/chat/completions`，程序自动拼接。
- `model.system_prompt` 规定 AI 扮演的角色，留空则用通用默认值。
- `model.api_key` 是敏感信息：不会写入日志，也不会通过任何接口返回给前端。

## 3. 启动服务

```bash
cd backend
go run ./cmd/clawbot --config config.json
```

可选参数：
- `--config`：配置文件路径，默认 `config.json`。
- `--data`：本机凭据/状态目录，默认 `os.UserConfigDir()/clawbot`（见第 6 节）。

启动后日志出现 `local api listening addr=127.0.0.1:8722` 即就绪。

**自动恢复**：若该数据目录里已有登录绑定，启动时会自动 `notifystart` 并进入长轮询，无需再次扫码。首次运行没有绑定时，日志提示 `no bound account; waiting for scan`，按第 4 节扫码。

## 4. 扫码登录

以下用 `curl` 演示，端口按你的配置替换。

1）创建二维码，拿到图片地址：

```bash
curl -s -X POST http://127.0.0.1:8722/api/scan
# {"image_url":"https://.../qr.png"}
```

在浏览器打开 `image_url`，用微信扫码。二维码原始值只留在服务端，不会返回。

2）轮询扫码状态（每 1–2 秒一次）：

```bash
curl -s http://127.0.0.1:8722/api/scan/status
# {"status":"wait"}      等待扫码
# {"status":"scaned"}    已扫码，待确认
# {"status":"confirmed"} 已确认
```

看到 `confirmed` 表示登录成功：绑定已落盘，连接器**自动启动**并开始收发消息。此后重启进程会自动恢复，不必重复本步。

其他可能的状态：
- `scaned_but_redirect`：iLink 要求切换到就近机房，服务端已自动更新地址，继续轮询即可。
- `need_verifycode`：需要验证码。本阶段仅透传该状态，暂不支持在接口内提交验证码。
- `expired`：二维码过期，重新执行第 1 步。

## 5. 查看状态与控制连接

查看连接状态（不含任何 token）：

```bash
curl -s http://127.0.0.1:8722/api/status
```

返回字段：

```json
{
  "account_id": "bot@im.bot",
  "weixin_user_id": "user@im.wechat",
  "status": "connected",
  "last_error": "",
  "connected_at": "2026-07-21T09:00:00Z",
  "updated_at": "2026-07-21T09:00:05Z"
}
```

`status` 取值：
- `disconnected`：未连接（无绑定或已停止）。
- `connecting`：正在建立连接。
- `connected`：已连接，正在长轮询收发。
- `failed`：连接失败，详情见 `last_error`；会按退避自动重试，会话过期除外。

手动控制连接：

```bash
curl -s -X POST http://127.0.0.1:8722/api/connection/stop        # 停止
curl -s -X POST http://127.0.0.1:8722/api/connection/start       # 启动（需已有绑定）
curl -s -X POST http://127.0.0.1:8722/api/connection/reconnect   # 重连
```

## 6. 消息收发验证

连接状态为 `connected` 后，用**另一个微信账号**给已登录的机器人账号发一条文字私聊消息。链路为：

```
微信 → iLink getupdates → 连接器 → 模型 SSE → 连接器 → iLink sendmessage → 微信
```

正常情况下你会在微信侧收到模型生成的回复。较长的回复会自动按长度上限切分成多条发送。

注意：第一阶段**不保留对话历史**，每条消息独立请求模型，AI 不会记住上一句。

## 7. 凭据与数据位置

运行时数据存放在数据目录（`--data` 覆盖，默认 `os.UserConfigDir()/clawbot`）。各平台默认位置：
- Windows：`%AppData%\clawbot`（即 `C:\Users\<你>\AppData\Roaming\clawbot`）
- macOS：`~/Library/Application Support/clawbot`
- Linux：`~/.config/clawbot`

目录内文件：
- `binding.json`：账号绑定，**含 `bot_token`**、baseurl、同步游标等。属敏感文件，权限 `0600`，目录权限 `0700`。
- `context_tokens.json`：各联系人的最新会话上下文标识。

安全须知：
- 这些文件包含敏感凭据，**切勿**提交 Git、上传或分享。
- 模型 API Key 只在 `config.json` 里，同样不要提交（模板 `config.example.json` 才是可提交的）。
- 日志对 `bot_token`、`api_key`、消息正文等字段做了脱敏，但请仍避免把日志随意外发。

## 8. 故障处理

- **启动报 `server.host must be loopback`**：`server.host` 只能填回环地址。
- **启动报 `model.base_url is required` / `ilink.app_id is required`**：对应必填项为空，补全 `config.json`。
- **`/api/status` 一直 `failed`，`last_error` 提示 session expired**：iLink 会话已失效，需重新扫码（第 4 节）。这种情况不会自动重试。
- **`failed` 但 `last_error` 是网络类错误**：连接器会按 1s→2s→…→最长 60s 的退避自动重连，网络恢复后自愈；也可手动 `reconnect`。
- **微信收不到回复**：先看 `/api/status` 是否 `connected`；再确认 `model.api_key`、`base_url`、`name` 正确，模型服务可达。模型调用失败只记日志、不影响长轮询，单条失败会被跳过。
- **回复被切成多条**：正常行为，长文本按 iLink 文本长度上限分段发送。
- **`/api/scan/status` 返回 `no scan in progress`**：尚未调用 `POST /api/scan`，或上次已 `confirmed` 清除了会话，重新创建二维码即可。

## 9. 退出与数据清理

- **正常退出**：在运行终端按 `Ctrl+C`。服务会发送 best-effort `notifystop` 通知 iLink 停止，再退出。
- **注销 / 更换账号**：停止服务后删除数据目录里的 `binding.json`（可连同 `context_tokens.json` 一起删），再重新启动扫码登录。
- **彻底清理**：停止服务后删除整个数据目录，并删除或清空 `config.json` 中的 `model.api_key`。

```bash
# 示例（Windows，使用默认数据目录）
rm "$AppData/clawbot/binding.json" "$AppData/clawbot/context_tokens.json"
```

## 10. 开发者自检

改动后在 `backend/` 下运行：

```bash
gofmt -l .        # 无输出即格式正确
go vet ./...
go test ./...
```

若本机装有 C 编译器，可加竞态检测：`CGO_ENABLED=1 go test -race ./...`。
