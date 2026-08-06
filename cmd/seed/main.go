package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/blog/logic"
	"cairn.cn/internal/blog/repository"
	"cairn.cn/internal/config"
)

func main() {
	// 读配置
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	// 连数据库
	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		slog.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 读 Markdown 文件
	md, err := os.ReadFile("docs/day-one-server-ready.md")
	if err != nil {
		slog.Error("read file", "error", err)
		os.Exit(1)
	}

	// 插入
	repo := repository.New(db)
	svc := logic.NewPostLogic(repo)
	post, err := svc.Publish(ctx, &logic.PublishPostRequest{
		Slug:      "day-one-server-ready",
		Title:     "Cairn 的第一天：把服务器搞定了",
		ContentMD: string(md),
	})
	if err != nil {
		slog.Error("publish", "error", err)
		os.Exit(1)
	}

	slog.Info("published", "id", post.ID, "slug", post.Slug)
}
