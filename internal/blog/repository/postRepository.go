package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/blog"
	blogmodel "cairn.cn/internal/blog/model"
)

// PostRepository 负责 posts 表的数据库操作
type PostRepository struct {
	db *pgxpool.Pool
}

// New 创建 PostRepository 实例
func New(db *pgxpool.Pool) *PostRepository {
	return &PostRepository{db: db}
}

// Create 插入一篇新文章
func (r *PostRepository) Create(ctx context.Context, p *blogmodel.Post) error {
	sql := `INSERT INTO posts (slug, title, content_md, content_html, status, published_at)
	        VALUES ($1, $2, $3, $4, $5, $6)
	        RETURNING id, created_at, updated_at`

	args := []any{p.Slug, p.Title, p.ContentMD, p.ContentHTML, p.Status, p.PublishedAt}
	return r.db.QueryRow(ctx, sql, args...).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// GetBySlug 根据 slug 查询一篇文章
func (r *PostRepository) GetBySlug(ctx context.Context, slug string) (*blogmodel.Post, error) {
	sql := `SELECT id, slug, title, content_md, content_html, status,
	               created_at, updated_at, published_at
	        FROM posts WHERE slug = $1 AND status = 'published'`

	p := &blogmodel.Post{}
	var pubAt *time.Time
	err := r.db.QueryRow(ctx, sql, slug).Scan(
		&p.ID, &p.Slug, &p.Title, &p.ContentMD, &p.ContentHTML,
		&p.Status, &p.CreatedAt, &p.UpdatedAt, &pubAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", blog.ErrPostNotFound, slug)
		}
		return nil, err
	}
	p.PublishedAt = pubAt
	return p, nil
}

// ListPublished 返回已发布文章列表，按发布时间倒序
func (r *PostRepository) ListPublished(ctx context.Context, limit, offset int) ([]blogmodel.Post, error) {
	sql := `SELECT id, slug, title, content_md, content_html, status,
	               created_at, updated_at, published_at
	        FROM posts WHERE status = 'published'
	        ORDER BY published_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, sql, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []blogmodel.Post
	for rows.Next() {
		var p blogmodel.Post
		var pubAt *time.Time
		err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.ContentMD, &p.ContentHTML,
			&p.Status, &p.CreatedAt, &p.UpdatedAt, &pubAt)
		if err != nil {
			return nil, err
		}
		p.PublishedAt = pubAt
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// ListPublishedMetadata 返回 Sitemap 所需的最小文章元数据。
func (r *PostRepository) ListPublishedMetadata(ctx context.Context) ([]blogmodel.Post, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, updated_at
		FROM posts WHERE status = 'published' ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []blogmodel.Post
	for rows.Next() {
		var post blogmodel.Post
		if err := rows.Scan(&post.Slug, &post.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}
