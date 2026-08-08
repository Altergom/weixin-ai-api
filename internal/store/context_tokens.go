package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const contextTokensFileName = "context_tokens.json"

// SaveContextToken 持久化一个 iLink 联系人的最新回复上下文。
func (s *FileStore) SaveContextToken(accountID, peerID, token string) error {
	key, err := contextTokenKey(accountID, peerID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("context token is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.loadContextTokensLocked()
	if err != nil {
		return err
	}
	tokens[key] = token
	data, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("encode context tokens: %w", err)
	}
	return writePrivateFile(filepath.Join(s.root, contextTokensFileName), data)
}

// LoadContextToken 在没有已知会话时返回空 token。
func (s *FileStore) LoadContextToken(accountID, peerID string) (string, error) {
	key, err := contextTokenKey(accountID, peerID)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, err := s.loadContextTokensLocked()
	if err != nil {
		return "", err
	}
	return tokens[key], nil
}

func (s *FileStore) loadContextTokensLocked() (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(s.root, contextTokensFileName))
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read context tokens: %w", err)
	}

	tokens := make(map[string]string)
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("decode context tokens: %w", err)
	}
	return tokens, nil
}

func contextTokenKey(accountID, peerID string) (string, error) {
	if strings.TrimSpace(accountID) == "" {
		return "", fmt.Errorf("context token account ID is required")
	}
	if strings.TrimSpace(peerID) == "" {
		return "", fmt.Errorf("context token peer ID is required")
	}
	return accountID + "\x00" + peerID, nil
}
