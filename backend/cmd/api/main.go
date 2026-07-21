package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/config"
	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	httpapi "github.com/GonzaloSecades/nuchi/backend/internal/http"
	"github.com/GonzaloSecades/nuchi/backend/internal/mail"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()

	pool, err := db.NewPool(startupCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "host", databaseHost(cfg.DatabaseURL), "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database", "host", databaseHost(cfg.DatabaseURL))

	mailer := mail.NewSMTPMailer(cfg.SMTPAddr, cfg.MailFrom, cfg.AppBaseURL)
	authServer := httpapi.NewAuthServer(pool, cfg, mailer)

	server := &http.Server{
		Addr:    cfg.Addr(),
		Handler: httpapi.NewRouter(authServer),
		// ReadTimeout bounds the whole request read (headers + body), not
		// just headers: without it an anonymous client on the public auth
		// endpoints can hold a connection open indefinitely by streaming a
		// body one byte at a time. WriteTimeout and IdleTimeout bound the
		// response and keep-alive halves the same way.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("starting api", "addr", cfg.Addr())
		errs <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errs:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api stopped", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		logger.Info("shutting down api", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
	}
}

// databaseHost extracts the host portion of a database URL for logging.
// Credentials and other connection details are never logged.
func databaseHost(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Host == "" {
		return "unknown"
	}
	return parsed.Host
}
