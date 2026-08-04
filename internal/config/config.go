package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultILinkBaseURL       = "https://ilinkai.weixin.qq.com"
	defaultILinkAppID         = "bot"
	defaultILinkClientVersion = "1.0.0"
)

type Config struct {
	ILinkBaseURL       string
	ILinkAppID         string
	ILinkClientVersion string
}

func Load(path string) (Config, error) {
	cfg := Config{
		ILinkBaseURL:       defaultILinkBaseURL,
		ILinkAppID:         defaultILinkAppID,
		ILinkClientVersion: defaultILinkClientVersion,
	}

	values, err := godotenv.Read(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: invalid format", filepath.Base(path))
	}

	cfg.ILinkBaseURL = valueOrDefault(values, "ILINK_BASE_URL", cfg.ILinkBaseURL)
	cfg.ILinkAppID = valueOrDefault(values, "ILINK_APP_ID", cfg.ILinkAppID)
	cfg.ILinkClientVersion = valueOrDefault(values, "ILINK_CLIENT_VERSION", cfg.ILinkClientVersion)

	return cfg, nil
}

func valueOrDefault(values map[string]string, key, fallback string) string {
	if value := strings.TrimSpace(values[key]); value != "" {
		return value
	}
	return fallback
}
