package logic

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

var shanghaiLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type AccountingLogic struct {
	repo tool.AccountingRepo
}

type BookResult struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BookSummaryResult struct {
	BookResult
	TotalAmountCents int64      `json:"total_amount_cents"`
	ExpenseCount     int64      `json:"expense_count"`
	LastExpenseAt    *time.Time `json:"last_expense_at,omitempty"`
}

type CategoryStatisticResult struct {
	Category     toolmodel.AccountingExpenseCategory `json:"category"`
	AmountCents  int64                               `json:"amount_cents"`
	ExpenseCount int64                               `json:"expense_count"`
}

type BookStatisticResult struct {
	BookID       int64  `json:"book_id"`
	BookName     string `json:"book_name"`
	AmountCents  int64  `json:"amount_cents"`
	ExpenseCount int64  `json:"expense_count"`
}

type BookListResult struct {
	Items []BookSummaryResult `json:"items"`
}

type BookDetailResult struct {
	BookSummaryResult
	Categories []CategoryStatisticResult `json:"categories"`
}

type ExpenseResult struct {
	ID          int64                               `json:"id"`
	BookID      int64                               `json:"book_id"`
	AmountCents int64                               `json:"amount_cents"`
	Category    toolmodel.AccountingExpenseCategory `json:"category"`
	SpentAt     time.Time                           `json:"spent_at"`
	Note        string                              `json:"note"`
	CreatedAt   time.Time                           `json:"created_at"`
	UpdatedAt   time.Time                           `json:"updated_at"`
}

