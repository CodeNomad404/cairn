<template>
  <div class="page">
    <section class="intro">
      <span class="eyebrow">field notebook · 卷一</span>
      <h1>盖一座博客的<br>登山手账</h1>
      <p class="lede">从空目录到 HTTPS 上线，一路把每块石头堆成路标。翻开这本，跟着走一遍。</p>
    </section>

    <main class="log">
      <!-- 加载中 -->
      <div v-if="loading" class="empty-state">
        <span class="icon">🗻</span>
        <p>正在翻山越岭…</p>
      </div>

      <!-- 加载出错 -->
      <div v-else-if="error" class="empty-state">
        <span class="icon">🌧️</span>
        <p>{{ error }}</p>
      </div>

      <!-- 文章列表 -->
      <template v-else>
        <div
          v-for="(post, index) in postList"
          :key="post.id"
          class="entry"
          :class="{ summit: index === 0 }"
          @click="goToPost(post.id)"
        >
          <!-- SVG marker -->
          <svg
            v-if="index === 0"
            class="marker"
            viewBox="0 0 48 48" fill="none" stroke="currentColor"
            stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
          >
            <path d="M24 6 v36" />
            <path d="M24 12 h14 l4 4 -4 4 h-14 z" />
            <path d="M24 22 h-14 l-4 4 4 4 h14 z" />
            <path d="M17 42 h14" />
          </svg>
          <svg
            v-else
            class="marker"
            viewBox="0 0 40 40" fill="none" stroke="currentColor"
            stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
          >
            <path d="M14 12 q6 -5 12 0 q-6 4 -12 0z" />
            <path d="M11 20 q9 -6 18 0 q-9 5 -18 0z" />
            <path d="M8 29 q12 -7 24 0 q-12 6 -24 0z" />
          </svg>

          <span v-if="index === 0" class="flag">最新一站 ✎</span>

          <span class="stamp">
            <span class="n">{{ post.coords || 'N 40.09° · E 116.35°' }}</span>
            · {{ post.weather || '云开' }}，↑{{ post.elevation || '2140' }}m
          </span>
          <h2>{{ post.title }}</h2>
          <p class="dek">{{ post.summary || post.content?.slice(0, 120) + '…' }}</p>
          <span class="date">{{ formatDate(post) }}</span>
        </div>

        <!-- 空列表 -->
        <div v-if="postList.length === 0" class="empty-state">
          <span class="icon">📭</span>
          <p>还没有任何文章，快去写第一篇吧！</p>
        </div>
      </template>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getPostList } from '../api/post'

const router = useRouter()
const postList = ref([])
const loading = ref(true)
const error = ref('')

function formatDate(post) {
  if (post.created_at) {
    return 'WP.' + new Date(post.created_at).getDate().toString().padStart(2, '0') +
      ' · ' + new Date(post.created_at).toISOString().slice(0, 10)
  }
  if (post.date) return post.date
  return 'WP.11 · 2026-07-18'
}

function goToPost(id) {
  router.push(`/post/${id}`)
}

onMounted(async () => {
  try {
    const res = await getPostList()
    postList.value = Array.isArray(res) ? res : (res.data || res.list || [])
  } catch (e) {
    error.value = '山间信号不好，请稍后再试…'
  } finally {
    loading.value = false
  }
})
</script>
