package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	authhandler "cairn.cn/internal/auth/handler"
	authmid "cairn.cn/internal/auth/middleware"
	authrepo "cairn.cn/internal/auth/repository"
	authsvc "cairn.cn/internal/auth/service"
	bloghandler "cairn.cn/internal/blog/handler"
	blogrepo "cairn.cn/internal/blog/repository"
	blogsvc "cairn.cn/internal/blog/service"
	"cairn.cn/internal/config"
	toolhandler "cairn.cn/internal/tool/handler"
	toolsvc "cairn.cn/internal/tool/service"
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

	renderer, err := web.NewRenderer("web/templates/*.html", logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("load templates: %w", err)
	}

	router := buildRouter(cfg, db, renderer)
	return &App{Config: cfg, DB: db, Router: router}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}

func buildRouter(cfg *config.Config, db *pgxpool.Pool, renderer *web.Renderer) http.Handler {
	userRepo := authrepo.NewUserRepo(db)
	sessionRepo := authrepo.NewSessionRepo(db)
	authService := authsvc.New(userRepo, sessionRepo)
	authHandler := authhandler.New(authService, renderer, cfg.IsProduction())

	postRepo := blogrepo.New(db)
	commentRepo := blogrepo.NewCommentRepo(db)
	postService := blogsvc.New(postRepo)
	commentService := blogsvc.NewCommentService(commentRepo)
	postHandler := bloghandler.New(postService, commentService, renderer)
	commentHandler := bloghandler.NewCommentHandler(postService, commentService, renderer)

	toolService := toolsvc.New()
	toolHandler := toolhandler.New(renderer, toolService)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.TrustedProxyRealIP())
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(web.SecurityHeaders(cfg.IsProduction()))
	r.Use(web.SameOrigin())
	r.Use(authmid.LoadUser(authService))

	fileServer(r, "/static", http.Dir("web/static"))

	r.Get("/healthz", healthz)

	r.Get("/login", authHandler.LoginPage)
	r.With(web.RateLimit(10, time.Minute)).Post("/login", authHandler.Login)
	r.Get("/register", authHandler.RegisterPage)
	r.With(web.RateLimit(5, time.Hour)).Post("/register", authHandler.Register)
	r.Post("/logout", authHandler.Logout)

	postHandler.RegisterRoutes(r)
	commentHandler.RegisterRoutes(r)
	toolHandler.RegisterRoutes(r)

	return r
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","version":"dev"}`))
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))
	r.Get(path+"/*", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("DEV") == "true" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fs.ServeHTTP(w, r)
	})
}
