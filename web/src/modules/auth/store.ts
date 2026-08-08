import { ref } from 'vue'
import { defineStore } from 'pinia'

import { APIRequestError } from '../../shared/api/http'
import { getCurrentUser, login as requestLogin, register as requestRegister, logout as requestLogout } from './api/auth'
import type { Credentials, User } from './types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)

  async function login(credentials: Credentials) {
    loading.value = true
    try {
      user.value = await requestLogin(credentials)
    } finally {
      loading.value = false
    }
  }

  async function register(credentials: Credentials) {
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
    } catch (error) {
      if (!(error instanceof APIRequestError) || error.status !== 401) {
        throw error
      }
      user.value = null
    }
  }

  async function logout() {
    await requestLogout()
    user.value = null
  }

  return { user, loading, login, register, loadCurrentUser, logout }
})
