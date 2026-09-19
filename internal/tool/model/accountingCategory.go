package model

// AccountingExpenseCategory 是持久化到数据库和 JSON API 的固定支出分类编号。
type AccountingExpenseCategory int16

const (
	AccountingExpenseCategoryFood          AccountingExpenseCategory = 1
	AccountingExpenseCategoryTransport     AccountingExpenseCategory = 2
	AccountingExpenseCategoryAccommodation AccountingExpenseCategory = 3
	AccountingExpenseCategoryShopping      AccountingExpenseCategory = 4
	AccountingExpenseCategoryTicket        AccountingExpenseCategory = 5
	AccountingExpenseCategoryEntertainment AccountingExpenseCategory = 6
	AccountingExpenseCategoryMedical       AccountingExpenseCategory = 7
	AccountingExpenseCategoryOther         AccountingExpenseCategory = 8
)

func (c AccountingExpenseCategory) Valid() bool {
	switch c {
	case AccountingExpenseCategoryFood,
		AccountingExpenseCategoryTransport,
		AccountingExpenseCategoryAccommodation,
		AccountingExpenseCategoryShopping,
		AccountingExpenseCategoryTicket,
		AccountingExpenseCategoryEntertainment,
		AccountingExpenseCategoryMedical,
		AccountingExpenseCategoryOther:
		return true
	default:
		return false
	}
}
