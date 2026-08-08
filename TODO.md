# 后端实施清单

## 0. 设计前置

- ✓ 确认桌面 GUI 技术栈与 Gin 服务生命周期。依赖：无。
- ✓ 确认模型 SSE 请求与响应契约：URL、鉴权、请求 JSON、事件格式、结束标识及失败语义。依赖：无。
- ✓ 编写 iLink HTTP 协议记录，固定已验证的端点、请求头、状态值和文本消息结构。依赖：无。

## 1. 项目骨架

- ✓ 初始化 Go module、Gin 服务入口和仅本机监听配置。依赖：0.1。（`cmd/clawbot/main.go`、`internal/httpapi/server.go`；`config.host` 强制 loopback）
- ✓ 定义目录与配置加载：`cmd/`、`internal/ilink/`、`internal/model/`、`internal/app/`、`internal/store/`、`internal/httpapi/`。依赖：1.1。（`internal/app/config.go` 加载并校验）
- ✓ 创建非敏感配置模板，预留 iLink Base URL、App ID、客户端版本、模型 URL、模型名和请求超时。依赖：1.2。（`config.example.json`；真实 `config.json` 已加入 `.gitignore`）
- ✓ 建立结构化日志与敏感字段脱敏规则。依赖：1.2。（`internal/app/logging.go`：slog JSON + 敏感 key 脱敏）

> 验证（2026-08-08）：`gofmt -l .` 无输出、`go vet ./...` 通过、`go build ./...` 通过、`go test ./...` 通过；启动后 `GET /healthz` 返回 `{"status":"ok"}`，未知路由 404，日志为脱敏 JSON。

## 2. 本地凭据与状态

- ✓ 定义账号绑定模型：account ID、Weixin user ID、Base URL、bot token、同步 cursor、更新时间和连接状态。依赖：1.2。
- ✓ 实现本机凭据存储与权限限制，禁止凭据进入配置和日志。依赖：2.1。
- ✓ 实现会话上下文存储，按 `account_id + peer_id` 保存 iLink `context_token`。依赖：2.2。
- ✓ 为凭据、cursor 与 context token 存储编写单元测试。依赖：2.2、2.3。

## 3. iLink HTTP 客户端

- ✓ 实现公共 HTTP 客户端：鉴权头、iLink App 头、随机 `X-WECHAT-UIN`、超时、错误解析和脱敏日志。依赖：1.3、1.4。
- ✓ 实现二维码创建：`get_bot_qrcode`，返回二维码内容与展示 URL。依赖：3.1。
- ✓ 实现二维码状态原语 `PollQRCodeStatus`：返回全部状态值，支持 IDC 重定向（`baseURL` 入参 + `RedirectHost` 回参）、过期与验证码状态。依赖：3.1。（长轮询循环属编排层 §5）
- ✓ 在扫码确认后持久化 `bot_token`、`ilink_bot_id`、`baseurl`、`ilink_user_id`。依赖：2.2、3.3。（`connector.NewBinding(*ilink.QRCodeStatus)→store.Binding`，保持 ilink 无状态；§6 登录处理器调用它 + `SaveBinding`）
- ✓ 实现 `NotifyStart` 和 `NotifyStop`：按账号 `baseURL` 与 token 发送 `base_info` 请求。依赖：3.1。（`internal/ilink/lifecycle.go`）
- ✓ 实现 `GetUpdates` 长轮询：cursor 收发、`longpolling_timeout_ms`、`errcode=-14` 映射为 `ErrSessionExpired`、仅保留文本消息。依赖：3.1。（`internal/ilink/messages.go`；持久化 cursor 与退避重试归入 §5）
- ✓ 实现纯文本 `SendMessage`：携带 `to_user_id` 与对应 `context_token`，校验 `ret==0`。依赖：3.1。（`internal/ilink/send.go`）
- ✓ 为 iLink 客户端创建 `httptest` 覆盖：请求头、base_info、cursor、会话过期、文本发送与失败。依赖：3.2 至 3.7。（`lifecycle_test.go`、`messages_test.go`、`send_test.go`）

