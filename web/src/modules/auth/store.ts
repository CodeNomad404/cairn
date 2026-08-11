import { ref } from 'vue'
import { defineStore } from 'pinia'

import { APIRequestError } from '../../shared/api/http'
import { getCurrentUser, login as requestLogin, register as requestRegister, logout as requestLogout } from './api/auth'
import type { LoginRequest, RegisterRequest, UserResult } from './types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<UserResult | null>(null)
  const loading = ref(false)
  const initialized = ref(false)

  let initPromise: Promise<void> | null = null

  async function login(credentials: LoginRequest) {
    loading.value = true
    try {
      user.value = await requestLogin(credentials)
    } finally {
      loading.value = false
    }
  }

  async function register(credentials: RegisterRequest) {
    loading.value = true
    try {
      user.value = await requestRegister(credentials)
    } finally {
      loading.value = false
    }
  }

  async function loadCurrentUser() {
    try {
      user.value = await getCurrentUser()
      initialized.value = true
    } catch (error) {
      if (error instanceof APIRequestError && error.status === 401) {
        user.value = null
        initialized.value = true
        return
      }
      // Network error or 500: leave initialized false to allow retry
      throw error
    }
  }

  /** Deduplicates in-flight requests. Throws on network/500 so callers can distinguish "unavailable" from "guest". */
  async function ensureCurrentUser() {
    if (initialized.value) return
    if (initPromise) return initPromise
    initPromise = loadCurrentUser().finally(() => {
      initPromise = null
    })
    return initPromise
  }

  async function logout() {
    await requestLogout()
    user.value = null
  }

  return { user, loading, initialized, login, register, loadCurrentUser, ensureCurrentUser, logout }
})
