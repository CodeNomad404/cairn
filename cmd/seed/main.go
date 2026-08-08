package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/blog/logic"
	"cairn.cn/internal/blog/repository"
	"cairn.cn/internal/config"
	"cairn.cn/internal/util"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		slog.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	md, err := os.ReadFile("docs/day-one-server-ready.md")
	if err != nil {
		slog.Error("read file", "error", err)
		os.Exit(1)
	}

	l := logic.NewPostLogic(repository.New(db))
	_, code, msg := l.Create(
		ctx,
		"day-one-server-ready",
		"Cairn 的第一天：把服务器搞定了",
		string(md),
	)
	if code != util.CodeCreated {
		slog.Error("publish", "code", code, "msg", msg)
		os.Exit(1)
	}

	slog.Info("published", "slug", "day-one-server-ready")
}
