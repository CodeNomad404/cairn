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
  courses: CourseResult[]
}

export type CourseGrabRequest = {
  courses: string[]
}

export type CourseGrabItemResult = {
  target: string
  task_id?: number
  status: 'not_found' | 'ambiguous' | 'already_selected' | 'available' | 'unconfirmed' | CourseGrabTaskStatus
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
