import { request } from '../../../shared/api/http'
import type {
  CourseRegistrationLoginRequest,
  CourseRegistrationLoginResult,
  FormatJSONRequest,
  FormatJSONResult,
} from '../types'

export function formatJSON(input: FormatJSONRequest) {
  return request<FormatJSONResult>('/api/v1/tools/json/format', {
    method: 'POST',
    json: input,
  })
}

export function loginCourseRegistration(input: CourseRegistrationLoginRequest) {
  return request<CourseRegistrationLoginResult>('/api/v1/tools/course-registration/login', {
    method: 'POST',
    json: input,
  })
}
