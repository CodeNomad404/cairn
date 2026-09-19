package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	authhandler "cairn.cn/internal/auth/handler"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/config"
	webmw "cairn.cn/internal/middleware"
)

type App struct {
	Config       *config.Config
	DB           *pgxpool.Pool
	Router       http.Handler
	workerCancel context.CancelFunc
}

func New(ctx context.Context, logger *slog.Logger) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	components := newApplicationComponents(cfg, db, logger)
	router := buildRouter(cfg, components)
	workerContext, workerCancel := context.WithCancel(context.Background())
	if err := startBackgroundWorkers(workerContext, cfg, components); err != nil {
		workerCancel()
		db.Close()
		return nil, err
	}
	return &App{Config: cfg, DB: db, Router: router, workerCancel: workerCancel}, nil
}

func (a *App) Close() {
	if a.workerCancel != nil {
		a.workerCancel()
	}
	if a.DB != nil {
		a.DB.Close()
	}
}

func buildRouter(
	cfg *config.Config,
	components *applicationComponents,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(webmw.TrustedProxyRealIP())
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(webmw.SecurityHeaders(cfg.IsProduction()))
	r.Use(webmw.SameOrigin())
	r.Use(authmid.LoadUser(components.authLogic))

	r.Get("/healthz", healthz)
	components.seoHandler.RegisterRoutes(r)
	r.Route("/api/v1", func(r chi.Router) {
		r.With(webmw.RateLimit(10, time.Minute)).Mount("/auth", authRoutes(components.authHandler))
		components.postHandler.RegisterRoutes(r)
		components.commentHandler.RegisterRoutes(r)
		components.toolHandler.RegisterRoutes(r)
		components.courseHandler.RegisterRoutes(r)
		components.classroomHandler.RegisterRoutes(r)
		components.accountingHandler.RegisterRoutes(r)
	})

	return r
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","version":"dev"}`))
}

func authRoutes(h *authhandler.AuthHandler) http.Handler {
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}
