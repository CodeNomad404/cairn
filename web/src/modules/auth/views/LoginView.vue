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
const isRegister = ref(false)

function toggleMode() {
  isRegister.value = !isRegister.value
  errorMessage.value = ''
}

async function submit() {
  errorMessage.value = ''
  try {
    const credentials = { username: username.value, password: password.value }
    if (isRegister.value) {
      await auth.register(credentials)
    } else {
      await auth.login(credentials)
    }
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
  <div class="site">
    <header class="mast">
      <RouterLink class="mark" to="/">
        <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" fill="none" stroke="#2A2620" stroke-width="1.4" stroke-linejoin="round" stroke-linecap="round">
          <path d="M10 3 L13 8 H7 Z"/><path d="M6.5 9.5 h7 v2.4 h-7 z"/><path d="M8 12.5 h4 v2.2 h-4 z"/><ellipse cx="10" cy="16.4" rx="5.2" ry="1.3" stroke-width="1.1"/>
        </svg>
        Cairn
      </RouterLink>
    </header>

    <main class="auth-page">
      <section class="login-card" aria-labelledby="login-title">
        <p class="eyebrow">{{ isRegister ? '加入登山' : '欢迎回来' }}</p>
        <h1 id="login-title">{{ isRegister ? '注册' : '登录' }}</h1>
        <p class="description">{{ isRegister ? '注册后即可发表评论和管理文章。' : '登录后即可发表评论和管理文章。' }}</p>

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
            {{ auth.loading ? (isRegister ? '注册中…' : '登录中…') : (isRegister ? '注册' : '登录') }}
          </button>
        </form>

        <p class="switch">
          {{ isRegister ? '已有账号？' : '没有账号？' }}
          <a role="button" @click="toggleMode">{{ isRegister ? '去登录' : '去注册' }}</a>
        </p>
      </section>
    </main>
  </div>
</template>
