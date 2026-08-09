<script setup lang="ts">
import SiteHeader from '../../../shared/components/SiteHeader.vue'
import SiteFooter from '../../../shared/components/SiteFooter.vue'
</script>

<template>
  <div class="site">
    <SiteHeader />
    <div class="page">
      <section class="intro">
        <span class="eyebrow">迷路了</span>
        <h1>这片区域<br>还没有路标</h1>
        <p class="lede">
          <RouterLink to="/" style="color: var(--pine); text-underline-offset: 3px;">← 回到手账</RouterLink>
        </p>
      </section>
    </div>
    <SiteFooter />
  </div>
</template>
