<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { APIRequestError } from '../../../shared/api/http'
import { useAuthStore } from '../store'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const username = ref('')
const password = ref('')
const errorMessage = ref('')

async function submit() {
  errorMessage.value = ''
  try {
    await auth.login({ username: username.value, password: password.value })
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (error) {
    errorMessage.value = error instanceof APIRequestError
      ? error.message
      : '网络异常，请稍后重试。'
  }
}
</script>

<template>
  <main class="auth-page">
    <section class="login-card" aria-labelledby="login-title">
      <RouterLink class="brand" to="/">Cairn</RouterLink>
      <p class="eyebrow">欢迎回来</p>
      <h1 id="login-title">登录</h1>
      <p class="description">登录后即可发表评论和管理文章。</p>

      <form class="login-form" @submit.prevent="submit">
        <label>
          用户名
          <input v-model.trim="username" name="username" autocomplete="username" required minlength="2" maxlength="64" />
        </label>
        <label>
          密码
          <input v-model="password" type="password" name="password" autocomplete="current-password" required minlength="6" />
        </label>
        <p v-if="errorMessage" class="form-error" role="alert">{{ errorMessage }}</p>
        <button type="submit" :disabled="auth.loading">
          {{ auth.loading ? '登录中…' : '登录' }}
        </button>
      </form>

    </section>
  </main>
</template>

<style scoped>
.auth-page { min-height: 100svh; display: grid; place-items: center; padding: 24px; }
.login-card { width: min(100%, 400px); padding: 40px; border: 1px solid var(--border); border-radius: 16px; background: var(--surface); box-shadow: var(--shadow); }
.brand { color: var(--accent); font-weight: 700; text-decoration: none; }
.eyebrow { margin: 32px 0 8px; color: var(--muted); font-size: 14px; }
h1 { margin: 0; font-size: 32px; letter-spacing: -0.04em; }
.description { margin: 12px 0 28px; color: var(--muted); }
.login-form { display: grid; gap: 18px; }
label { display: grid; gap: 8px; color: var(--text); font-size: 14px; font-weight: 600; }
input { width: 100%; border: 1px solid var(--border); border-radius: 8px; padding: 11px 12px; color: var(--text); background: var(--surface); }
input:focus { outline: 2px solid var(--accent-focus); border-color: var(--accent); }
button { border: 0; border-radius: 8px; padding: 12px; color: #fff; background: var(--accent); font-weight: 600; cursor: pointer; }
button:disabled { cursor: wait; opacity: .65; }
.form-error { margin: -4px 0 0; color: var(--danger); font-size: 14px; }
@media (max-width: 480px) { .login-card { padding: 28px 24px; } }
</style>
