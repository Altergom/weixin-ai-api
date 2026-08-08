package store

import (
	"path/filepath"
	"testing"
)

func TestFileStoreSaveAndLoadBinding(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	want := Binding{
		AccountID:    "bot@im.bot",
		WeixinUserID: "user@im.wechat",
		BaseURL:      "https://ilinkai.weixin.qq.com",
		BotToken:     "secret-token",
		Cursor:       "cursor-1",
		Status:       ConnectionStatusConnected,
	}
	if err := store.SaveBinding(want); err != nil {
		t.Fatalf("SaveBinding() error = %v", err)
	}

	got, err := store.LoadBinding()
	if err != nil {
		t.Fatalf("LoadBinding() error = %v", err)
	}
	if got == nil {
		t.Fatal("LoadBinding() = nil, want binding")
	}
	if got.AccountID != want.AccountID || got.BotToken != want.BotToken || got.Cursor != want.Cursor {
		t.Fatalf("LoadBinding() = %+v, want account, token and cursor preserved", got)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("LoadBinding() UpdatedAt is zero")
	}
}

func TestFileStoreLoadBindingAbsent(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	got, err := store.LoadBinding()
	if err != nil {
		t.Fatalf("LoadBinding() error = %v", err)
	}
	if got != nil {
		t.Fatalf("LoadBinding() = %+v, want nil", got)
	}
}
