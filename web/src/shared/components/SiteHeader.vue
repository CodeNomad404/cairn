<script setup lang="ts">
import { useAuthStore } from '../../modules/auth/store'

const auth = useAuthStore()
</script>

<template>
  <header class="mast">
    <RouterLink class="mark" to="/">
      <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" fill="none" stroke="#2A2620" stroke-width="1.4" stroke-linejoin="round" stroke-linecap="round">
        <path d="M10 3 L13 8 H7 Z"/><path d="M6.5 9.5 h7 v2.4 h-7 z"/><path d="M8 12.5 h4 v2.2 h-4 z"/><ellipse cx="10" cy="16.4" rx="5.2" ry="1.3" stroke-width="1.1"/>
      </svg>
      Cairn
    </RouterLink>
    <nav class="nav">
      <RouterLink to="/">手账</RouterLink>
      <RouterLink v-if="auth.user?.role === 'admin'" to="/posts/new">发文章</RouterLink>
      <RouterLink to="/tools">工具</RouterLink>
      <span class="muted">作品</span>
      <RouterLink v-if="auth.user" to="/login" @click.prevent="auth.logout()">登出</RouterLink>
      <RouterLink v-else to="/login">登录</RouterLink>
    </nav>
  </header>
</template>
