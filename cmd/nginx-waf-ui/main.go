package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/RumenDamyanov/nginx-waf-ui/internal/api"
	"github.com/RumenDamyanov/nginx-waf-ui/internal/config"
	"github.com/RumenDamyanov/nginx-waf-ui/internal/handler"
	"github.com/RumenDamyanov/nginx-waf-ui/internal/session"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "/etc/nginx-waf-ui/config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion || (len(os.Args) > 1 && os.Args[1] == "--version") {
		fmt.Printf("nginx-waf-ui %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	var level slog.Level
	switch strings.ToLower(cfg.Logging.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	// Create dependencies
	apiClient := api.NewClient(cfg.API.URL, cfg.API.APIKey, cfg.API.Timeout)
	store := session.NewStore(cfg.Session.Secret, cfg.Session.MaxAge)

	// Admin password from environment, fallback to "admin"
	adminPw := os.Getenv("NGINX_WAF_UI_ADMIN_PASSWORD")
	if adminPw == "" {
		adminPw = "admin"
		logger.Warn("using default admin password, set NGINX_WAF_UI_ADMIN_PASSWORD in production")
	}

	h := handler.New(apiClient, store, logger, adminPw)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: mux,
	}

	logger.Info("starting nginx-waf-ui", "listen", cfg.Server.Listen, "version", version)

	go func() {
		var listenErr error
		if cfg.Server.TLS.Cert != "" && cfg.Server.TLS.Key != "" {
			listenErr = srv.ListenAndServeTLS(cfg.Server.TLS.Cert, cfg.Server.TLS.Key)
		} else {
			listenErr = srv.ListenAndServe()
		}
		if listenErr != nil && listenErr != http.ErrServerClosed {
			logger.Error("server error", "error", listenErr)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*1e9)
	defer cancel()
	srv.Shutdown(ctx)
}
