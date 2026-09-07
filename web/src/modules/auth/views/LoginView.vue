<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import cairnLogoUrl from '../../../assets/cairn-logo.png'
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
    const redirect = getSafeRedirect(route.query.redirect)
    await router.replace(redirect)
  } catch (error) {
    errorMessage.value = error instanceof APIRequestError
      ? error.message
      : '网络异常，请稍后重试。'
  }
}

function getSafeRedirect(raw: unknown): string {
  if (typeof raw === 'string' && raw.startsWith('/') && !raw.startsWith('//') && !raw.startsWith('\\')) {
    return raw
  }
  return '/'
}
</script>

<template>
  <div class="site">
    <header class="mast">
      <RouterLink class="mark" to="/">
        <img class="mark-logo" :src="cairnLogoUrl" alt="" aria-hidden="true" />
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
