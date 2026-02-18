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

	"github.com/nickdawson/sambmin/internal/auth"
	"github.com/nickdawson/sambmin/internal/config"
	"github.com/nickdawson/sambmin/internal/directory"
	"github.com/nickdawson/sambmin/internal/handlers"
	sambldap "github.com/nickdawson/sambmin/internal/ldap"
	"github.com/nickdawson/sambmin/internal/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize directory client if DCs are configured
	var dirClient *directory.Client
	if len(cfg.DCs) > 0 && cfg.BaseDN != "" {
		dirClient, err = initDirectory(cfg)
		if err != nil {
			slog.Warn("LDAP initialization failed, running in mock mode", "error", err)
		} else {
			slog.Info("LDAP connected, using live directory data")
		}
	} else {
		slog.Info("no DCs configured, running in mock mode")
	}

	// Initialize authentication if DCs are configured
	if len(cfg.DCs) > 0 && cfg.BaseDN != "" {
		if err := initAuth(cfg); err != nil {
			slog.Warn("auth initialization failed, login disabled", "error", err)
		} else {
			slog.Info("authentication system initialized")
		}
	}

	mux := http.NewServeMux()
	handlers.Register(mux, cfg, dirClient)

	handler := middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logger,
		middleware.Recovery,
		middleware.CORS(cfg.AllowedOrigins),
	)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("starting sambmin server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func initDirectory(cfg *config.Config) (*directory.Client, error) {
	// Build DC list for pool
	dcs := make([]sambldap.DCInfo, len(cfg.DCs))
	for i, dc := range cfg.DCs {
		port := dc.Port
		if port == 0 {
			port = 636 // default to LDAPS
		}
		dcs[i] = sambldap.DCInfo{
			Hostname: dc.Hostname,
			Address:  dc.Address,
			Port:     port,
			Site:     dc.Site,
			Primary:  dc.Primary,
		}
	}

	// Bind password from env takes precedence over config
	bindPW := os.Getenv("SAMBMIN_BIND_PW")
	if bindPW == "" {
		bindPW = cfg.BindPW
	}
	// Store resolved password back into config so handlers can use it
	cfg.BindPW = bindPW

	pool, err := sambldap.NewPool(sambldap.PoolConfig{
		DCs:     dcs,
		BaseDN:  cfg.BaseDN,
		BindDN:  cfg.BindDN,
		BindPW:  bindPW,
		UseTLS:  true,
		MaxIdle: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("create LDAP pool: %w", err)
	}

	client := directory.NewClient(pool, cfg.BaseDN)

	// Test connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Health(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("LDAP health check: %w", err)
	}

	return client, nil
}

func initAuth(cfg *config.Config) error {
	// Create session store with AES-GCM password encryption
	store, err := auth.NewStore(cfg.SessionTimeout)
	if err != nil {
		return fmt.Errorf("create session store: %w", err)
	}

	// Find primary DC for authentication binds
	var dcAddress, dcHost string
	for _, dc := range cfg.DCs {
		if dc.Primary {
			port := dc.Port
			if port == 0 {
				port = 636
			}
			dcAddress = fmt.Sprintf("%s:%d", dc.Address, port)
			dcHost = dc.Hostname
			break
		}
	}
	if dcAddress == "" && len(cfg.DCs) > 0 {
		dc := cfg.DCs[0]
		port := dc.Port
		if port == 0 {
			port = 636
		}
		dcAddress = fmt.Sprintf("%s:%d", dc.Address, port)
		dcHost = dc.Hostname
	}

	useTLS := true // All DCs use LDAPS (port 636)
	authenticator := auth.NewLDAPAuthenticator(dcAddress, dcHost, cfg.BaseDN, useTLS)

	handlers.InitAuth(store, authenticator)
	slog.Info("auth: LDAP authenticator targeting", "dc", dcHost, "address", dcAddress)
	return nil
}
