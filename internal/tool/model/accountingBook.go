package model

import "time"

type AccountingBook struct {
	ID          int64
	UserID      int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AccountingBookSummary struct {
	AccountingBook
	TotalAmountCents int64
	ExpenseCount     int64
	LastExpenseAt    *time.Time
}

type AccountingBookStatistic struct {
	BookID       int64
	BookName     string
	AmountCents  int64
	ExpenseCount int64
}
