package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	tool "cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type AccountingRepository struct {
	db *pgxpool.Pool
}

func NewAccountingRepo(db *pgxpool.Pool) *AccountingRepository {
	return &AccountingRepository{db: db}
}

func (r *AccountingRepository) CreateBook(ctx context.Context, book *toolmodel.AccountingBook) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO accounting_books (user_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, book.UserID, book.Name, book.Description).Scan(
		&book.ID, &book.CreatedAt, &book.UpdatedAt,
	)
}

func (r *AccountingRepository) GetBook(ctx context.Context, userID, bookID int64) (*toolmodel.AccountingBookSummary, error) {
	var book toolmodel.AccountingBookSummary
	err := r.db.QueryRow(ctx, `
		SELECT b.id, b.user_id, b.name, b.description, b.created_at, b.updated_at,
		       COALESCE(SUM(e.amount_cents), 0)::BIGINT,
		       COUNT(e.id)::BIGINT,
		       MAX(e.spent_at)
		FROM accounting_books AS b
		LEFT JOIN accounting_expenses AS e ON e.book_id = b.id
		WHERE b.id = $1 AND b.user_id = $2
		GROUP BY b.id
	`, bookID, userID).Scan(
		&book.ID, &book.UserID, &book.Name, &book.Description, &book.CreatedAt, &book.UpdatedAt,
		&book.TotalAmountCents, &book.ExpenseCount, &book.LastExpenseAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", tool.ErrAccountingBookNotFound, bookID)
	}
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *AccountingRepository) ListBooks(ctx context.Context, userID int64) ([]toolmodel.AccountingBookSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.user_id, b.name, b.description, b.created_at, b.updated_at,
		       COALESCE(SUM(e.amount_cents), 0)::BIGINT,
		       COUNT(e.id)::BIGINT,
		       MAX(e.spent_at)
		FROM accounting_books AS b
		LEFT JOIN accounting_expenses AS e ON e.book_id = b.id
		WHERE b.user_id = $1
		GROUP BY b.id
		ORDER BY MAX(e.spent_at) DESC NULLS LAST, b.created_at DESC, b.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := make([]toolmodel.AccountingBookSummary, 0)
	for rows.Next() {
		var book toolmodel.AccountingBookSummary
		if err := rows.Scan(
			&book.ID, &book.UserID, &book.Name, &book.Description, &book.CreatedAt, &book.UpdatedAt,
			&book.TotalAmountCents, &book.ExpenseCount, &book.LastExpenseAt,
		); err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	return books, rows.Err()
}

func (r *AccountingRepository) UpdateBook(ctx context.Context, book *toolmodel.AccountingBook) error {
	err := r.db.QueryRow(ctx, `
		UPDATE accounting_books
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING created_at, updated_at
	`, book.Name, book.Description, book.ID, book.UserID).Scan(&book.CreatedAt, &book.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %d", tool.ErrAccountingBookNotFound, book.ID)
	}
	return err
}

func (r *AccountingRepository) DeleteBook(ctx context.Context, userID, bookID int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM accounting_books WHERE id = $1 AND user_id = $2`, bookID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %d", tool.ErrAccountingBookNotFound, bookID)
	}
	return nil
}

func (r *AccountingRepository) ListBookCategoryStatistics(
	ctx context.Context,
	userID, bookID int64,
) ([]toolmodel.AccountingCategoryStatistic, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.category, SUM(e.amount_cents)::BIGINT, COUNT(*)::BIGINT
		FROM accounting_expenses AS e
		JOIN accounting_books AS b ON b.id = e.book_id
		WHERE b.user_id = $1 AND b.id = $2
		GROUP BY e.category
		ORDER BY SUM(e.amount_cents) DESC, e.category
	`, userID, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]toolmodel.AccountingCategoryStatistic, 0)
	for rows.Next() {
		var item toolmodel.AccountingCategoryStatistic
		var category int16
		if err := rows.Scan(&category, &item.AmountCents, &item.ExpenseCount); err != nil {
			return nil, err
		}
		item.Category = toolmodel.AccountingExpenseCategory(category)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *AccountingRepository) CreateExpense(
	ctx context.Context,
	userID int64,
	expense *toolmodel.AccountingExpense,
) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO accounting_expenses (book_id, amount_cents, category, spent_at, note)
		SELECT b.id, $1, $2, $3, $4
		FROM accounting_books AS b
		WHERE b.id = $5 AND b.user_id = $6
		RETURNING id, created_at, updated_at
	`, expense.AmountCents, expense.Category, expense.SpentAt, expense.Note, expense.BookID, userID).Scan(
		&expense.ID, &expense.CreatedAt, &expense.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %d", tool.ErrAccountingBookNotFound, expense.BookID)
	}
	return err
}

func (r *AccountingRepository) ListExpenses(
	ctx context.Context,
	userID, bookID int64,
	limit, offset int,
) ([]toolmodel.AccountingExpense, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.book_id, e.amount_cents, e.category, e.spent_at, e.note,
		       e.created_at, e.updated_at
		FROM accounting_expenses AS e
		JOIN accounting_books AS b ON b.id = e.book_id
		WHERE b.user_id = $1 AND b.id = $2
		ORDER BY e.spent_at DESC, e.id DESC
		LIMIT $3 OFFSET $4
	`, userID, bookID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := make([]toolmodel.AccountingExpense, 0)
	for rows.Next() {
		expense, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}
	return expenses, rows.Err()
}

