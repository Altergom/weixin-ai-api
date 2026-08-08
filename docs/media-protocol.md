# iLink 图片与语音协议记录

本记录根据 `tencent/openclaw-weixin` 官方实现核对。

## 入站消息

`getupdates` 响应顶层字段是 `msgs`，媒体位于每条消息的 `item_list`：

- 图片：`type = 2`，内容在 `image_item`。下载引用在 `image_item.media`，通常包含 `encrypt_query_param` 和 Base64 编码的 `aes_key`。
- 语音：`type = 3`，内容在 `voice_item`。下载引用和 `aes_key` 位于 `voice_item.media`；`voice_item.text` 可能包含语音转文字结果。

官方流程是通过 CDN 引用下载媒体，再使用 AES-128-ECB 解密。语音解密后是 SILK，优先转为 WAV；转码不可用时保留原始 SILK。

## 出站消息

发送媒体前调用 `POST /ilink/bot/getuploadurl`，按 `media_type` 区分图片（1）和语音（4），使用返回参数向 CDN 上传 AES-128-ECB 密文，最后通过 `POST /ilink/bot/sendmessage` 发送结构化 `image_item` 或 `voice_item`。
