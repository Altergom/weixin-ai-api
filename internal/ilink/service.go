package ilink

import (
	"context"
	"errors"
	"sync"
	"time"
)

type ModelConfig struct {
	Model   string `json:"model"`
	BaseURL string `json:"baseurl"`
	Key     string `json:"-"`
}
type Session struct {
	ID        string    `json:"session_id"`
	QRCode    QRCode    `json:"qrcode"`
	Status    string    `json:"status"`
	AccountID string    `json:"account_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Service struct {
	client   *Client
	mu       sync.RWMutex
	model    ModelConfig
	sessions map[string]*session
}
type session struct {
	Session
	code  string
	token string
}

func NewService(client *Client) *Service {
	return &Service{client: client, sessions: make(map[string]*session)}
}
func (s *Service) SetModel(cfg ModelConfig) error {
	if cfg.Model == "" || cfg.BaseURL == "" || cfg.Key == "" {
		return errors.New("model, baseurl and key are required")
	}
	s.mu.Lock()
	s.model = cfg
	s.mu.Unlock()
	return nil
}
func (s *Service) Model() (ModelConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.model.Model == "" {
		return ModelConfig{}, false
	}
	cfg := s.model
	cfg.Key = maskKey(cfg.Key)
	return cfg, true
}
func (s *Service) StartLogin(ctx context.Context) (*Session, error) {
	qr, err := s.client.QRCode(ctx)
	if err != nil {
		return nil, err
	}
	item := &session{Session: Session{ID: randomID(), QRCode: *qr, Status: "wait", ExpiresAt: time.Now().Add(2 * time.Minute)}, code: qr.Code}
	s.mu.Lock()
	s.sessions[item.ID] = item
	s.mu.Unlock()
	return &item.Session, nil
}
func (s *Service) LoginStatus(ctx context.Context, id string) (*Session, error) {
	s.mu.RLock()
	item := s.sessions[id]
	s.mu.RUnlock()
	if item == nil {
		return nil, errors.New("session not found")
	}
	if time.Now().After(item.ExpiresAt) {
		s.mu.Lock()
		item.Status = "expired"
		result := item.Session
		s.mu.Unlock()
		return &result, nil
	}
	status, err := s.client.QRCodeStatus(ctx, item.code)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	item.Status, item.AccountID, item.UserID, item.Error, item.token = status.Status, status.AccountID, status.UserID, status.Error, status.BotToken
	result := item.Session
	s.mu.Unlock()
	return &result, nil
}
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}