func (r *AccountingRepository) UpdateExpense(
	ctx context.Context,
	userID int64,
	expense *toolmodel.AccountingExpense,
) error {
	var category int16
	err := r.db.QueryRow(ctx, `
		UPDATE accounting_expenses AS e
		SET amount_cents = $1, category = $2, spent_at = $3, note = $4, updated_at = NOW()
		FROM accounting_books AS b
		WHERE e.book_id = b.id AND b.user_id = $5 AND e.id = $6
		RETURNING e.book_id, e.category, e.created_at, e.updated_at
	`, expense.AmountCents, expense.Category, expense.SpentAt, expense.Note, userID, expense.ID).Scan(
		&expense.BookID, &category, &expense.CreatedAt, &expense.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %d", tool.ErrAccountingExpenseNotFound, expense.ID)
	}
	if err != nil {
		return err
	}
	expense.Category = toolmodel.AccountingExpenseCategory(category)
	return nil
}

func (r *AccountingRepository) DeleteExpense(ctx context.Context, userID, expenseID int64) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM accounting_expenses AS e
		USING accounting_books AS b
		WHERE e.book_id = b.id AND b.user_id = $1 AND e.id = $2
	`, userID, expenseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %d", tool.ErrAccountingExpenseNotFound, expenseID)
	}
	return nil
}

func (r *AccountingRepository) GetMonthlyStatistics(
	ctx context.Context,
	userID int64,
	start, end time.Time,
) (toolmodel.AccountingMonthlyStatistics, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(e.category, 0)::SMALLINT,
		       COALESCE(b.id, 0)::BIGINT,
		       COALESCE(b.name, ''),
		       SUM(e.amount_cents)::BIGINT,
		       COUNT(*)::BIGINT,
		       GROUPING(e.category)
		FROM accounting_expenses AS e
		JOIN accounting_books AS b ON b.id = e.book_id
		WHERE b.user_id = $1 AND e.spent_at >= $2 AND e.spent_at < $3
		GROUP BY GROUPING SETS ((e.category), (b.id, b.name))
		ORDER BY SUM(e.amount_cents) DESC
	`, userID, start, end)
	if err != nil {
		return toolmodel.AccountingMonthlyStatistics{}, err
	}
	defer rows.Close()

	result := toolmodel.AccountingMonthlyStatistics{
		ByCategory: make([]toolmodel.AccountingCategoryStatistic, 0),
		ByBook:     make([]toolmodel.AccountingBookStatistic, 0),
	}
	for rows.Next() {
		var category int16
		var bookID, amountCents, expenseCount int64
		var bookName string
		var categoryGrouped int
		if err := rows.Scan(
			&category, &bookID, &bookName, &amountCents, &expenseCount, &categoryGrouped,
		); err != nil {
			return toolmodel.AccountingMonthlyStatistics{}, err
		}
		if categoryGrouped == 0 {
			result.ByCategory = append(result.ByCategory, toolmodel.AccountingCategoryStatistic{
				Category:     toolmodel.AccountingExpenseCategory(category),
				AmountCents:  amountCents,
				ExpenseCount: expenseCount,
			})
			continue
		}
		result.ByBook = append(result.ByBook, toolmodel.AccountingBookStatistic{
			BookID:       bookID,
			BookName:     bookName,
			AmountCents:  amountCents,
			ExpenseCount: expenseCount,
		})
	}
	return result, rows.Err()
}

func scanExpense(row rowScanner) (toolmodel.AccountingExpense, error) {
	var expense toolmodel.AccountingExpense
	var category int16
	err := row.Scan(
		&expense.ID, &expense.BookID, &expense.AmountCents, &category, &expense.SpentAt,
		&expense.Note, &expense.CreatedAt, &expense.UpdatedAt,
	)
	expense.Category = toolmodel.AccountingExpenseCategory(category)
	return expense, err
}
