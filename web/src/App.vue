<template>
  <!-- site 包裹【头部 + 全部内容区域】 -->
  <div class="site">
    <header class="head">
      <div class="title-text">
        <a class="mark" @click="goHome">
          <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true"
            fill="none" stroke="#2A2620" stroke-width="1.4"
            stroke-linejoin="round" stroke-linecap="round">
            <path d="M10 3 L13 8 H7 Z" />
            <path d="M6.5 9.5 h7 v2.4 h-7 z" />
            <path d="M8 12.5 h4 v2.2 h-4 z" />
            <ellipse cx="10" cy="16.4" rx="5.2" ry="1.3" stroke-width="1.1" />
          </svg>
          Cairn
        </a>
      </div>
      <nav class="mast-nav">
        <a :class="{ active: $route.path === '/' }" @click="goHome">手账</a>
        <a href="#">工具</a>
        <span class="muted">作品</span>
        <a :class="{ active: $route.path === '/login' }" @click="goLogin">登录</a>
        <a v-if="!isLoggedIn" @click="goRegister">注册</a>
        <a v-else @click="logout">登出</a>
      </nav>
    </header>

    <!-- 路由出口：列表页 / 文章页 / 登录页 -->
    <router-view />
  </div>

  <!-- 底部放在 site 外面，不受 max-width 限制 -->
  <div class="footer-wrap">
    <span class="introduce">Frontend : violet</span>
    <span class="introduce">Backend : sxt</span>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const isLoggedIn = ref(!!localStorage.getItem('token'))

function goHome() {
  router.push('/')
}

function goLogin() {
  router.push('/login')
}

function goRegister() {
  router.push({ path: '/login', query: { mode: 'register' } })
}

function logout() {
  localStorage.removeItem('token')
  isLoggedIn.value = false
  router.push('/')
}
</script>
