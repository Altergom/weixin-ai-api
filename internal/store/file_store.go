package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const bindingFileName = "binding.json"

// FileStore 将本地 iLink 状态放在非敏感应用配置之外。
// root 必须位于当前用户的数据目录下。
type FileStore struct {
	root string
	mu   sync.Mutex
}

// NewFileStore 在需要时创建受限的数据目录。
func NewFileStore(root string) (*FileStore, error) {
	if root == "" {
		return nil, errors.New("store root is required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("restrict store directory: %w", err)
	}
	return &FileStore{root: root}, nil
}

// SaveBinding 以原子方式持久化已认证的 iLink 绑定。
func (s *FileStore) SaveBinding(binding Binding) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	binding.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(binding)
	if err != nil {
		return fmt.Errorf("encode binding: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return writePrivateFile(filepath.Join(s.root, bindingFileName), data)
}

// LoadBinding 在客户端尚未认证账号时返回 nil。
func (s *FileStore) LoadBinding() (*Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(filepath.Join(s.root, bindingFileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read binding: %w", err)
	}

	var binding Binding
	if err := json.Unmarshal(data, &binding); err != nil {
		return nil, fmt.Errorf("decode binding: %w", err)
	}
	if err := binding.Validate(); err != nil {
		return nil, fmt.Errorf("validate saved binding: %w", err)
	}
	return &binding, nil
}

// ClearBinding 删除当前 bot 绑定及联系人上下文凭据。
func (s *FileStore) ClearBinding() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range []string{bindingFileName, contextTokensFileName} {
		err := os.Remove(filepath.Join(s.root, name))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	return nil
}

func writePrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create private temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("restrict private temp file: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write private temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync private temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close private temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace private file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict private file: %w", err)
	}
	return nil
}
