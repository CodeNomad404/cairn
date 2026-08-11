<script setup lang="ts">
import type { PostDetailResult } from '../types'

defineProps<{
  post: PostDetailResult
  isAdmin: boolean
  deleting: boolean
}>()

defineEmits<{
  delete: []
}>()

function formatDate(d: string | null) {
  if (!d) return ''
  return new Date(d).toISOString().slice(0, 10)
}

function pickMarker(slug: string) {
  const markers = [
    '<path d="M24 6 v36"/><path d="M24 12 h14 l4 4 -4 4 h-14 z"/><path d="M24 22 h-14 l-4 4 4 4 h14 z"/><path d="M17 42 h14"/>',
    '<path d="M14 12 q6 -5 12 0 q-6 4 -12 0z"/><path d="M11 20 q9 -6 18 0 q-9 5 -18 0z"/><path d="M8 29 q12 -7 24 0 q-12 6 -24 0z"/>',
    '<path d="M6 32 L17 12 L24 24 L28 18 L34 32 Z"/><path d="M17 12 v-6 l6 2 -6 2"/>',
    '<circle cx="20" cy="20" r="13"/><path d="M20 12 L24 22 L20 20 L16 22 Z"/>',
  ]
  let hash = 0
  for (let i = 0; i < slug.length; i++) hash = ((hash << 5) - hash) + slug.charCodeAt(i) | 0
  return markers[Math.abs(hash) % markers.length]
}
</script>

<template>
  <div class="reader">
    <RouterLink class="back" to="/">← 返回手账</RouterLink>

    <div class="fieldlog">
      <svg class="marker" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" v-html="pickMarker(post.slug)"></svg>
      <div class="meta">
        {{ formatDate(post.published_at) }}
      </div>
    </div>

    <article class="article">
      <h1>{{ post.title }}</h1>
      <div v-html="post.content_html" />
    </article>

    <p v-if="isAdmin" style="margin-top: 2rem">
      <button
        style="border: 0; background: none; color: var(--blaze); font-family: var(--mono); font-size: .68rem; letter-spacing: .08em; text-transform: uppercase; cursor: pointer;"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? '删除中…' : '删除文章' }}
      </button>
    </p>
  </div>
</template>
