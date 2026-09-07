import { request } from '../../../shared/api/http'
import type { LoginRequest, RegisterRequest, UserResult } from '../types'

export function login(credentials: LoginRequest) {
  return request<UserResult>('/api/v1/auth/login', {
    method: 'POST',
    json: credentials,
  })
}

export function register(credentials: RegisterRequest) {
  return request<UserResult>('/api/v1/auth/register', {
    method: 'POST',
    json: credentials,
  })
}

export function getCurrentUser() {
  return request<UserResult>('/api/v1/auth/me')
}

export function logout() {
  return request<void>('/api/v1/auth/logout', { method: 'POST' })
}
