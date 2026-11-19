<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import cairnLogoUrl from '../../../assets/cairn-logo.png'
import { APIRequestError } from '../../../shared/api/http'
import { useAuthStore } from '../store'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const username = ref('')
const password = ref('')
const confirmation = ref('')
const showPassword = ref(false)
const errorMessage = ref('')
const isRegister = computed(() => route.name === 'register')

watch(isRegister, () => {
  password.value = ''
  confirmation.value = ''
  showPassword.value = false
  errorMessage.value = ''
})

async function submit() {
  if (auth.loading) return
  errorMessage.value = ''
  const name = username.value.replace(/^\p{White_Space}+|\p{White_Space}+$/gu, '')
  if (isRegister.value) {
    const nameLength = [...name].length
    if (nameLength < 2 || nameLength > 64) {
      errorMessage.value = '用户名需要 2–64 个字符'
    } else if (/\p{Cc}/u.test(username.value)) {
      errorMessage.value = '用户名不能包含控制字符'
    } else if ([...password.value].length < 8) {
      errorMessage.value = '密码至少需要 8 个字符'
    } else if (new TextEncoder().encode(password.value).length > 72) {
      errorMessage.value = '密码不能超过 72 字节，请缩短密码'
    } else if (confirmation.value !== password.value) {
      errorMessage.value = '两次输入的密码不一致'
    }
    if (errorMessage.value) return
  }
  const redirect = getSafeRedirect(route.query.redirect)
  try {
    const credentials = { username: name, password: password.value }
    if (isRegister.value) {
      await auth.register(credentials)
    } else {
      await auth.login(credentials)
    }
    await router.replace(redirect)
  } catch (error) {
    if (error instanceof APIRequestError) {
      errorMessage.value = error.status === 429 ? '请求过于频繁，请稍后重试。' : error.message
    } else {
      errorMessage.value = '网络异常，请稍后重试。'
    }
  }
}

function getSafeRedirect(raw: unknown): string {
  if (typeof raw === 'string' && raw.startsWith('/') && !raw.startsWith('//') && !raw.includes('\\') && !/\p{Cc}/u.test(raw)) {
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
        <p class="description">{{ isRegister ? '创建账号后即可发表评论。' : '登录后即可发表评论并使用需要账号的工具。' }}</p>

        <form @submit.prevent="submit">
          <fieldset class="login-form" :disabled="auth.loading">
            <label>
              用户名
              <input v-model="username" name="username" autocomplete="username" required
                :aria-describedby="isRegister ? 'username-hint' : undefined" />
              <small v-if="isRegister" id="username-hint" class="auth-hint">2–64 个字符，支持中文。</small>
            </label>
            <label>
              密码
              <input v-model="password" :type="showPassword ? 'text' : 'password'" name="password"
                :autocomplete="isRegister ? 'new-password' : 'current-password'" required
                :aria-describedby="isRegister ? 'password-hint' : undefined" />
              <small v-if="isRegister" id="password-hint" class="auth-hint">至少 8 个字符，最多 72 字节（中文通常占 3 字节）。</small>
            </label>
            <label v-if="isRegister">
              确认密码
              <input v-model="confirmation" :type="showPassword ? 'text' : 'password'" name="confirm-password"
                autocomplete="new-password" required />
            </label>
            <label v-if="isRegister" class="password-visibility">
              <input v-model="showPassword" type="checkbox" />
              显示密码
            </label>
            <p v-if="errorMessage" class="form-error" role="alert">{{ errorMessage }}</p>
            <button type="submit">
              {{ auth.loading ? (isRegister ? '注册中…' : '登录中…') : (isRegister ? '注册' : '登录') }}
            </button>
          </fieldset>
        </form>

        <p class="switch">
          {{ isRegister ? '已有账号？' : '没有账号？' }}
          <RouterLink v-if="!auth.loading" :to="{ name: isRegister ? 'login' : 'register', query: route.query }">
            {{ isRegister ? '去登录' : '去注册' }}
          </RouterLink>
        </p>
      </section>
    </main>
  </div>
</template>
