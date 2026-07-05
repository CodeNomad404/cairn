package web

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"

	"cairn.cn/internal/auth"
)

// PageData is shared by full-page templates.
type PageData struct {
	Title         string
	Description   string
	Canonical     string
	Robots        string
	OpenGraphType string
	Active        string
	Page          string
	User          *auth.User
	Scripts       []string
	BodyClass     string
}

const SiteBaseURL = "https://cairn.cn"

func CanonicalURL(path string) string {
	return SiteBaseURL + path
}

func PublicPageData(title, description, path string) PageData {
	return PageData{
		Title:         title,
		Description:   description,
		Canonical:     CanonicalURL(path),
		Robots:        "index,follow",
		OpenGraphType: "website",
	}
}

func PrivatePageData(title string) PageData {
	return PageData{Title: title, Robots: "noindex,nofollow"}
}

// Renderer centralizes template execution and HTTP response details.
type Renderer struct {
	tpl    *template.Template
	logger *slog.Logger
}

func NewRenderer(pattern string, logger *slog.Logger) (*Renderer, error) {
	funcMap := template.FuncMap{
		"firstChar": func(s string) string {
			for _, r := range s {
				return string(r)
			}
			return ""
		},
		"pathEscape": url.PathEscape,
	}

	tpl, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	return &Renderer{tpl: tpl, logger: logger}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data any) {
	r.RenderStatus(w, http.StatusOK, name, data)
}

func (r *Renderer) RenderStatus(w http.ResponseWriter, status int, name string, data any) {
	var buf bytes.Buffer
	if err := r.tpl.ExecuteTemplate(&buf, name, data); err != nil {
		r.logger.Error("render template", "template", name, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
