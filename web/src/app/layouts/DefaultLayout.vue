<script setup lang="ts">
import SiteHeader from '../components/SiteHeader.vue'
import SiteFooter from '../components/SiteFooter.vue'
</script>

<template>
  <div class="site">
    <SiteHeader />
    <RouterView />
    <SiteFooter />
  </div>
</template>
