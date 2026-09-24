package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	ListenAddress      string
	DataFile           string
	GatewaySecret      string
	APNSKeyID          string
	APNSTeamID         string
	APNSBundleID       string
	APNSPrivateKeyFile string
	ShutdownTimeout    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddress:      value("SHUM_LISTEN_ADDRESS", ":8080"),
		DataFile:           value("SHUM_DATA_FILE", "/var/lib/shum-push/devices.json"),
		GatewaySecret:      strings.TrimSpace(os.Getenv("SHUM_GATEWAY_SECRET")),
		APNSKeyID:          strings.TrimSpace(os.Getenv("APNS_KEY_ID")),
		APNSTeamID:         strings.TrimSpace(os.Getenv("APNS_TEAM_ID")),
		APNSBundleID:       value("APNS_BUNDLE_ID", "hiTeam.ShumiOS"),
		APNSPrivateKeyFile: strings.TrimSpace(os.Getenv("APNS_PRIVATE_KEY_FILE")),
		ShutdownTimeout:    10 * time.Second,
	}
	if strings.TrimSpace(cfg.ListenAddress) == "" {
		return Config{}, errors.New("SHUM_LISTEN_ADDRESS cannot be empty")
	}
	return cfg, nil
}

func (c Config) APNSConfigured() bool {
	return c.APNSKeyID != "" && c.APNSTeamID != "" && c.APNSBundleID != "" && c.APNSPrivateKeyFile != ""
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
