package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Altergom/weixin-ai-api/internal/config"
	protocol "github.com/Altergom/weixin-ai-api/internal/ilink"
)

// NewHandlerFromEnv creates the iLink connection HTTP handler from a dotenv file.
// Process environment variables are intentionally ignored.
func NewHandlerFromEnv(path string) (http.Handler, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return newHandler(cfg), nil
}

func newHandler(cfg config.Config) http.Handler {
	service := protocol.NewService(protocol.NewClient(cfg.ILinkBaseURL, cfg.ILinkAppID, cfg.ILinkClientVersion))
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/wechat/qrcode", qrcodeHandler(service))
	mux.HandleFunc("/api/v1/wechat/qrcode/status", statusHandler(service))
	return mux
}

func qrcodeHandler(service *protocol.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		result, err := service.StartLogin(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func statusHandler(service *protocol.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			writeError(w, http.StatusBadRequest, "session_id is required")
			return
		}
		result, err := service.LoginStatus(r.Context(), sessionID)
		if errors.Is(err, protocol.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
