import { request } from '../../../shared/api/http'

export type HomeworkStatus = 'pending' | 'overdue' | 'submitted' | 'upcoming' | 'unknown'
export type HomeworkFilter = HomeworkStatus | 'todo' | 'all'
export interface HomeworkItem {
  id: number
  course_id: string
  course_name: string
  title: string
  opens_at: string | null
  deadline: string | null
  submitted_at: string | null
  submission: string
  status: HomeworkStatus
  grading: string
}
export interface HomeworkCourse {
  id: string
  name: string
  last_attempt_at: string
  last_success_at: string | null
  error: string
}
export interface HomeworkList {
  bound: boolean
  student_id: string
  has_snapshot: boolean
  term_code: string
  last_success_at: string | null
  courses: HomeworkCourse[]
  items: HomeworkItem[]
  counts: Record<HomeworkStatus, number>
  total: number
  page: number
  page_size: number
}
export interface HomeworkDetail {
  item: HomeworkItem
  content: string
  attachments: { name: string }[]
  warning?: string
}
export interface HomeworkSync {
  status: 'success' | 'partial' | 'failed'
  succeeded: number
  failed: number
  courses: HomeworkCourse[]
}

export function bindMIS(input: { student_id: string; password: string }) {
  return request<{ bound: boolean; student_id: string }>('/api/v1/tool/mis-account/bind', { method: 'POST', json: input })
}
export function listHomework(input: { course_id: string; status: HomeworkFilter; page: number }, signal?: AbortSignal) {
  return request<HomeworkList>('/api/v1/tool/course-homework/list', { params: { ...input, page_size: 20 }, signal })
}
export function getHomework(id: number, signal?: AbortSignal) {
  return request<HomeworkDetail>('/api/v1/tool/course-homework/get', { params: { id }, signal })
}
export function syncHomework(signal?: AbortSignal) {
  return request<HomeworkSync>('/api/v1/tool/course-homework/sync', { method: 'POST', json: {}, signal })
}
