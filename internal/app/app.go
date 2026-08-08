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
	authlogic "cairn.cn/internal/auth/logic"
	authmid "cairn.cn/internal/auth/middleware"
	authrepo "cairn.cn/internal/auth/repository"
	bloghandler "cairn.cn/internal/blog/handler"
	bloglogic "cairn.cn/internal/blog/logic"
	blogrepo "cairn.cn/internal/blog/repository"
	"cairn.cn/internal/config"
	"cairn.cn/internal/seo"
	toolhandler "cairn.cn/internal/tool/handler"
	toollogic "cairn.cn/internal/tool/logic"
	"cairn.cn/internal/web"
)

type App struct {
	Config *config.Config
	DB     *pgxpool.Pool
	Router http.Handler
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

	router := buildRouter(cfg, db)
	_ = logger
	return &App{Config: cfg, DB: db, Router: router}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}

func buildRouter(cfg *config.Config, db *pgxpool.Pool) http.Handler {
	// Composition root：Repository、Logic、Handler 均只组装一次，请求状态只通过 Context 传递。
	userRepo := authrepo.NewUserRepo(db)
	sessionRepo := authrepo.NewSessionRepo(db)
	postRepo := blogrepo.New(db)
	commentRepo := blogrepo.NewCommentRepo(db)

	authLogic := authlogic.NewAuthLogic(userRepo, sessionRepo)
	postLogic := bloglogic.NewPostLogic(postRepo)
	commentLogic := bloglogic.NewCommentLogic(postRepo, commentRepo)
	jsonLogic := toollogic.NewJSONLogic()

	authHandler := authhandler.NewAuthHandler(authLogic, cfg.IsProduction())
	postHandler := bloghandler.NewPostHandler(postLogic)
	commentHandler := bloghandler.NewCommentHandler(commentLogic)
	toolHandler := toolhandler.New(jsonLogic)
	seoHandler := seo.New(postLogic)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.TrustedProxyRealIP())
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(web.SecurityHeaders(cfg.IsProduction()))
	r.Use(web.SameOrigin())
	r.Use(authmid.LoadUser(authLogic))

	r.Get("/healthz", healthz)
	seoHandler.RegisterRoutes(r)
	r.Route("/api/v1", func(r chi.Router) {
		r.With(web.RateLimit(10, time.Minute)).Mount("/auth", authRoutes(authHandler))
		postHandler.RegisterRoutes(r)
		commentHandler.RegisterRoutes(r)
		toolHandler.RegisterRoutes(r)
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
