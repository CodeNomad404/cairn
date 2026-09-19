export type FormatJSONRequest = {
  input: string
}

export type FormatJSONResult = {
  formatted: string
}

export type CourseRegistrationLoginRequest = {
  student_id?: string
  password?: string
  tujian_username?: string
  tujian_password?: string
}

export type CourseRegistrationLoginResult = {
  logged_in: boolean
}

export type CourseResult = {
  id?: string
  name: string
  teacher: string
  remaining: number
  selected: boolean
}

export type CourseListResult = {
  selection_open: boolean
  courses: CourseResult[]
}

export type CourseGrabRequest = {
  courses: string[]
}

export type CourseGrabItemResult = {
  target: string
  task_id?: number
  status: 'already_selected' | 'ambiguous' | 'unconfirmed' | CourseGrabTaskStatus
  course?: CourseResult
  selected: boolean
  message: string
}

export type CourseGrabResult = {
  results: CourseGrabItemResult[]
}

export type CourseGrabTaskStatus = 'grabbing' | 'paused' | 'succeeded' | 'not_found'

export type CourseGrabTask = {
  id: number
  course_name: string
  upstream_course_id?: string
  teacher: string
  remaining: number
  status: CourseGrabTaskStatus
  message: string
  attempt_count: number
  last_checked_at?: string
  succeeded_at?: string
  created_at: string
  updated_at: string
}

export type CourseGrabTaskListResult = {
  worker_status: 'idle' | 'running' | 'auth_failed'
  login_count: number
  last_login_at?: string
  items: CourseGrabTask[]
}

export type CourseGrabTaskStatusRequest = {
  status: 'grabbing' | 'paused'
}

export type ClassroomOccupationType =
  | 'free'
  | 'scheduled'
  | 'rescheduled'
  | 'exam'
  | 'laboratory'
  | 'other'

export type ClassroomListParams = {
  period?: number
  building_id?: string
  min_capacity?: number
}

export type ClassroomRoom = {
  period: number
  building_id: string
  building_name: string
  room_name: string
  capacity: number
  occupation_type: ClassroomOccupationType
}

export type ClassroomListResult = {
  date: string
  period?: number
  synced_at?: string
  rooms: ClassroomRoom[]
}

export type AccountingExpenseCategory = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8

export type AccountingBook = {
  id: number
  name: string
  description: string
  created_at: string
  updated_at: string
}

export type AccountingBookSummary = AccountingBook & {
  total_amount_cents: number
  expense_count: number
  last_expense_at?: string
}

export type AccountingCategoryStatistic = {
  category: AccountingExpenseCategory
  amount_cents: number
  expense_count: number
}

export type AccountingBookDetail = AccountingBookSummary & {
  categories: AccountingCategoryStatistic[]
}

export type AccountingBookInput = {
  name: string
  description: string
}

export type AccountingExpense = {
  id: number
  book_id: number
  amount_cents: number
  category: AccountingExpenseCategory
  spent_at: string
  note: string
  created_at: string
  updated_at: string
}

export type AccountingExpenseInput = {
  amount_cents: number
  category: AccountingExpenseCategory
  spent_at: string
  note: string
}

export type AccountingExpenseListResult = {
  items: AccountingExpense[]
  page: number
  page_size: number
  has_more: boolean
}

export type AccountingBookStatistic = {
  book_id: number
  book_name: string
  amount_cents: number
  expense_count: number
}

export type AccountingMonthlyStatistic = {
  month: string
  total_amount_cents: number
  by_category: AccountingCategoryStatistic[]
  by_book: AccountingBookStatistic[]
}
