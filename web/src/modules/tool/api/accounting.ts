import { request } from '../../../shared/api/http'
import type {
  AccountingBook,
  AccountingBookDetail,
  AccountingBookInput,
  AccountingBookSummary,
  AccountingExpense,
  AccountingExpenseInput,
  AccountingExpenseListResult,
  AccountingMonthlyStatistic,
} from '../types'

export function listAccountingBooks() {
  return request<{ items: AccountingBookSummary[] }>('/api/v1/tool/accounting-book/list')
}

export function getAccountingBook(id: number) {
  return request<AccountingBookDetail>('/api/v1/tool/accounting-book/get', { params: { id } })
}

export function createAccountingBook(input: AccountingBookInput) {
  return request<AccountingBook>('/api/v1/tool/accounting-book/create', {
    method: 'POST',
    json: input,
  })
}

export function updateAccountingBook(id: number, input: AccountingBookInput) {
  return request<AccountingBook>('/api/v1/tool/accounting-book/update', {
    method: 'POST',
    json: { id, ...input },
  })
}

export function deleteAccountingBook(id: number) {
  return request<void>('/api/v1/tool/accounting-book/delete', {
    method: 'POST',
    json: { id },
  })
}

export function listAccountingExpenses(bookID: number, page = 1, pageSize = 20) {
  return request<AccountingExpenseListResult>('/api/v1/tool/accounting-expense/list', {
    params: { book_id: bookID, page, page_size: pageSize },
  })
}

export function createAccountingExpense(bookID: number, input: AccountingExpenseInput) {
  return request<AccountingExpense>('/api/v1/tool/accounting-expense/create', {
    method: 'POST',
    json: { book_id: bookID, ...input },
  })
}

export function updateAccountingExpense(id: number, input: AccountingExpenseInput) {
  return request<AccountingExpense>('/api/v1/tool/accounting-expense/update', {
    method: 'POST',
    json: { id, ...input },
  })
}

export function deleteAccountingExpense(id: number) {
  return request<void>('/api/v1/tool/accounting-expense/delete', {
    method: 'POST',
    json: { id },
  })
}

export function getAccountingMonthlyStatistic(month: string) {
  return request<AccountingMonthlyStatistic>('/api/v1/tool/accounting-statistic/get', {
    params: { month },
  })
}
