import { request } from '../../../shared/api/http'
import type {
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
