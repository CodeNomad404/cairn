import { request } from '../../../shared/api/http'
import type {
  ClassroomListParams,
  ClassroomListResult,
  CourseRegistrationLoginRequest,
  CourseRegistrationLoginResult,
  CourseGrabRequest,
  CourseGrabResult,
  CourseGrabTask,
  CourseGrabTaskListResult,
  CourseGrabTaskStatusRequest,
  CourseListResult,
  FormatJSONRequest,
  FormatJSONResult,
} from '../types'

export function listClassrooms(input: ClassroomListParams = {}) {
  const params: Record<string, string | number> = {}
  if (input.period !== undefined) params.period = input.period
  if (input.building_id !== undefined) params.building_id = input.building_id
  if (input.min_capacity !== undefined) params.min_capacity = input.min_capacity
  return request<ClassroomListResult>('/api/v1/tool/classroom/list', { params })
}

export function formatJSON(input: FormatJSONRequest) {
  return request<FormatJSONResult>('/api/v1/tools/json/format', {
    method: 'POST',
    json: input,
  })
}

export function listCourses() {
  return request<CourseListResult>('/api/v1/tools/course-registration/courses')
}

export function grabCourses(input: CourseGrabRequest) {
  return request<CourseGrabResult>('/api/v1/tools/course-registration/grab', {
    method: 'POST',
    json: input,
  })
}

export function listCourseGrabTasks() {
  return request<CourseGrabTaskListResult>('/api/v1/tools/course-registration/grab-tasks')
}

export function setCourseGrabTaskStatus(taskID: number, input: CourseGrabTaskStatusRequest) {
  return request<CourseGrabTask>(`/api/v1/tools/course-registration/grab-tasks/${taskID}`, {
    method: 'PATCH',
    json: input,
  })
}

export function loginCourseRegistration(input: CourseRegistrationLoginRequest) {
  return request<CourseRegistrationLoginResult>('/api/v1/tools/course-registration/login', {
    method: 'POST',
    json: input,
  })
}
