package seo

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"

	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/web"
)

// Handler serves SEO endpoints (robots.txt, sitemap.xml).
type Handler struct {
	postLogic *bloglogic.PostLogic
}

func New(postLogic *bloglogic.PostLogic) *Handler {
	return &Handler{postLogic: postLogic}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/robots.txt", h.robots)
	r.Get("/sitemap.xml", h.sitemap)
}

func (h *Handler) robots(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte("User-agent: *\nDisallow: /upload\nDisallow: /healthz\n\nSitemap: " + web.CanonicalURL("/sitemap.xml") + "\n"))
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
	LastMod  string `xml:"lastmod,omitempty"`
}

func (h *Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	posts, err := h.postLogic.ListPublishedMetadata(r.Context())
	if err != nil {
		http.Error(w, "failed to build sitemap", http.StatusInternalServerError)
		return
	}

	paths := []string{"/", "/posts", "/tools", "/tools/json", "/tools/diff"}
	urls := make([]sitemapURL, 0, len(paths)+len(posts))
	for _, path := range paths {
		urls = append(urls, sitemapURL{Location: web.CanonicalURL(path)})
	}
	for _, post := range posts {
		urls = append(urls, sitemapURL{
			Location: web.CanonicalURL("/posts/" + url.PathEscape(post.Slug)),
			LastMod:  post.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	payload, err := xml.MarshalIndent(urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}, "", "  ")
	if err != nil {
		http.Error(w, "failed to build sitemap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(payload)
}