type ExpenseListResult struct {
	Items    []ExpenseResult `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	HasMore  bool            `json:"has_more"`
}

type MonthlyStatisticResult struct {
	Month            string                    `json:"month"`
	TotalAmountCents int64                     `json:"total_amount_cents"`
	ByCategory       []CategoryStatisticResult `json:"by_category"`
	ByBook           []BookStatisticResult     `json:"by_book"`
}

func NewAccountingLogic(repo tool.AccountingRepo) *AccountingLogic {
	return &AccountingLogic{repo: repo}
}

func (l *AccountingLogic) CreateBook(
	ctx context.Context,
	userID int64,
	name, description string,
) (BookResult, httpx.Code, error) {
	name, description, err := normalizeBookInput(name, description)
	if err != nil {
		return BookResult{}, httpx.CodeBadRequest, err
	}
	book := &toolmodel.AccountingBook{UserID: userID, Name: name, Description: description}
	if err := l.repo.CreateBook(ctx, book); err != nil {
		return BookResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to create accounting book", err)
	}
	return toBookResult(book), httpx.CodeCreated, nil
}

func (l *AccountingLogic) ListBooks(ctx context.Context, userID int64) (BookListResult, httpx.Code, error) {
	books, err := l.repo.ListBooks(ctx, userID)
	if err != nil {
		return BookListResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to load accounting books", err)
	}
	items := make([]BookSummaryResult, len(books))
	for i := range books {
		items[i] = toBookSummaryResult(&books[i])
	}
	return BookListResult{Items: items}, httpx.CodeSuccess, nil
}

func (l *AccountingLogic) GetBook(ctx context.Context, userID, bookID int64) (BookDetailResult, httpx.Code, error) {
	if bookID < 1 {
		return BookDetailResult{}, httpx.CodeBadRequest, httpx.NewAppError("invalid book id", nil)
	}
	book, err := l.repo.GetBook(ctx, userID, bookID)
	if err != nil {
		code, appErr := bookError("failed to load accounting book", err)
		return BookDetailResult{}, code, appErr
	}
	categories, err := l.repo.ListBookCategoryStatistics(ctx, userID, bookID)
	if err != nil {
		return BookDetailResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to load category statistics", err)
	}
	return BookDetailResult{
		BookSummaryResult: toBookSummaryResult(book),
		Categories:        toCategoryStatisticResults(categories),
	}, httpx.CodeSuccess, nil
}

func (l *AccountingLogic) UpdateBook(
	ctx context.Context,
	userID, bookID int64,
	name, description string,
) (BookResult, httpx.Code, error) {
	if bookID < 1 {
		return BookResult{}, httpx.CodeBadRequest, httpx.NewAppError("invalid book id", nil)
	}
	name, description, err := normalizeBookInput(name, description)
	if err != nil {
		return BookResult{}, httpx.CodeBadRequest, err
	}
	book := &toolmodel.AccountingBook{ID: bookID, UserID: userID, Name: name, Description: description}
	if err := l.repo.UpdateBook(ctx, book); err != nil {
		if errors.Is(err, tool.ErrAccountingBookNotFound) {
			return BookResult{}, httpx.CodeNotFound, httpx.NewAppError("accounting book not found", err)
		}
		return BookResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to update accounting book", err)
	}
	return toBookResult(book), httpx.CodeSuccess, nil
}

func (l *AccountingLogic) DeleteBook(ctx context.Context, userID, bookID int64) (httpx.Code, error) {
	if bookID < 1 {
		return httpx.CodeBadRequest, httpx.NewAppError("invalid book id", nil)
	}
	if err := l.repo.DeleteBook(ctx, userID, bookID); err != nil {
		if errors.Is(err, tool.ErrAccountingBookNotFound) {
			return httpx.CodeNotFound, httpx.NewAppError("accounting book not found", err)
		}
		return httpx.CodeInternalError, httpx.NewAppError("failed to delete accounting book", err)
	}
	return httpx.CodeNoContent, nil
}

func (l *AccountingLogic) CreateExpense(
	ctx context.Context,
	userID, bookID, amountCents int64,
	category toolmodel.AccountingExpenseCategory,
	spentAt time.Time,
	note string,
) (ExpenseResult, httpx.Code, error) {
	note, err := validateExpenseInput(bookID, amountCents, category, spentAt, note)
	if err != nil {
		return ExpenseResult{}, httpx.CodeBadRequest, err
	}
	expense := &toolmodel.AccountingExpense{
		BookID: bookID, AmountCents: amountCents, Category: category, SpentAt: spentAt, Note: note,
	}
	if err := l.repo.CreateExpense(ctx, userID, expense); err != nil {
		if errors.Is(err, tool.ErrAccountingBookNotFound) {
			return ExpenseResult{}, httpx.CodeNotFound, httpx.NewAppError("accounting book not found", err)
		}
		return ExpenseResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to create expense", err)
	}
	return toExpenseResult(expense), httpx.CodeCreated, nil
}

func (l *AccountingLogic) ListExpenses(
	ctx context.Context,
	userID, bookID int64,
	category toolmodel.AccountingExpenseCategory,
	page, pageSize int,
) (ExpenseListResult, httpx.Code, error) {
	if bookID < 1 || (category != 0 && !category.Valid()) || page < 1 || pageSize < 1 || pageSize > 100 {
		return ExpenseListResult{}, httpx.CodeBadRequest, httpx.NewAppError("invalid expense list parameters", nil)
	}
	if _, err := l.repo.GetBook(ctx, userID, bookID); err != nil {
		code, appErr := bookError("failed to load accounting book", err)
		return ExpenseListResult{}, code, appErr
	}
	expenses, err := l.repo.ListExpenses(ctx, userID, bookID, category, pageSize+1, (page-1)*pageSize)
	if err != nil {
		return ExpenseListResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to load expenses", err)
	}
	hasMore := len(expenses) > pageSize
	if hasMore {
		expenses = expenses[:pageSize]
	}
	items := make([]ExpenseResult, len(expenses))
	for i := range expenses {
		items[i] = toExpenseResult(&expenses[i])
	}
	return ExpenseListResult{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore}, httpx.CodeSuccess, nil
}

func (l *AccountingLogic) UpdateExpense(
	ctx context.Context,
	userID, expenseID, amountCents int64,
	category toolmodel.AccountingExpenseCategory,
	spentAt time.Time,
	note string,
) (ExpenseResult, httpx.Code, error) {
	if expenseID < 1 {
		return ExpenseResult{}, httpx.CodeBadRequest, httpx.NewAppError("invalid expense id", nil)
	}
	note, err := validateExpenseInput(1, amountCents, category, spentAt, note)
	if err != nil {
		return ExpenseResult{}, httpx.CodeBadRequest, err
	}
	expense := &toolmodel.AccountingExpense{
		ID: expenseID, AmountCents: amountCents, Category: category, SpentAt: spentAt, Note: note,
	}
	if err := l.repo.UpdateExpense(ctx, userID, expense); err != nil {
		if errors.Is(err, tool.ErrAccountingExpenseNotFound) {
			return ExpenseResult{}, httpx.CodeNotFound, httpx.NewAppError("expense not found", err)
		}
		return ExpenseResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to update expense", err)
	}
	return toExpenseResult(expense), httpx.CodeSuccess, nil
}

func (l *AccountingLogic) DeleteExpense(ctx context.Context, userID, expenseID int64) (httpx.Code, error) {
	if expenseID < 1 {
		return httpx.CodeBadRequest, httpx.NewAppError("invalid expense id", nil)
	}
	if err := l.repo.DeleteExpense(ctx, userID, expenseID); err != nil {
		if errors.Is(err, tool.ErrAccountingExpenseNotFound) {
			return httpx.CodeNotFound, httpx.NewAppError("expense not found", err)
		}
		return httpx.CodeInternalError, httpx.NewAppError("failed to delete expense", err)
	}
	return httpx.CodeNoContent, nil
}

func (l *AccountingLogic) GetMonthlyStatistics(
	ctx context.Context,
	userID int64,
	month time.Time,
) (MonthlyStatisticResult, httpx.Code, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, shanghaiLocation)
	end := start.AddDate(0, 1, 0)
	statistics, err := l.repo.GetMonthlyStatistics(ctx, userID, start, end)
	if err != nil {
		return MonthlyStatisticResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to load monthly statistics", err)
	}
	byBook := make([]BookStatisticResult, len(statistics.ByBook))
	for i, item := range statistics.ByBook {
		byBook[i] = BookStatisticResult{
			BookID: item.BookID, BookName: item.BookName,
			AmountCents: item.AmountCents, ExpenseCount: item.ExpenseCount,
		}
	}
	byCategory := toCategoryStatisticResults(statistics.ByCategory)
	var total int64
	for _, item := range byCategory {
		total += item.AmountCents
	}
	return MonthlyStatisticResult{
		Month: start.Format("2006-01"), TotalAmountCents: total,
		ByCategory: byCategory, ByBook: byBook,
	}, httpx.CodeSuccess, nil
}

func normalizeBookInput(name, description string) (string, string, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return "", "", httpx.NewAppError("name is required", nil)
	}
	if utf8.RuneCountInString(name) > 100 {
		return "", "", httpx.NewAppError("name must not exceed 100 characters", nil)
	}
	if utf8.RuneCountInString(description) > 500 {
		return "", "", httpx.NewAppError("description must not exceed 500 characters", nil)
	}
	return name, description, nil
}

func validateExpenseInput(
	bookID, amountCents int64,
	category toolmodel.AccountingExpenseCategory,
	spentAt time.Time,
	note string,
) (string, error) {
	if bookID < 1 {
		return "", httpx.NewAppError("book_id must be positive", nil)
	}
	if amountCents < 1 {
		return "", httpx.NewAppError("amount_cents must be positive", nil)
	}
	if !category.Valid() {
		return "", httpx.NewAppError("invalid category", nil)
	}
	if spentAt.IsZero() {
		return "", httpx.NewAppError("spent_at is required", nil)
	}
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > 200 {
		return "", httpx.NewAppError("note must not exceed 200 characters", nil)
	}
	return note, nil
}

func bookError(message string, err error) (httpx.Code, error) {
	if errors.Is(err, tool.ErrAccountingBookNotFound) {
		return httpx.CodeNotFound, httpx.NewAppError("accounting book not found", err)
	}
	return httpx.CodeInternalError, httpx.NewAppError(message, err)
}

func toBookResult(book *toolmodel.AccountingBook) BookResult {
	return BookResult{
		ID: book.ID, Name: book.Name, Description: book.Description,
		CreatedAt: book.CreatedAt, UpdatedAt: book.UpdatedAt,
	}
}

func toBookSummaryResult(book *toolmodel.AccountingBookSummary) BookSummaryResult {
	return BookSummaryResult{
		BookResult:       toBookResult(&book.AccountingBook),
		TotalAmountCents: book.TotalAmountCents,
		ExpenseCount:     book.ExpenseCount,
		LastExpenseAt:    book.LastExpenseAt,
	}
}

func toCategoryStatisticResults(items []toolmodel.AccountingCategoryStatistic) []CategoryStatisticResult {
	results := make([]CategoryStatisticResult, len(items))
	for i, item := range items {
		results[i] = CategoryStatisticResult{
			Category: item.Category, AmountCents: item.AmountCents, ExpenseCount: item.ExpenseCount,
		}
	}
	return results
}

func toExpenseResult(expense *toolmodel.AccountingExpense) ExpenseResult {
	return ExpenseResult{
		ID: expense.ID, BookID: expense.BookID, AmountCents: expense.AmountCents,
		Category: expense.Category, SpentAt: expense.SpentAt, Note: expense.Note,
		CreatedAt: expense.CreatedAt, UpdatedAt: expense.UpdatedAt,
	}
}
