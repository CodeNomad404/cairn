<template>
  <main class="placeholder-page">
    <h1>Cairn</h1>
    <p>博客首页正在迁移到 Vue。</p>
    <RouterLink to="/login">前往登录</RouterLink>
  </main>
</template>

<style scoped>
.placeholder-page { min-height: 100svh; display: grid; place-content: center; gap: 16px; text-align: center; }
a { color: var(--accent); }
</style>
