package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"hrprogress/internal/audit"
	"hrprogress/internal/auth"
	"hrprogress/internal/competency"
	"hrprogress/internal/config"
	"hrprogress/internal/db"
	"hrprogress/internal/httpx"
	"hrprogress/internal/onef"
	"hrprogress/internal/workers"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", slog.String("err", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	migrationsURL := os.Getenv("MIGRATIONS_URL")
	if migrationsURL == "" {
		migrationsURL = "file:///app/migrations"
	}
	if err := db.RunMigrations(cfg.DatabaseURL, migrationsURL); err != nil {
		log.Error("migrations", slog.String("err", err.Error()))
		os.Exit(1)
	}
	log.Info("migrations applied")

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db pool", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := auth.EnsureBootstrapAdmin(ctx, pool, log,
		cfg.BootstrapAdminUsername, cfg.BootstrapAdminPassword); err != nil {
		log.Error("bootstrap admin", slog.String("err", err.Error()))
		os.Exit(1)
	}

	auditWriter := audit.NewWriter(pool, log)
	authRepo := auth.NewRepository(pool)
	jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, cfg.AccessTokenTTL)
	authSvc := auth.NewService(pool, authRepo, jwtIssuer, auditWriter, cfg.RefreshTokenTTL)
	authHandler := auth.NewHandler(authSvc, cfg.RefreshTokenTTL, cfg.AppEnv == "production")

	compRepo := competency.NewRepository(pool)
	compSvc := competency.NewService(compRepo)
	compHandler := competency.NewHandler(compSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httpx.RequestLogger(log))
	r.Use(httpx.Recoverer(log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   splitCSV(cfg.CORSAllowedOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "db not reachable")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	workersRepo := workers.NewRepository(pool)
	workersHandler := workers.NewHandler(workersRepo)

	oneFClient := onef.NewClient(cfg.OneFBaseURL, cfg.OneFAuthToken)
	oneFRepo := onef.NewRepository(pool)
	oneFSvc := onef.NewService(oneFRepo, oneFClient, workersRepo, log)
	oneFHandler := onef.NewHandler(oneFSvc)
	onef.StartScheduler(ctx, oneFSvc, cfg.OneFSyncInterval, log)

	r.Route("/api/v1", func(r chi.Router) {
		authHandler.Mount(r, jwtIssuer)
		compHandler.Mount(r, jwtIssuer)
		workersHandler.Mount(r, jwtIssuer)
		oneFHandler.Mount(r, jwtIssuer)
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
