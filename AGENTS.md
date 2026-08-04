# 微信 iLink 基础服务

这是独立的 Go HTTP 服务，不依赖上级贴吧项目。`cmd/server` 只负责启动，`internal/ilink` 负责 iLink 协议和服务状态。

验证命令：

```bash
go fmt ./...
go vet ./...
go test ./...
```

模型 key 只能通过环境变量或请求体进入内存，禁止写入日志、响应和提交记录。
