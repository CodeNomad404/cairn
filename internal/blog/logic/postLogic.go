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
	"cairn.cn/internal/httpx"
)

// PostLogic 博客文章相关的业务逻辑
type PostLogic struct {
	postRepo blog.PostRepo
	md       goldmark.Markdown
	policy   *bluemonday.Policy
}

// PostDetailResult is the complete API output for one post.
type PostDetailResult struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	PublishedAt *time.Time `json:"published_at"`
	ContentHTML string     `json:"content_html"`
}

// PostSummaryResult is the API output for a post in a list.
type PostSummaryResult struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	PublishedAt *time.Time `json:"published_at"`
}

// PostListResult is the complete API output for a page of posts.
type PostListResult struct {
	Items    []PostSummaryResult `json:"items"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	HasMore  bool                `json:"has_more"`
}

// NewPostLogic 创建 PostLogic 实例。
// Goldmark 和 Bluemonday Policy 初始化后只读，可安全并发复用，故只在启动时创建一次。
func NewPostLogic(postRepo blog.PostRepo) *PostLogic {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(false),
				),
			),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)

	policy := bluemonday.UGCPolicy().
		AllowAttrs("class", "style").OnElements("span", "pre", "code").
		AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	return &PostLogic{postRepo: postRepo, md: md, policy: policy}
}

// Create 创建文章：Markdown → HTML → 安全过滤，写入数据库
func (l *PostLogic) Create(ctx context.Context, slug, title, contentMD string) (PostDetailResult, httpx.Code, error) {
	if strings.TrimSpace(slug) == "" || strings.TrimSpace(title) == "" || strings.TrimSpace(contentMD) == "" {
		return PostDetailResult{}, httpx.CodeBadRequest, httpx.NewAppError("slug, title, and content_md are required", nil)
	}

	var buf strings.Builder
	if err := l.md.Convert([]byte(contentMD), &buf); err != nil {
		return PostDetailResult{}, httpx.CodeInternalError, httpx.NewAppError("markdown conversion failed", err)
	}

	safeHTML := l.policy.Sanitize(buf.String())

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
	if err := l.postRepo.Create(ctx, p); err != nil {
		return PostDetailResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to create post", err)
	}
	return toPostDetailResult(p), httpx.CodeCreated, nil
}

// Get 按 slug 取文章
func (l *PostLogic) Get(ctx context.Context, slug string) (PostDetailResult, httpx.Code, error) {
	post, err := l.postRepo.GetBySlug(ctx, slug)
	if errors.Is(err, blog.ErrPostNotFound) {
		return PostDetailResult{}, httpx.CodeNotFound, httpx.NewAppError("post not found", err)
	}
	if err != nil {
		return PostDetailResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to load post", err)
	}
	return toPostDetailResult(post), httpx.CodeSuccess, nil
}

// List 取已发布文章列表
func (l *PostLogic) List(ctx context.Context, page, pageSize int) (PostListResult, httpx.Code, error) {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return PostListResult{}, httpx.CodeBadRequest, httpx.NewAppError("page must be >= 1 and page_size must be between 1 and 100", nil)
	}

	posts, err := l.postRepo.ListPublished(ctx, pageSize+1, (page-1)*pageSize)
	if err != nil {
		return PostListResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to load posts", err)
	}

	hasMore := len(posts) > pageSize
	if hasMore {
		posts = posts[:pageSize]
	}

	items := make([]PostSummaryResult, len(posts))
	for i, post := range posts {
		items[i] = PostSummaryResult{
			ID:          post.ID,
			Slug:        post.Slug,
			Title:       post.Title,
			PublishedAt: post.PublishedAt,
		}
	}

	return PostListResult{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, httpx.CodeSuccess, nil
}

func toPostDetailResult(post *blog.Post) PostDetailResult {
	return PostDetailResult{
		ID:          post.ID,
		Slug:        post.Slug,
		Title:       post.Title,
		PublishedAt: post.PublishedAt,
		ContentHTML: post.ContentHTML,
	}
}

// ListPublishedMetadata 返回 Sitemap 所需的最小文章元数据（内部方法）。
func (l *PostLogic) ListPublishedMetadata(ctx context.Context) ([]blog.Post, error) {
	return l.postRepo.ListPublishedMetadata(ctx)
}