> 验证（2026-08-08）：`gofmt -l .` 无输出、`go vet ./...` 通过、`go test ./...` 通过（`internal/ilink` 覆盖 QR 创建/轮询、notifystart/stop、getupdates 过滤与会话过期、sendmessage 成功/失败/校验）。
> 备注：字段名（`message_list`、`from_user_id`、`get_updates_buf`、`ret` 等）依据协议文档与 sendmessage 结构推断，需在真实 iLink 账号回归时核对；如有出入只需在 `messages.go`/`send.go` 的 JSON tag 处集中修改。

## 4. 模型 SSE 桥接

- ✓ 实现独立的 OpenAI 兼容 SSE 解析器，仅处理流事件和文本拼接，不依赖配置或 HTTP 客户端。依赖：0.2。
- ✓ 实现模型 SSE 客户端 `Complete`：请求构造、`Bearer`/`Accept` 头、首包超时（`time.AfterFunc`+ctx cancel，不限制流式正文时长）、复用 `sse.ParseChatCompletion`、取消传播、非 2xx 失败；未完成流不返回文本。依赖：0.2、1.3。（`internal/model/client.go`、`complete.go`；API Key 按次传入不落盘）
- ✓ 定义微信文本消息到模型请求的适配模型 `Prompt`：保留 account/peer/context token/message ID，`buildRequest` 仅发送 system+user 文本，元数据不外泄。依赖：4.1、3.6。（`internal/model/prompt.go`）
- ✓ 定义回复策略 `SplitReply`：完整缓冲后按 `MaxReplyRunes` rune 安全分段，优先在换行/空格断开，空回复返回 nil。依赖：0.2、3.7、4.1。（`internal/model/reply.go`）
- ✓ 为客户端流式成功、非 2xx、取消传播、首包超时，及 prompt 构造、分段策略编写单元测试。依赖：4.1。（`complete_test.go`、`prompt_test.go`、`reply_test.go`）

> 验证（2026-08-08）：`gofmt -l .` 无输出、`go vet ./...` 通过、`go test ./...` 通过（`internal/model` 覆盖流式拼接、HTTP 失败、取消、首包超时、元数据不外泄、rune 安全分段）。
> 备注：系统提示词由 `model.system_prompt` 配置规定 AI 角色，为空时回退到 `defaultSystemPrompt`（`internal/model/client.go`）。模型 API Key 由 `model.api_key` 配置提供，连接器按次注入。

## 5. 消息编排与连接恢复

- ✓ 建立单账号连接管理器 `Connector`：`Start` 恢复已保存绑定→`NotifyStart`→进入长轮询；`Stop`/`Restart`/`Status`。依赖：2.2、3.5、3.6。（`internal/connector/connector.go`；用窄接口 `ILinkClient`/`ModelClient` 便于单测）
- ✓ 过滤非文本消息（由 `GetUpdates` 保证）；文本消息保存 context token 后构造 `model.Prompt` 转发模型 SSE，元数据不外泄。依赖：2.3、3.6、4.2。（`internal/connector/loop.go` handleMessage）
- ✓ 模型回复经 `SplitReply` 缓冲分段后逐块 `SendMessage`，单块发送失败记日志并继续、不中断循环。依赖：3.7、4.3。（`loop.go` handleMessage）
- ✓ 连接状态机（connecting/connected/failed/disconnected）持久化、指数退避（1s×2，上限 60s，成功清零）、可控重连（`Restart`）；`Stop` 与退出用独立 ctx best-effort `NotifyStop`。依赖：5.1 至 5.3。（`loop.go` run/connectAndPoll、`connector.go` Stop）
- ✓ 为完整消息链路编写 mock 集成测试：收发链路、context token 保存、cursor 推进、会话过期置 failed 退出、无绑定拒启动、`Stop` 触发 notifystop。依赖：5.1 至 5.4。（`connector_test.go`；fake 客户端实现窄接口）

