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

	"cairn.cn/internal/auth"
	"cairn.cn/internal/blog"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
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
	ctx    context.Context
	svcCtx *svc.ServiceContext
	md     goldmark.Markdown
	policy *bluemonday.Policy
}

// NewPostLogic 创建 PostLogic 实例
func NewPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PostLogic {
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

	return &PostLogic{ctx: ctx, svcCtx: svcCtx, md: md, policy: policy}
}

// Publish 发布文章：Markdown → HTML → 安全过滤，写入数据库
func (l *PostLogic) Publish(req *PublishPostRequest, user *auth.User) (any, response.Code, string) {
	if req == nil {
		return nil, response.CodeBadRequest, "publish post request is required"
	}
	if err := req.Validate(); err != nil {
		return nil, response.CodeBadRequest, err.Error()
	}

	var buf strings.Builder
	if err := l.md.Convert([]byte(req.ContentMD), &buf); err != nil {
		return nil, response.CodeInternalError, "markdown conversion failed"
	}

	safeHTML := l.policy.Sanitize(buf.String())

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
	if err := l.svcCtx.PostRepo.Create(l.ctx, p); err != nil {
		return nil, response.CodeInternalError, "failed to create post"
	}
	_ = user // reserved
	return p, response.CodeCreated, ""
}

// GetBySlug 按 slug 取文章
func (l *PostLogic) GetBySlug(slug string) (any, response.Code, string) {
	post, err := l.svcCtx.PostRepo.GetBySlug(l.ctx, slug)
	if errors.Is(err, blog.ErrPostNotFound) {
		return nil, response.CodeNotFound, "post not found"
	}
	if err != nil {
		return nil, response.CodeInternalError, "failed to load post"
	}
	// Return the domain post so callers (like comment handler) can extract
	// internal fields.  The HTTP handler may convert it to a DTO.
	return post, response.CodeSuccess, ""
}

// ListPublished 取已发布文章列表
func (l *PostLogic) ListPublished(page, pageSize int) (any, response.Code, string) {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, response.CodeBadRequest, "page must be >= 1 and page_size must be between 1 and 100"
	}

	posts, err := l.svcCtx.PostRepo.ListPublished(l.ctx, pageSize+1, (page-1)*pageSize)
	if err != nil {
		return nil, response.CodeInternalError, "failed to load posts"
	}

	hasMore := len(posts) > pageSize
	if hasMore {
		posts = posts[:pageSize]
	}

	type postSummary struct {
		ID          int64      `json:"id"`
		Slug        string     `json:"slug"`
		Title       string     `json:"title"`
		PublishedAt *time.Time `json:"published_at"`
	}
	items := make([]postSummary, len(posts))
	for i, post := range posts {
		items[i] = postSummary{
			ID:          post.ID,
			Slug:        post.Slug,
			Title:       post.Title,
			PublishedAt: post.PublishedAt,
		}
	}

	type postListResponse struct {
		Items    []postSummary `json:"items"`
		Page     int           `json:"page"`
		PageSize int           `json:"page_size"`
		HasMore  bool          `json:"has_more"`
	}
	return postListResponse{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, response.CodeSuccess, ""
}

// ListPublishedMetadata 返回 Sitemap 所需的最小文章元数据（内部方法）。
func (l *PostLogic) ListPublishedMetadata() ([]blog.Post, error) {
	return l.svcCtx.PostRepo.ListPublishedMetadata(l.ctx)
}
