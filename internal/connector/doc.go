// connector 包负责消息编排：恢复已保存的 iLink 绑定，运行 getupdates 长轮询，
// 将收到的文本转发给模型 SSE 客户端，再通过 iLink 把完整回复发回去。
// 它管理连接状态、指数退避重连，以及退出时尽力调用 notifystop。
package connector
