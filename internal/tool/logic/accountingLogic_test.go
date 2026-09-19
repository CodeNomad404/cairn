package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type fakeAccountingRepo struct {
	getBook              func(context.Context, int64, int64) (*toolmodel.AccountingBookSummary, error)
	listExpenses         func(context.Context, int64, int64, toolmodel.AccountingExpenseCategory, int, int) ([]toolmodel.AccountingExpense, error)
	getMonthlyStatistics func(context.Context, int64, time.Time, time.Time) (toolmodel.AccountingMonthlyStatistics, error)
}

func (f *fakeAccountingRepo) CreateBook(context.Context, *toolmodel.AccountingBook) error {
	return nil
}

func (f *fakeAccountingRepo) GetBook(
	ctx context.Context,
	userID, bookID int64,
) (*toolmodel.AccountingBookSummary, error) {
	if f.getBook != nil {
		return f.getBook(ctx, userID, bookID)
	}
	return &toolmodel.AccountingBookSummary{}, nil
}

func (f *fakeAccountingRepo) ListBooks(context.Context, int64) ([]toolmodel.AccountingBookSummary, error) {
	return nil, nil
}

func (f *fakeAccountingRepo) UpdateBook(context.Context, *toolmodel.AccountingBook) error {
	return nil
}

func (f *fakeAccountingRepo) DeleteBook(context.Context, int64, int64) error {
	return nil
}

func (f *fakeAccountingRepo) ListBookCategoryStatistics(
	context.Context,
	int64,
	int64,
) ([]toolmodel.AccountingCategoryStatistic, error) {
	return nil, nil
}

func (f *fakeAccountingRepo) CreateExpense(context.Context, int64, *toolmodel.AccountingExpense) error {
	return nil
}

func (f *fakeAccountingRepo) ListExpenses(
	ctx context.Context,
	userID, bookID int64,
	category toolmodel.AccountingExpenseCategory,
	limit, offset int,
) ([]toolmodel.AccountingExpense, error) {
	if f.listExpenses != nil {
		return f.listExpenses(ctx, userID, bookID, category, limit, offset)
	}
	return nil, nil
}

func (f *fakeAccountingRepo) UpdateExpense(context.Context, int64, *toolmodel.AccountingExpense) error {
	return nil
}

func (f *fakeAccountingRepo) DeleteExpense(context.Context, int64, int64) error {
	return nil
}

func (f *fakeAccountingRepo) GetMonthlyStatistics(
	ctx context.Context,
	userID int64,
	start, end time.Time,
) (toolmodel.AccountingMonthlyStatistics, error) {
	if f.getMonthlyStatistics != nil {
		return f.getMonthlyStatistics(ctx, userID, start, end)
	}
	return toolmodel.AccountingMonthlyStatistics{}, nil
}

func TestListExpensesRejectsAnotherUsersBook(t *testing.T) {
	listCalled := false
	repo := &fakeAccountingRepo{
		getBook: func(context.Context, int64, int64) (*toolmodel.AccountingBookSummary, error) {
			return nil, tool.ErrAccountingBookNotFound
		},
		listExpenses: func(context.Context, int64, int64, toolmodel.AccountingExpenseCategory, int, int) ([]toolmodel.AccountingExpense, error) {
			listCalled = true
			return nil, nil
		},
	}

	_, code, err := NewAccountingLogic(repo).ListExpenses(context.Background(), 7, 9, 0, 1, 20)
	if code != httpx.CodeNotFound || err == nil || !errors.Is(err, tool.ErrAccountingBookNotFound) {
		t.Fatalf("code = %d, err = %v", code, err)
	}
	if listCalled {
		t.Fatal("expenses queried after book ownership check failed")
	}
}

func TestListExpensesPassesCategoryToRepository(t *testing.T) {
	repo := &fakeAccountingRepo{
		listExpenses: func(
			_ context.Context,
			userID, bookID int64,
			category toolmodel.AccountingExpenseCategory,
			limit, offset int,
		) ([]toolmodel.AccountingExpense, error) {
			if userID != 7 || bookID != 9 {
				t.Fatalf("user ID = %d, book ID = %d", userID, bookID)
			}
			if category != toolmodel.AccountingExpenseCategoryFood {
				t.Fatalf("category = %d", category)
			}
			if limit != 21 || offset != 20 {
				t.Fatalf("limit = %d, offset = %d", limit, offset)
			}
			return nil, nil
		},
	}

	result, code, err := NewAccountingLogic(repo).ListExpenses(
		context.Background(), 7, 9, toolmodel.AccountingExpenseCategoryFood, 2, 20,
	)
	if err != nil || code != httpx.CodeSuccess {
		t.Fatalf("code = %d, err = %v", code, err)
	}
	if result.Page != 2 || result.PageSize != 20 || result.HasMore {
		t.Fatalf("result = %+v", result)
	}
}

func TestGetMonthlyStatisticsUsesShanghaiMonthBoundaries(t *testing.T) {
	repo := &fakeAccountingRepo{
		getMonthlyStatistics: func(
			_ context.Context,
			userID int64,
			start, end time.Time,
		) (toolmodel.AccountingMonthlyStatistics, error) {
			if userID != 7 {
				t.Fatalf("user ID = %d", userID)
			}
			if got := start.Format(time.RFC3339); got != "2026-09-01T00:00:00+08:00" {
				t.Fatalf("start = %s", got)
			}
			if got := end.Format(time.RFC3339); got != "2026-10-01T00:00:00+08:00" {
				t.Fatalf("end = %s", got)
			}
			return toolmodel.AccountingMonthlyStatistics{
				ByCategory: []toolmodel.AccountingCategoryStatistic{
					{Category: toolmodel.AccountingExpenseCategoryFood, AmountCents: 1200, ExpenseCount: 2},
					{Category: toolmodel.AccountingExpenseCategoryTransport, AmountCents: 800, ExpenseCount: 1},
				},
				ByBook: []toolmodel.AccountingBookStatistic{
					{BookID: 3, BookName: "云南旅游", AmountCents: 2000, ExpenseCount: 3},
				},
			}, nil
		},
	}

	month := time.Date(2026, time.September, 19, 18, 0, 0, 0, time.UTC)
	result, code, err := NewAccountingLogic(repo).GetMonthlyStatistics(context.Background(), 7, month)
	if err != nil || code != httpx.CodeSuccess {
		t.Fatalf("code = %d, err = %v", code, err)
	}
	if result.Month != "2026-09" || result.TotalAmountCents != 2000 {
		t.Fatalf("result = %+v", result)
	}
	if len(result.ByCategory) != 2 || len(result.ByBook) != 1 {
		t.Fatalf("result = %+v", result)
	}
}
