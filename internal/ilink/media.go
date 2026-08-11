package ilink

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// 默认微信 CDN 地址。登录接口返回 upload_full_url 时优先使用返回地址。
const DefaultCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"

const (
	MediaTypeImage MediaType = 1
	MediaTypeVoice MediaType = 4
)

type MediaType int

// UploadedMedia 是上传到微信 CDN 后用于组装消息的媒体信息。
type UploadedMedia struct {
	FileKey           string
	EncryptQueryParam string
	AESKey            string
	PlaintextSize     int
	CiphertextSize    int
}

type uploadURLResponse struct {
	Ret           int    `json:"ret"`
	ErrMsg        string `json:"errmsg"`
	UploadParam   string `json:"upload_param"`
	UploadFullURL string `json:"upload_full_url"`
}

// UploadMedia 完成 getuploadurl、AES-128-ECB 加密和 CDN 上传。
func (c *Client) UploadMedia(ctx context.Context, baseURL, token, cdnBaseURL, peerID string, mediaType MediaType, data []byte) (UploadedMedia, error) {
	if strings.TrimSpace(peerID) == "" {
		return UploadedMedia{}, fmt.Errorf("ilink upload media: peer ID is required")
	}
	if len(data) == 0 {
		return UploadedMedia{}, fmt.Errorf("ilink upload media: data is empty")
	}
	if mediaType != MediaTypeImage && mediaType != MediaTypeVoice {
		return UploadedMedia{}, fmt.Errorf("ilink upload media: unsupported media type %d", mediaType)
	}

	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return UploadedMedia{}, fmt.Errorf("generate media key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return UploadedMedia{}, fmt.Errorf("create media cipher: %w", err)
	}
	ciphertext := pkcs7ECBEncrypt(block, data)
	fileKeyBytes := make([]byte, 16)
	if _, err := rand.Read(fileKeyBytes); err != nil {
		return UploadedMedia{}, fmt.Errorf("generate media file key: %w", err)
	}
	fileKey := hex.EncodeToString(fileKeyBytes)
	hash := md5.Sum(data)
	keyHex := hex.EncodeToString(key)

	target, err := c.resolveBaseURL(baseURL)
	if err != nil {
		return UploadedMedia{}, err
	}
	body := map[string]any{
		"filekey": fileKey, "media_type": int(mediaType), "to_user_id": peerID,
		"rawsize": len(data), "rawfilemd5": hex.EncodeToString(hash[:]),
		"filesize": len(ciphertext), "no_need_thumb": true, "aeskey": keyHex,
		"base_info": c.baseInfo(),
	}
	var upload uploadURLResponse
	if err := c.doJSONAt(ctx, target, http.MethodPost, "ilink/bot/getuploadurl", token, body, &upload); err != nil {
		return UploadedMedia{}, fmt.Errorf("ilink getuploadurl: %w", err)
	}
	if upload.Ret != 0 {
		return UploadedMedia{}, fmt.Errorf("ilink getuploadurl: ret %d: %s", upload.Ret, upload.ErrMsg)
	}
	if strings.TrimSpace(upload.UploadFullURL) == "" && strings.TrimSpace(upload.UploadParam) == "" {
		return UploadedMedia{}, fmt.Errorf("ilink getuploadurl: upload URL is empty")
	}
	cdnURL := strings.TrimSpace(upload.UploadFullURL)
	if cdnURL == "" {
		cdnURL, err = buildCDNUploadURL(cdnBaseURL, upload.UploadParam, fileKey)
		if err != nil {
			return UploadedMedia{}, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cdnURL, bytes.NewReader(ciphertext))
	if err != nil {
		return UploadedMedia{}, fmt.Errorf("create CDN upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return UploadedMedia{}, fmt.Errorf("upload media to CDN: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return UploadedMedia{}, fmt.Errorf("upload media to CDN: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	encryptedParam := strings.TrimSpace(resp.Header.Get("x-encrypted-param"))
	if encryptedParam == "" {
		return UploadedMedia{}, fmt.Errorf("upload media to CDN: x-encrypted-param is missing")
	}
	return UploadedMedia{FileKey: fileKey, EncryptQueryParam: encryptedParam, AESKey: base64.StdEncoding.EncodeToString([]byte(keyHex)), PlaintextSize: len(data), CiphertextSize: len(ciphertext)}, nil
}

func buildCDNUploadURL(raw, uploadParam, fileKey string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = DefaultCDNBaseURL
	}
	u, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid CDN base URL")
	}
	u.Path = path.Join(u.Path, "upload")
	q := u.Query()
	q.Set("encrypted_query_param", uploadParam)
	q.Set("filekey", fileKey)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func pkcs7ECBEncrypt(block cipher.Block, input []byte) []byte {
	pad := aes.BlockSize - len(input)%aes.BlockSize
	data := make([]byte, len(input)+pad)
	copy(data, input)
	for i := len(input); i < len(data); i++ {
		data[i] = byte(pad)
	}
	output := make([]byte, len(data))
	for i := 0; i < len(data); i += aes.BlockSize {
		block.Encrypt(output[i:i+aes.BlockSize], data[i:i+aes.BlockSize])
	}
	return output
}

// SendImageMessage 发送已上传的图片消息。
func (c *Client) SendImageMessage(ctx context.Context, baseURL, token, peerID, contextToken, caption string, media UploadedMedia) error {
	item := map[string]any{"type": 2, "image_item": map[string]any{"media": map[string]any{"encrypt_query_param": media.EncryptQueryParam, "aes_key": media.AESKey, "encrypt_type": 1}, "mid_size": media.CiphertextSize}}
	return c.sendMediaMessage(ctx, baseURL, token, peerID, contextToken, caption, item)
}

// SendVoiceMessage 发送已上传的语音消息。encodeType=6 表示 SILK，7 表示 MP3。
func (c *Client) SendVoiceMessage(ctx context.Context, baseURL, token, peerID, contextToken, text string, media UploadedMedia, encodeType, playtimeMs int) error {
	voice := map[string]any{"media": map[string]any{"encrypt_query_param": media.EncryptQueryParam, "aes_key": media.AESKey, "encrypt_type": 1}, "encode_type": encodeType, "playtime": playtimeMs}
	if strings.TrimSpace(text) != "" {
		voice["text"] = text
	}
	return c.sendMediaMessage(ctx, baseURL, token, peerID, contextToken, "", map[string]any{"type": 3, "voice_item": voice})
}

func (c *Client) sendMediaMessage(ctx context.Context, baseURL, token, peerID, contextToken, caption string, mediaItem map[string]any) error {
	if strings.TrimSpace(peerID) == "" {
		return fmt.Errorf("ilink send media: peer ID is required")
	}
	items := make([]map[string]any, 0, 2)
	if strings.TrimSpace(caption) != "" {
		items = append(items, map[string]any{"type": textItemType, "text_item": map[string]any{"text": caption}})
	}
	items = append(items, mediaItem)
	message := map[string]any{"from_user_id": "", "to_user_id": peerID, "client_id": newClientID(), "message_type": 2, "message_state": 2, "item_list": items}
	if strings.TrimSpace(contextToken) != "" {
		message["context_token"] = contextToken
	}
	body := map[string]any{"msg": message, "base_info": c.baseInfo()}
	var response struct {
		Ret    int    `json:"ret"`
		ErrMsg string `json:"errmsg"`
	}
	target, err := c.resolveBaseURL(baseURL)
	if err != nil {
		return err
	}
	if err := c.doJSONAt(ctx, target, http.MethodPost, "ilink/bot/sendmessage", token, body, &response); err != nil {
		return fmt.Errorf("ilink send media: %w", err)
	}
	if response.Ret != 0 {
		return fmt.Errorf("ilink send media: ret %d: %s", response.Ret, response.ErrMsg)
	}
	return nil
}
