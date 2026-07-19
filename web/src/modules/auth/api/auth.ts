import { request } from '../../../shared/api/http'
import type { Credentials, User } from '../types'

export function login(credentials: Credentials) {
  return request<User>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify(credentials),
  })
}

export function getCurrentUser() {
  return request<User>('/api/v1/auth/me')
}

export function logout() {
  return request<void>('/api/v1/auth/logout', { method: 'POST' })
}
