package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hiTechTeam/Shum-Push-Server/internal/api"
	"github.com/hiTechTeam/Shum-Push-Server/internal/apns"
	"github.com/hiTechTeam/Shum-Push-Server/internal/config"
	"github.com/hiTechTeam/Shum-Push-Server/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	devices, err := store.Open(cfg.DataFile)
	if err != nil {
		logger.Error("device store failed", "error", err)
		os.Exit(1)
	}
	var push api.PushSender
	if cfg.APNSConfigured() {
		push, err = apns.New(cfg.APNSKeyID, cfg.APNSTeamID, cfg.APNSBundleID, cfg.APNSPrivateKeyFile)
		if err != nil {
			logger.Error("APNs client failed", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Warn("APNs credentials are not configured; registration works but delivery is disabled")
	}
	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           api.New(devices, push, cfg.GatewaySecret, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("Shum push server started", "address", cfg.ListenAddress, "apns_configured", cfg.APNSConfigured())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
