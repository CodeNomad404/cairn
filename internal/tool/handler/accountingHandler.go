package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	toollogic "cairn.cn/internal/tool/logic"
	toolmodel "cairn.cn/internal/tool/model"
)

const requestBodyLimit = 16 << 10

var accountingLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type AccountingHandler struct {
	logic *toollogic.AccountingLogic
}

func NewAccountingHandler(logic *toollogic.AccountingLogic) *AccountingHandler {
	return &AccountingHandler{logic: logic}
}

func (h *AccountingHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireUser).Get("/tool/accounting-book/list", h.listBooks)
	r.With(authmid.RequireUser).Get("/tool/accounting-book/get", h.getBook)
	r.With(authmid.RequireUser).Post("/tool/accounting-book/create", h.createBook)
	r.With(authmid.RequireUser).Post("/tool/accounting-book/update", h.updateBook)
	r.With(authmid.RequireUser).Post("/tool/accounting-book/delete", h.deleteBook)
	r.With(authmid.RequireUser).Get("/tool/accounting-expense/list", h.listExpenses)
	r.With(authmid.RequireUser).Post("/tool/accounting-expense/create", h.createExpense)
	r.With(authmid.RequireUser).Post("/tool/accounting-expense/update", h.updateExpense)
	r.With(authmid.RequireUser).Post("/tool/accounting-expense/delete", h.deleteExpense)
	r.With(authmid.RequireUser).Get("/tool/accounting-statistic/get", h.getMonthlyStatistic)
}

func (h *AccountingHandler) listBooks(w http.ResponseWriter, r *http.Request) {
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.ListBooks(r.Context(), principal.ID)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *AccountingHandler) getBook(w http.ResponseWriter, r *http.Request) {
	bookID, err := httpx.ParsePositiveInt64Query(r, "id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.GetBook(r.Context(), principal.ID, bookID)
	httpx.Reply(r.Context(), w, data, code, err)
}

type CreateBookRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r CreateBookRequest) Validate() error {
	return validateBookInput(r.Name, r.Description)
}

func (h *AccountingHandler) createBook(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[CreateBookRequest](w, r, requestBodyLimit)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.CreateBook(r.Context(), principal.ID, req.Name, req.Description)
	httpx.Reply(r.Context(), w, data, code, err)
}

type UpdateBookRequest struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r UpdateBookRequest) Validate() error {
	if r.ID < 1 {
		return errors.New("id must be positive")
	}
	return validateBookInput(r.Name, r.Description)
}

func (h *AccountingHandler) updateBook(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[UpdateBookRequest](w, r, requestBodyLimit)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.UpdateBook(r.Context(), principal.ID, req.ID, req.Name, req.Description)
	httpx.Reply(r.Context(), w, data, code, err)
}

type DeleteBookRequest struct {
	ID int64 `json:"id"`
}

func (r DeleteBookRequest) Validate() error {
	if r.ID < 1 {
		return errors.New("id must be positive")
	}
	return nil
}

func (h *AccountingHandler) deleteBook(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[DeleteBookRequest](w, r, requestBodyLimit)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	code, err := h.logic.DeleteBook(r.Context(), principal.ID, req.ID)
	httpx.Reply(r.Context(), w, nil, code, err)
}

func (h *AccountingHandler) listExpenses(w http.ResponseWriter, r *http.Request) {
	bookID, err := httpx.ParsePositiveInt64Query(r, "book_id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	pagination, err := httpx.ParsePagination(r)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.ListExpenses(
		r.Context(), principal.ID, bookID, pagination.Page, pagination.PageSize,
	)
	httpx.Reply(r.Context(), w, data, code, err)
}

type CreateExpenseRequest struct {
	BookID      int64                               `json:"book_id"`
	AmountCents int64                               `json:"amount_cents"`
	Category    toolmodel.AccountingExpenseCategory `json:"category"`
	SpentAt     time.Time                           `json:"spent_at"`
	Note        string                              `json:"note"`
}

func (r CreateExpenseRequest) Validate() error {
	if r.BookID < 1 {
		return errors.New("book_id must be positive")
	}
	return validateExpenseInput(r.AmountCents, r.Category, r.SpentAt, r.Note)
}

func (h *AccountingHandler) createExpense(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[CreateExpenseRequest](w, r, requestBodyLimit)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.CreateExpense(
		r.Context(), principal.ID, req.BookID, req.AmountCents, req.Category, req.SpentAt, req.Note,
	)
	httpx.Reply(r.Context(), w, data, code, err)
}

type UpdateExpenseRequest struct {
	ID          int64                               `json:"id"`
	AmountCents int64                               `json:"amount_cents"`
	Category    toolmodel.AccountingExpenseCategory `json:"category"`
	SpentAt     time.Time                           `json:"spent_at"`
	Note        string                              `json:"note"`
}

func (r UpdateExpenseRequest) Validate() error {
	if r.ID < 1 {
		return errors.New("id must be positive")
	}
	return validateExpenseInput(r.AmountCents, r.Category, r.SpentAt, r.Note)
}

func (h *AccountingHandler) updateExpense(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[UpdateExpenseRequest](w, r, requestBodyLimit)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.UpdateExpense(
		r.Context(), principal.ID, req.ID, req.AmountCents, req.Category, req.SpentAt, req.Note,
	)
	httpx.Reply(r.Context(), w, data, code, err)
}

type DeleteExpenseRequest struct {
	ID int64 `json:"id"`
}

func (r DeleteExpenseRequest) Validate() error {
	if r.ID < 1 {
		return errors.New("id must be positive")
	}
	return nil
}

func (h *AccountingHandler) deleteExpense(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[DeleteExpenseRequest](w, r, requestBodyLimit)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	code, err := h.logic.DeleteExpense(r.Context(), principal.ID, req.ID)
	httpx.Reply(r.Context(), w, nil, code, err)
}

func (h *AccountingHandler) getMonthlyStatistic(w http.ResponseWriter, r *http.Request) {
	month, err := httpx.ParseQueryMonth(r, "month", time.Now().In(accountingLocation))
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.logic.GetMonthlyStatistics(r.Context(), principal.ID, month)
	httpx.Reply(r.Context(), w, data, code, err)
}

func validateBookInput(name, description string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(name)) > 100 {
		return errors.New("name must not exceed 100 characters")
	}
	if utf8.RuneCountInString(strings.TrimSpace(description)) > 500 {
		return errors.New("description must not exceed 500 characters")
	}
	return nil
}

func validateExpenseInput(
	amountCents int64,
	category toolmodel.AccountingExpenseCategory,
	spentAt time.Time,
	note string,
) error {
	if amountCents < 1 {
		return errors.New("amount_cents must be positive")
	}
	if !category.Valid() {
		return errors.New("invalid category")
	}
	if spentAt.IsZero() {
		return errors.New("spent_at is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(note)) > 200 {
		return errors.New("note must not exceed 200 characters")
	}
	return nil
}
