package handler

import (
	"errors"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toollogic "cairn.cn/internal/tool/logic"
)

type TransferHandler struct{ logic *toollogic.TransferLogic }

func NewTransferHandler(logic *toollogic.TransferLogic) *TransferHandler {
	return &TransferHandler{logic: logic}
}

func (h *TransferHandler) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("X-Robots-Tag", "noindex, nofollow")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				next.ServeHTTP(w, r)
			})
		})
		r.Use(authmid.RequireRole(auth.RoleAdmin))
		r.Post("/tool/transfer-channel/create", h.createChannel)
		r.Get("/tool/transfer-channel/list", h.listChannels)
		r.Post("/tool/transfer-channel/update", h.renameChannel)
		r.Post("/tool/transfer-channel/delete", h.deleteChannel)
		r.Post("/tool/transfer/create", h.createText)
		r.Post("/tool/transfer/upload", h.upload)
		r.Get("/tool/transfer/list", h.listItems)
		r.Get("/tool/transfer/get", h.getItem)
		r.Get("/tool/transfer/download", h.download)
		r.Post("/tool/transfer/delete", h.deleteItem)
		r.Get("/tool/transfer-usage/get", h.usage)
	})
}

type CreateTransferChannelRequest struct {
	Name string `json:"name"`
}

func (r CreateTransferChannelRequest) Validate() error {
	name := strings.TrimSpace(r.Name)
	if name == "" || utf8.RuneCountInString(name) > 50 || strings.ContainsRune(name, 0) {
		return errors.New("通道名称须为 1–50 个字符")
	}
	return nil
}
func (h *TransferHandler) createChannel(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[CreateTransferChannelRequest](w, r, 4096)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.CreateChannel(r.Context(), authmid.GetPrincipal(r.Context()).ID, req.Name)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *TransferHandler) listChannels(w http.ResponseWriter, r *http.Request) {
	data, code, err := h.logic.ListChannels(r.Context(), authmid.GetPrincipal(r.Context()).ID)
	httpx.Reply(r.Context(), w, data, code, err)
}

type UpdateTransferChannelRequest struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (r UpdateTransferChannelRequest) Validate() error {
	if r.ID < 1 {
		return errors.New("id must be positive")
	}
	name := strings.TrimSpace(r.Name)
	if name == "" || utf8.RuneCountInString(name) > 50 || strings.ContainsRune(name, 0) {
		return errors.New("通道名称须为 1–50 个字符")
	}
	return nil
}
func (h *TransferHandler) renameChannel(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[UpdateTransferChannelRequest](w, r, 4096)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.RenameChannel(r.Context(), authmid.GetPrincipal(r.Context()).ID, req.ID, req.Name)
	httpx.Reply(r.Context(), w, data, code, err)
}

type DeleteTransferChannelRequest struct {
	ID int64 `json:"id"`
}

func (r DeleteTransferChannelRequest) Validate() error {
	if r.ID < 1 {
		return errors.New("id must be positive")
	}
	return nil
}
func (h *TransferHandler) deleteChannel(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[DeleteTransferChannelRequest](w, r, 4096)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	code, err := h.logic.DeleteChannel(r.Context(), authmid.GetPrincipal(r.Context()).ID, req.ID)
	httpx.Reply(r.Context(), w, nil, code, err)
}

type CreateTransferTextRequest struct {
	ChannelID int64  `json:"channel_id"`
	Text      string `json:"text"`
}

func (r CreateTransferTextRequest) Validate() error {
	if r.ChannelID < 1 {
		return errors.New("channel_id must be positive")
	}
	if strings.TrimSpace(r.Text) == "" || len(r.Text) > tool.TransferTextLimit || strings.ContainsRune(r.Text, 0) {
		return errors.New("文本不能为空且不能超过 100 KiB")
	}
	return nil
}
func (h *TransferHandler) createText(w http.ResponseWriter, r *http.Request) {
	// JSON escapes can require six bytes per original ASCII control character.
	req, err := httpx.ParseBody[CreateTransferTextRequest](w, r, tool.TransferTextLimit*6+4096)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.CreateText(r.Context(), authmid.GetPrincipal(r.Context()).ID, req.ChannelID, req.Text)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *TransferHandler) upload(w http.ResponseWriter, r *http.Request) {
	upload, err := httpx.ParseFileUpload(w, r, tool.TransferFileLimit, "channel_id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	channelID, err := upload.PositiveInt64("channel_id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.Upload(r.Context(), authmid.GetPrincipal(r.Context()).ID, channelID, upload.File.FileName(), upload, upload.Finish)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *TransferHandler) listItems(w http.ResponseWriter, r *http.Request) {
	channelID, err := httpx.ParsePositiveInt64Query(r, "channel_id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	pagination, err := httpx.ParsePagination(r)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	kind, err := httpx.ParseQueryString(r, "kind", 4)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	if kind != "" && kind != "text" && kind != "file" {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, httpx.NewAppError("kind 必须为 text 或 file", nil))
		return
	}
	data, code, err := h.logic.ListItems(r.Context(), authmid.GetPrincipal(r.Context()).ID, channelID, kind, pagination.Page, pagination.PageSize)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *TransferHandler) getItem(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.ParsePositiveInt64Query(r, "id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.GetItem(r.Context(), authmid.GetPrincipal(r.Context()).ID, id)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *TransferHandler) download(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.ParsePositiveInt64Query(r, "id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.Download(r.Context(), authmid.GetPrincipal(r.Context()).ID, id)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, code, err)
		return
	}
	defer func() { _ = data.File.Close() }()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": data.Filename}))
	http.ServeContent(w, r, data.Filename, data.CreatedAt, data.File)
}

type DeleteTransferItemRequest struct {
	ID int64 `json:"id"`
}

func (r DeleteTransferItemRequest) Validate() error {
	if r.ID < 1 {
		return errors.New("id must be positive")
	}
	return nil
}
func (h *TransferHandler) deleteItem(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[DeleteTransferItemRequest](w, r, 4096)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	code, err := h.logic.DeleteItem(r.Context(), authmid.GetPrincipal(r.Context()).ID, req.ID)
	httpx.Reply(r.Context(), w, nil, code, err)
}

func (h *TransferHandler) usage(w http.ResponseWriter, r *http.Request) {
	data, code, err := h.logic.Usage(r.Context(), authmid.GetPrincipal(r.Context()).ID)
	httpx.Reply(r.Context(), w, data, code, err)
}
