package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/EraldCaka/pi-web/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	srv := server.New(logger)

	go func() {
		logger.Info("server starting", "addr", ":3000")

		if err := srv.Start(":3000"); err != nil {
			logger.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("shutdown signal received", "signal", sig.String())

	if err := srv.Shutdown(10 * time.Second); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped cleanly")
}
