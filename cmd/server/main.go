package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cairn.cn/internal/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()
	application, err := app.New(ctx, logger)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}
	defer application.Close()

	addr := fmt.Sprintf("%s:%d", application.Config.Server.Host, application.Config.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      application.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 40 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go shutdownOnSignal(srv)

	slog.Info("server starting", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func shutdownOnSignal(srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
