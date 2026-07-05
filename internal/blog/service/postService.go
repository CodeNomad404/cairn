package service

import (
	"context"
	"strings"
	"time"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"cairn.cn/internal/blog"
)

// PostService 博客文章相关的业务逻辑
type PostService struct {
	repo   blog.PostRepo
	md     goldmark.Markdown
	policy *bluemonday.Policy
}

// New 创建 PostService 实例
func New(repo blog.PostRepo) *PostService {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(false), // 内联样式，无需额外 CSS
				),
			),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)

	policy := bluemonday.UGCPolicy().
		AllowAttrs("class", "style").OnElements("span", "pre", "code").
		AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	return &PostService{repo: repo, md: md, policy: policy}
}

// Publish 发布文章：Markdown → HTML → 安全过滤，写入数据库
func (s *PostService) Publish(ctx context.Context, slug, title, contentMD string) (*blog.Post, error) {
	var buf strings.Builder
	if err := s.md.Convert([]byte(contentMD), &buf); err != nil {
		return nil, err
	}

	// XSS 过滤：只保留安全的 HTML 标签和属性
	safeHTML := s.policy.Sanitize(buf.String())

	slug = strings.ToLower(strings.TrimSpace(slug))
	title = strings.TrimSpace(title)
	publishedAt := time.Now().UTC()

	p := &blog.Post{
		Slug:        slug,
		Title:       title,
		ContentMD:   contentMD,
		ContentHTML: safeHTML,
		Status:      "published",
		PublishedAt: &publishedAt,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetBySlug 按 slug 取文章
func (s *PostService) GetBySlug(ctx context.Context, slug string) (*blog.Post, error) {
	return s.repo.GetBySlug(ctx, slug)
}

// ListPublished 取已发布文章列表
func (s *PostService) ListPublished(ctx context.Context, limit, offset int) ([]blog.Post, error) {
	return s.repo.ListPublished(ctx, limit, offset)
}

func (s *PostService) ListPublishedMetadata(ctx context.Context) ([]blog.Post, error) {
	return s.repo.ListPublishedMetadata(ctx)
}
