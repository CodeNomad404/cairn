package model

import "time"

type AccountingExpense struct {
	ID          int64
	BookID      int64
	AmountCents int64
	Category    AccountingExpenseCategory
	SpentAt     time.Time
	Note        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AccountingCategoryStatistic struct {
	Category     AccountingExpenseCategory
	AmountCents  int64
	ExpenseCount int64
}

type AccountingMonthlyStatistics struct {
	ByCategory []AccountingCategoryStatistic
	ByBook     []AccountingBookStatistic
}
