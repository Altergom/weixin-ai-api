package store

import (
	"path/filepath"
	"testing"
)

func TestFileStoreContextTokensAreScopedToAccountAndPeer(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := store.SaveContextToken("account-a", "peer-a", "token-a"); err != nil {
		t.Fatalf("SaveContextToken() error = %v", err)
	}
	if err := store.SaveContextToken("account-b", "peer-a", "token-b"); err != nil {
		t.Fatalf("SaveContextToken() error = %v", err)
	}

	got, err := store.LoadContextToken("account-a", "peer-a")
	if err != nil {
		t.Fatalf("LoadContextToken() error = %v", err)
	}
	if got != "token-a" {
		t.Fatalf("LoadContextToken() = %q, want token-a", got)
	}
	missing, err := store.LoadContextToken("account-a", "peer-b")
	if err != nil {
		t.Fatalf("LoadContextToken() missing error = %v", err)
	}
	if missing != "" {
		t.Fatalf("LoadContextToken() missing = %q, want empty", missing)
	}
}
