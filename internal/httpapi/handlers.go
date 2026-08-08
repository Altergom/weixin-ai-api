package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// loginRequestTimeout 限制一次 iLink 创建/查询请求的时长，避免上游卡住 GUI。
// 这里使用较短超时，因为这些请求本身不是消息长轮询。
const loginRequestTimeout = 20 * time.Second

func (s *Server) registerRoutes() {
	s.engine.HandleFunc("GET /api/status", s.handleStatus)
	s.engine.HandleFunc("POST /api/scan", s.handleScanCreate)
	s.engine.HandleFunc("GET /api/scan/status", s.handleScanStatus)
	s.engine.HandleFunc("POST /api/connection/start", s.handleStart)
	s.engine.HandleFunc("POST /api/connection/stop", s.handleStop)
	s.engine.HandleFunc("POST /api/connection/reconnect", s.handleReconnect)
	s.engine.HandleFunc("GET /api/messages/next", s.handleNextMessage)
}

func (s *Server) handleNextMessage(w http.ResponseWriter, r *http.Request) {
	if s.deps.Messages == nil {
		s.fail(w, http.StatusServiceUnavailable, "message queue unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	message, err := s.deps.Messages.Next(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s.fail(w, http.StatusInternalServerError, "read message failed")
		return
	}
	writeJSON(w, http.StatusOK, message)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.deps.Coordinator.Status()
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "read status failed")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Coordinator.Start(r.Context()); err != nil {
		s.fail(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.deps.Coordinator.Stop()
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Coordinator.Restart(r.Context()); err != nil {
		s.fail(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reconnecting"})
}

func (s *Server) fail(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
