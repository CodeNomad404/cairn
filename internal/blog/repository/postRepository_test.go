package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/blog"
	blogmodel "cairn.cn/internal/blog/model"
)

// TestPostRepositorySoftDelete 验证软删除语义：UPDATE status、记录保留、
// published 查询过滤、重复删除返回 ErrPostNotFound。
// 仅当设置 CAIRN_TEST_DATABASE_URL 时运行，绝不连接开发数据库。
func TestPostRepositorySoftDelete(t *testing.T) {
	dsn := os.Getenv("CAIRN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CAIRN_TEST_DATABASE_URL 未设置，跳过 Repository 集成测试")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	r := New(pool)
	slug := fmt.Sprintf("soft-delete-it-%d", time.Now().UnixNano())
	publishedAt := time.Now().UTC()

	// 创建一篇 published 测试文章
	post := &blogmodel.Post{
		Slug:        slug,
		Title:       "integration test",
		ContentMD:   "# integration test",
		ContentHTML: "<h1>integration test</h1>",
		Status:      blog.PostStatusPublished,
		PublishedAt: &publishedAt,
	}
	if err := r.Create(ctx, post); err != nil {
		t.Fatal(err)
	}
	// 测试结束只清理自己创建的数据。
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM posts WHERE slug = $1`, slug)
	}()

	// 软删除
	if err := r.Delete(ctx, slug); err != nil {
		t.Fatal(err)
	}

	// 记录仍存在，status 已变为 deleted
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM posts WHERE slug = $1`, slug).Scan(&status); err != nil {
		t.Fatalf("文章记录不存在（不应物理删除）: %v", err)
	}
	if status != blog.PostStatusDeleted {
		t.Fatalf("status = %q, want %q", status, blog.PostStatusDeleted)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM posts WHERE slug = $1`, slug).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("记录数 = %d, want 1", count)
	}

	// GetBySlug 无法再读取该文章
	if _, err := r.GetBySlug(ctx, slug); !errors.Is(err, blog.ErrPostNotFound) {
		t.Fatalf("GetBySlug err = %v, want ErrPostNotFound", err)
	}

	// ListPublished 不再包含该文章
	posts, err := r.ListPublished(ctx, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range posts {
		if p.Slug == slug {
			t.Fatal("ListPublished 仍包含已删除文章")
		}
	}

	// 第二次 Delete 返回 ErrPostNotFound
	if err := r.Delete(ctx, slug); !errors.Is(err, blog.ErrPostNotFound) {
		t.Fatalf("重复 Delete err = %v, want ErrPostNotFound", err)
	}
}
