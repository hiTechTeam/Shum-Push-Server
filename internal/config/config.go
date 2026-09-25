package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	ListenAddress                string
	DataFile                     string
	GatewaySecret                string
	APNSSandboxKeyID             string
	APNSSandboxPrivateKeyFile    string
	APNSProductionKeyID          string
	APNSProductionPrivateKeyFile string
	APNSTeamID                   string
	APNSBundleID                 string
	ShutdownTimeout              time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddress:                value("SHUM_LISTEN_ADDRESS", ":8080"),
		DataFile:                     value("SHUM_DATA_FILE", "/var/lib/shum-push/devices.json"),
		GatewaySecret:                strings.TrimSpace(os.Getenv("SHUM_GATEWAY_SECRET")),
		APNSSandboxKeyID:             strings.TrimSpace(os.Getenv("APNS_SANDBOX_KEY_ID")),
		APNSSandboxPrivateKeyFile:    strings.TrimSpace(os.Getenv("APNS_SANDBOX_PRIVATE_KEY_FILE")),
		APNSProductionKeyID:          strings.TrimSpace(os.Getenv("APNS_PRODUCTION_KEY_ID")),
		APNSProductionPrivateKeyFile: strings.TrimSpace(os.Getenv("APNS_PRODUCTION_PRIVATE_KEY_FILE")),
		APNSTeamID:                   strings.TrimSpace(os.Getenv("APNS_TEAM_ID")),
		APNSBundleID:                 value("APNS_BUNDLE_ID", "hiTeam.ShumiOS"),
		ShutdownTimeout:              10 * time.Second,
	}
	if strings.TrimSpace(cfg.ListenAddress) == "" {
		return Config{}, errors.New("SHUM_LISTEN_ADDRESS cannot be empty")
	}
	return cfg, nil
}

func (c Config) APNSConfigured() bool {
	return c.APNSSandboxKeyID != "" && c.APNSSandboxPrivateKeyFile != "" &&
		c.APNSProductionKeyID != "" && c.APNSProductionPrivateKeyFile != "" &&
		c.APNSTeamID != "" && c.APNSBundleID != ""
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
