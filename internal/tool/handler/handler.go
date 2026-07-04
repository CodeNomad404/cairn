package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/tool/service"
	"cairn.cn/internal/web"
)

// Handler 工具模块的 HTTP 处理器
type Handler struct {
	r       *web.Renderer
	jsonSvc *service.JSONService
}

// New 创建 Handler 实例
func New(renderer *web.Renderer, jsonSvc *service.JSONService) *Handler {
	return &Handler{r: renderer, jsonSvc: jsonSvc}
}

// RegisterRoutes 注册工具路由
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/tools", h.index)
	r.Get("/tools/json", h.jsonPage)
	r.Post("/tools/json", h.jsonFormat)
	r.Get("/tools/diff", h.diffPage)
}

func (h *Handler) toolData(r *http.Request, title string, scripts ...string) web.PageData {
	return web.PageData{
		Title:   title,
		Active:  "tools",
		User:    authmid.GetUser(r.Context()),
		Scripts: scripts,
	}
}

// index 工具首页
func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	h.r.Render(w, "tools", struct{ web.PageData }{
		PageData: h.toolData(r, "工具 — Cairn"),
	})
}

// diffPage 文本对比工具页面
func (h *Handler) diffPage(w http.ResponseWriter, r *http.Request) {
	h.r.Render(w, "diff-tool", struct{ web.PageData }{
		PageData: h.toolData(r, "文本对比 — Cairn", "/static/js/vendor/difflib.js", "/static/js/tools/line-numbers.js", "/static/js/tools/diff.js"),
	})
}

// jsonPage JSON 格式化工具页面
func (h *Handler) jsonPage(w http.ResponseWriter, r *http.Request) {
	h.r.Render(w, "json-tool", struct{ web.PageData }{
		PageData: h.toolData(r, "JSON 格式化 — Cairn", "/static/js/vendor/json-formatter.js", "/static/js/tools/line-numbers.js", "/static/js/tools/json.js"),
	})
}

// jsonFormat 处理 JSON 格式化请求
func (h *Handler) jsonFormat(w http.ResponseWriter, r *http.Request) {
	input := r.FormValue("input")
	formatted, err := h.jsonSvc.Format(input)

	if err != nil {
		h.r.RenderStatus(w, http.StatusUnprocessableEntity, "json-result", map[string]interface{}{
			"Error": "JSON 格式错误：" + err.Error(),
		})
		return
	}
	h.r.Render(w, "json-result", map[string]interface{}{
		"Formatted": formatted,
	})
}
