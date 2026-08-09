<script setup lang="ts">
import SiteHeader from '../../../shared/components/SiteHeader.vue'
import SiteFooter from '../../../shared/components/SiteFooter.vue'
</script>

<template>
  <div class="site">
    <SiteHeader />
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
        <div class="tool-card" style="opacity: .5">
          <h2>文本 Diff</h2>
          <p>比较两段文本的差异 — 即将上线。</p>
        </div>
      </div>
    </div>
    <SiteFooter />
  </div>
</template>
