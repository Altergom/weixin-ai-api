package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Altergom/weixin-ai-api/internal/app"
	"github.com/Altergom/weixin-ai-api/internal/ilink"
	"github.com/Altergom/weixin-ai-api/internal/store"
)

const secretQR = "secret-qr-value"

func fakeILinkServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ilink/bot/get_bot_qrcode", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"qrcode":"` + secretQR + `","qrcode_img_content":"https://img.example.test/qr"}`))
	})
	mux.HandleFunc("/ilink/bot/get_qrcode_status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"confirmed","bot_token":"tok","ilink_bot_id":"bot@im.bot",` +
			`"baseurl":"https://route.example.test","ilink_user_id":"user@im.wechat"}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newScanServer(t *testing.T) (*Server, *store.FileStore, *fakeCoordinator) {
	t.Helper()
	server := fakeILinkServer(t)
	client, err := ilink.NewClient(app.ILinkConfig{
		BaseURL: server.URL, AppID: "app", ClientVersion: 1,
	}, server.Client(), nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	fs, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	fc := &fakeCoordinator{}
	s := newTestServer(t, Deps{Coordinator: fc, ILink: client, Store: fs})
	return s, fs, fc
}

func TestScanCreateHidesQRValue(t *testing.T) {
	s, _, _ := newScanServer(t)
	rec := do(s, http.MethodPost, "/api/scan")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, secretQR) {
		t.Fatalf("scan response leaked qrcode value: %s", body)
	}
	var got struct {
		ImageURL string `json:"image_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ImageURL != "https://img.example.test/qr" {
		t.Fatalf("image_url = %q", got.ImageURL)
	}
}

func TestScanStatusConfirmedPersistsAndStarts(t *testing.T) {
	s, fs, fc := newScanServer(t)
	if rec := do(s, http.MethodPost, "/api/scan"); rec.Code != http.StatusOK {
		t.Fatalf("scan create code = %d", rec.Code)
	}
	rec := do(s, http.MethodGet, "/api/scan/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("scan status code = %d", rec.Code)
	}
	var got struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Status != "confirmed" {
		t.Fatalf("status = %q", got.Status)
	}
	binding, err := fs.LoadBinding()
	if err != nil || binding == nil {
		t.Fatalf("LoadBinding() = %v, %v", binding, err)
	}
	if binding.AccountID != "bot@im.bot" || binding.BotToken != "tok" {
		t.Fatalf("binding = %+v", binding)
	}
	if !fc.started {
		t.Fatal("connector was not auto-started after confirm")
	}
}

func TestScanStatusWithoutSession(t *testing.T) {
	s, _, _ := newScanServer(t)
	if rec := do(s, http.MethodGet, "/api/scan/status"); rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}
