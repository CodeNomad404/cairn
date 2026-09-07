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
