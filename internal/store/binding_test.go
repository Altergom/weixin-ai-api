package store

import "testing"

func TestBindingValidate(t *testing.T) {
	binding := Binding{
		AccountID: "bot@im.bot",
		BaseURL:   "https://ilinkai.weixin.qq.com",
		BotToken:  "secret",
	}
	if err := binding.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestBindingPublicDoesNotExposeSecrets(t *testing.T) {
	binding := Binding{
		AccountID: "bot@im.bot",
		BotToken:  "secret",
		Cursor:    "sync-cursor",
	}
	public := binding.Public()
	if public.AccountID != binding.AccountID {
		t.Fatalf("Public() account ID = %q, want %q", public.AccountID, binding.AccountID)
	}
}
