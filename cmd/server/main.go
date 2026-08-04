package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"weixin-ilink-service/internal/ilink"
)

func main() {
	client := ilink.NewClient(env("ILINK_BASE_URL", "https://ilinkai.weixin.qq.com"), env("ILINK_APP_ID", "bot"), env("ILINK_CLIENT_VERSION", "1.0.0"))
	service := ilink.NewService(client)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/model", modelHandler(service))
	mux.HandleFunc("/api/v1/wechat/qrcode", qrcodeHandler(service))
	mux.HandleFunc("/api/v1/wechat/qrcode/status", statusHandler(service))
	addr := env("LISTEN_ADDR", ":8080")
	server := &http.Server{Addr: addr, Handler: logging(mux), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("weixin iLink service listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func modelHandler(s *ilink.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			cfg, ok := s.Model()
			if !ok {
				writeError(w, http.StatusNotFound, "model is not configured")
				return
			}
			writeJSON(w, http.StatusOK, cfg)
		case http.MethodPut, http.MethodPost:
			var cfg ilink.ModelConfig
			if json.NewDecoder(r.Body).Decode(&cfg) != nil {
				writeError(w, 400, "invalid JSON")
				return
			}
			if err := s.SetModel(cfg); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		default:
			w.Header().Set("Allow", "GET, PUT, POST")
			writeError(w, 405, "method not allowed")
		}
	}
}
func qrcodeHandler(s *ilink.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, 405, "method not allowed")
			return
		}
		result, err := s.StartLogin(r.Context())
		if err != nil {
			writeError(w, 502, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
func statusHandler(s *ilink.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, 405, "method not allowed")
			return
		}
		id := r.URL.Query().Get("session_id")
		if id == "" {
			writeError(w, 400, "session_id is required")
			return
		}
		result, err := s.LoginStatus(r.Context(), id)
		if err != nil {
			writeError(w, 404, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(started))
	})
}
