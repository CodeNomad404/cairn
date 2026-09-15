<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '../../auth'

const auth = useAuthStore()
// 教务抢课仅管理员可见；用户信息未加载完成前先显示，避免管理员看到卡片闪烁
const isAdmin = computed(() => auth.user?.role === 'admin' || !auth.initialized)
</script>

<template>
  <div class="page">
      <section class="tools-intro">
        <span class="eyebrow">工具站</span>
        <h1>在线小工具</h1>
        <p class="lede">一些随手写的轻量工具，不用就忘了。</p>
      </section>

      <div class="tool-list">
        <RouterLink to="/tools/json" class="tool-card">
          <h2>JSON 格式化</h2>
          <p>校验并美化 JSON 字符串，缩进 2 空格。</p>
        </RouterLink>
        <RouterLink to="/tools/diff" class="tool-card">
          <h2>文本 Diff</h2>
          <p>比较两段文本的差异，实时高亮增删改动。</p>
        </RouterLink>
        <RouterLink to="/tools/classroom" class="tool-card">
          <h2>空闲教室</h2>
          <p>按教学楼、节次和占用状态查询今天的教室。</p>
        </RouterLink>
        <RouterLink v-if="isAdmin" to="/tools/course-registration" class="tool-card">
          <h2>教务抢课</h2>
          <p>登录教务系统看余量、一键抢课，没有余量的课程自动排队。</p>
        </RouterLink>
      </div>
    </div>
</template>
