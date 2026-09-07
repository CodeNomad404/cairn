<script setup lang="ts">
import cairnLogoUrl from '../../assets/cairn-logo.png'
import { useAuthStore } from '../../modules/auth/store'

const auth = useAuthStore()
</script>

<template>
  <header class="mast">
    <RouterLink class="mark" to="/">
      <img class="mark-logo" :src="cairnLogoUrl" alt="" aria-hidden="true" />
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
