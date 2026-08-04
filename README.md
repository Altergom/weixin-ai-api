# 微信 iLink 连接库

提供微信 iLink 二维码登录 HTTP 接口，供其他 Go 项目作为依赖使用。本仓库不包含可执行入口、HTTP 监听、模型配置或模型推理。

## 引用

```bash
go get github.com/Altergom/weixin-ai-api
```

从指定 `.env` 文件创建 `http.Handler`，再由宿主项目挂载：

```go
import (
	"net/http"

	ilink "github.com/Altergom/weixin-ai-api"
)

handler, err := ilink.NewHandlerFromEnv(".env")
if err != nil {
	return err
}

mux := http.NewServeMux()
mux.Handle("/api/v1/wechat/", handler)
```

库只读取传入的 `.env` 文件，不读取系统环境变量。文件不存在时使用以下默认值；修改文件后需要由宿主重新创建 Handler，通常即重启宿主服务。

```dotenv
ILINK_BASE_URL=https://ilinkai.weixin.qq.com
ILINK_APP_ID=bot
ILINK_CLIENT_VERSION=1.0.0
```

真实 `.env` 已被 Git 忽略，可从 `.env.example` 创建。

## 接口

### 创建二维码

```http
POST /api/v1/wechat/qrcode
```

响应包含 `session_id`、`qrcode`、`status` 和 `expires_at`。其中 `qrcode` 包含原始二维码值，以及上游提供时的 `qrcode_url` 或 `qrcode_image`。

### 查询连接状态

```http
GET /api/v1/wechat/qrcode/status?session_id=<session_id>
```

状态包括 `wait`、`scaned`、`confirmed` 和 `expired`。确认连接后，`bot_token` 仅保存在 Handler 的内存会话中，不写入日志或 HTTP 响应。

所有错误响应统一为：

```json
{"error":"message"}
```