> 验证（2026-08-08）：`gofmt -l .` 无输出、`go vet ./...` 通过、`go test ./...` 通过（`internal/connector` 覆盖完整收发链路、状态机、会话过期、绑定映射）。`-race` 因本机无 cgo 未跑；共享字段用 mutex 保护、run goroutine 独占 binding 副本、状态读写走 store 内部锁，设计上无竞争。
> 备注：3.4（扫码确认后持久化绑定）由 `connector.NewBinding(*ilink.QRCodeStatus)→store.Binding` 关闭；§6 登录处理器调用它 + `SaveBinding` 落地。`main.go` 接线与启动自动恢复已在 §6 完成；`TextMessage` 暂无 MessageID，`Prompt.MessageID` 留空。

## 6. Gin 本地 API 与桌面接入

- ✓ 实现本地状态 API `GET /api/status`：返回 `store.PublicStatus`（连接状态/账号/最近错误），不含 token。依赖：5.4。（`internal/httpapi/handlers.go`）
- ✓ 实现本地扫码 API `POST /api/scan`（创建二维码，只回 image_url，qrcode 值留服务端）+ `GET /api/scan/status`（轮询；redirect 更新会话 baseURL；confirmed 落库并自动起连接）。依赖：3.2、3.3。（`internal/httpapi/login.go`）
- ✓ 实现本地控制 API `POST /api/connection/{start,stop,reconnect}`：委托 Connector。依赖：5.4。（`handlers.go`）
- ⏸ 将桌面 GUI 接入本地 API，展示二维码、连接状态和可恢复错误。依赖：0.1、6.1 至 6.3。（**缓做**：仓库无前端、需 Node/Wails 工具链，单独一步）

> 验证（2026-07-21）：`gofmt -l .` 无输出、`go vet ./...` 通过、`go test ./...` 通过（`internal/httpapi` 覆盖 status 不泄 token、控制端点委托、扫码不泄 qrcode 值、confirmed 落库并自动起连接、无会话拒轮询）。
> 接线：`main.go` 已串起 store/ilink/model/connector 并接入 API；**启动自动恢复**已保存绑定。iLink 客户端用无全局超时的 `&http.Client{}`（long-poll 需要），connector 每次 `GetUpdates` 由 `pollTimeout=120s` 兜底，扫码/发送等短请求由 handler 的 `loginRequestTimeout=20s` 限时。store 根目录默认 `os.UserConfigDir()/clawbot`，可用 `-data` 覆盖。退出时 `connector.Stop()` 触发 best-effort notifystop。
> 缓做：§6.4 Wails GUI；verify_code 提交回填（本轮只透传 `need_verifycode` 状态）。

## 7. 验证与交付

- ✓ 执行 `gofmt ./...`、`go vet ./...`、`go test ./...`，修复全部失败项。依赖：6.4（后端部分已验证；GUI 仍暂缓）。
- [ ] 在测试 iLink 账号上验证：扫码、重启恢复、文本收发、模型 SSE 失败和退出上报。依赖：7.1。
- [ ] 编写本地运行、凭据位置、故障处理和数据清理说明。依赖：7.2。
- ✓ 已核对官方 tencent/openclaw-weixin 图片、语音字段、CDN 下载解密和媒体上传流程。
- [ ] 扩展 getupdates 消息模型，保留图片和语音 item_list，同时保持文本兼容。
- [ ] 实现 CDN 媒体下载、AES-128-ECB 解密和本地媒体保存。
- [ ] 实现语音 SILK 转 WAV；转码不可用时保留原始 SILK 并标记 MIME。
- [ ] 将媒体路径和 MIME 传入消息编排层，定义模型适配边界。
- [ ] 补充图片和语音协议单元测试及集成测试。
- ✓ 已明确媒体功能边界：网关只接收并输出微信消息，不调用模型，不依赖 SSE。
- [ ] 设计并实现统一消息事件 API（文本、图片、语音元数据和本地媒体路径）。
- ✓ 已实现统一有序入站消息队列和 `GET /api/messages/next` 长轮询接口；连接器不再调用模型。
- ✓ 文本、图片、语音消息均可进入队列，消息 JSON 带 `type` 字段并保持接收顺序。
- [ ] 为图片和语音增加 CDN 下载、AES-128-ECB 解密及本地文件落盘。
