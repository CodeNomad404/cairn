package logic

import (
	"context"
	"errors"
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

// PublishPostRequest is the request body for creating a new blog post.
type PublishPostRequest struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	ContentMD string `json:"content_md"`
}

// Validate checks the API contract for publishing a post.
func (r PublishPostRequest) Validate() error {
	if strings.TrimSpace(r.Slug) == "" || strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.ContentMD) == "" {
		return errors.New("slug, title, and content_md are required")
	}
	return nil
}

// PostLogic 博客文章相关的业务逻辑
type PostLogic struct {
	repo   blog.PostRepo
	md     goldmark.Markdown
	policy *bluemonday.Policy
}

// NewPostLogic 创建 PostLogic 实例
func NewPostLogic(repo blog.PostRepo) *PostLogic {
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

	return &PostLogic{repo: repo, md: md, policy: policy}
}

// Publish 发布文章：Markdown → HTML → 安全过滤，写入数据库
func (s *PostLogic) Publish(ctx context.Context, req *PublishPostRequest) (*blog.Post, error) {
	if req == nil {
		return nil, errors.New("publish post request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var buf strings.Builder
	if err := s.md.Convert([]byte(req.ContentMD), &buf); err != nil {
		return nil, err
	}

	// XSS 过滤：只保留安全的 HTML 标签和属性
	safeHTML := s.policy.Sanitize(buf.String())

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	title := strings.TrimSpace(req.Title)
	publishedAt := time.Now().UTC()

	p := &blog.Post{
		Slug:        slug,
		Title:       title,
		ContentMD:   req.ContentMD,
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
func (s *PostLogic) GetBySlug(ctx context.Context, slug string) (*blog.Post, error) {
	return s.repo.GetBySlug(ctx, slug)
}

// ListPublished 取已发布文章列表
func (s *PostLogic) ListPublished(ctx context.Context, limit, offset int) ([]blog.Post, error) {
	return s.repo.ListPublished(ctx, limit, offset)
}

func (s *PostLogic) ListPublishedMetadata(ctx context.Context) ([]blog.Post, error) {
	return s.repo.ListPublishedMetadata(ctx)
}
