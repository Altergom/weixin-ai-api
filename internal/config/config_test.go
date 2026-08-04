package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsEnvFile(t *testing.T) {
	path := writeEnvFile(t, `
ILINK_BASE_URL=https://example.com/
ILINK_APP_ID=custom-bot
ILINK_CLIENT_VERSION=2.3.4
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ILinkBaseURL != "https://example.com/" || cfg.ILinkAppID != "custom-bot" || cfg.ILinkClientVersion != "2.3.4" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestLoadIgnoresProcessEnvironment(t *testing.T) {
	t.Setenv("ILINK_BASE_URL", "https://system.example.com")

	cfg, err := Load(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ILinkBaseURL != defaultILinkBaseURL {
		t.Fatalf("got %q, want default %q", cfg.ILinkBaseURL, defaultILinkBaseURL)
	}
}

func writeEnvFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
