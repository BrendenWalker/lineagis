package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BrendenWalker/verity/internal/api"
	"github.com/BrendenWalker/verity/internal/auth"
	"github.com/BrendenWalker/verity/internal/config"
	"github.com/BrendenWalker/verity/internal/db"
	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/jackc/pgx/v5/pgxpool"
)

type readinessChecker interface {
	PingDB(ctx context.Context) error
	CheckRegistry(ctx context.Context) error
}

type server struct {
	cfg    config.Config
	pool   *pgxpool.Pool
	authn  *auth.Authenticator
	client *http.Client
	log    *slog.Logger
}

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		return 1
	}

	log := newLogger(cfg)

	ctx := context.Background()
	pool, err := db.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect database", "error", err)
		return 1
	}
	defer pool.Close()

	if cfg.MigrateOnStartup {
		if err := db.MigrateUp(ctx, pool); err != nil {
			log.Error("run migrations", "error", err)
			return 1
		}
		log.Info("database migrations applied")
	}

	authn, err := auth.New(ctx, auth.Config{
		DevToken: cfg.DevToken,
		Issuer:   cfg.OIDCIssuer,
		Audience: cfg.OIDCAudience,
	})
	if err != nil {
		log.Error("configure authentication", "error", err)
		return 1
	}

	srv := &server{
		cfg:    cfg,
		pool:   pool,
		authn:  authn,
		client: &http.Client{Timeout: 5 * time.Second},
		log:    log,
	}

	httpSrv := &http.Server{
		Addr:    cfg.APIAddr,
		Handler: srv.routes(),
	}

	go func() {
		var err error
		if cfg.TLSCertFile != "" {
			log.Info("starting api server with tls", "addr", cfg.APIAddr)
			err = httpSrv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			log.Info("starting api server", "addr", cfg.APIAddr)
			err = httpSrv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "error", err)
		return 1
	}
	return 0
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var handler slog.Handler
	if cfg.LogFormat == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthHandler)
	mux.HandleFunc("/readyz", s.readyHandler)

	store := metadata.NewStore(s.pool)
	apiHandler := &api.Handler{
		Store:  store,
		Policy: api.NewStorePushPolicy(store),
		Auth:   api.AuthMiddleware(s.authn),
	}
	apiHandler.RegisterRoutes(mux)
	return mux
}

func (s *server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

func (s *server) readyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.PingDB(ctx); err != nil {
		s.log.Warn("readiness check failed", "component", "database", "error", err)
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := s.CheckRegistry(ctx); err != nil {
		s.log.Warn("readiness check failed", "component", "registry", "error", err)
		http.Error(w, "registry unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

func (s *server) PingDB(ctx context.Context) error {
	return db.Ping(ctx, s.pool)
}

func (s *server) CheckRegistry(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.RegistryURL+"/v2/", nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return errUnexpectedStatus(resp.StatusCode)
	}
	return nil
}

type unexpectedStatusError struct {
	code int
}

func errUnexpectedStatus(code int) error {
	return unexpectedStatusError{code: code}
}

func (e unexpectedStatusError) Error() string {
	return "unexpected registry status: " + http.StatusText(e.code)
}

// testMux exposes routes for handler tests without starting the full server.
func testMux(checker readinessChecker) *http.ServeMux {
	s := &server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthHandler)
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := checker.PingDB(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := checker.CheckRegistry(ctx); err != nil {
			http.Error(w, "registry unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	return mux
}
