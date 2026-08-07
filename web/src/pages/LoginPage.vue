<template>
  <div class="page">
    <div class="login-wrap">
      <div class="login-card">
        <h2>{{ isRegister ? '入山登记' : '登山报到' }}</h2>
        <span class="sub">{{ isRegister ? '领取你的第一根登山杖' : '继续上次的攀登' }}</span>

        <!-- 错误信息 -->
        <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>

        <form @submit.prevent="handleSubmit">
          <!-- 注册时多一个用户名 -->
          <div v-if="isRegister" class="form-group">
            <label>登山代号</label>
            <input
              v-model="form.username"
              type="text"
              placeholder="你的笔名"
              autocomplete="username"
              required
            />
          </div>

          <div class="form-group">
            <label>邮箱</label>
            <input
              v-model="form.email"
              type="email"
              placeholder="you@example.com"
              autocomplete="email"
              required
            />
          </div>

          <div class="form-group">
            <label>密码</label>
            <input
              v-model="form.password"
              type="password"
              placeholder="········"
              autocomplete="current-password"
              required
              minlength="6"
            />
          </div>

          <button type="submit" class="btn btn-primary" :disabled="submitting">
            {{ submitting ? '…' : isRegister ? '开始攀登' : '进入营地' }}
          </button>
        </form>

        <div class="switch">
          <span v-if="!isRegister">
            还没有登山杖？<a @click="toggleMode">注册一个</a>
          </span>
          <span v-else>
            已有登山杖？<a @click="toggleMode">直接登录</a>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { request } from '../api/request'

const router = useRouter()
const route = useRoute()

const isRegister = ref(route.query.mode === 'register')
const submitting = ref(false)
const errorMsg = ref('')

const form = reactive({
  username: '',
  email: '',
  password: ''
})

function toggleMode() {
  isRegister.value = !isRegister.value
  errorMsg.value = ''
  form.username = ''
  form.email = ''
  form.password = ''
}

async function handleSubmit() {
  errorMsg.value = ''
  submitting.value = true

  try {
    const endpoint = isRegister.value ? '/api/register' : '/api/login'
    const body = isRegister.value
      ? { username: form.username, email: form.email, password: form.password }
      : { email: form.email, password: form.password }

    const res = await request(endpoint, {
      method: 'POST',
      body: JSON.stringify(body)
    })

    if (res.token || res.data?.token) {
      localStorage.setItem('token', res.token || res.data.token)
    }

    // 登录/注册成功，跳回列表
    router.push('/')
  } catch (e) {
    errorMsg.value = '山间天气恶劣，请检查你的装备（账号或密码）…'
  } finally {
    submitting.value = false
  }
}
</script>
