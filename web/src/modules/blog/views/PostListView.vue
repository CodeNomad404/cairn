<script setup lang="ts">
import { onMounted } from 'vue'
import { markers } from '../markers'
import { usePostList } from '../composables/usePostList'

const { posts, page, hasMore, loading, error, loadPosts } = usePostList()

onMounted(() => { loadPosts(1) })

function formatDate(d: string | null) {
  if (!d) return ''
  const date = new Date(d)
  return `WP.${String(date.getMonth() + 1).padStart(2, '0')} · ${date.toISOString().slice(0, 10)}`
}

/** Pick a pseudo-random coordinate for the entry stamp based on index */
function coord(index: number) {
  const coords = [
    { lat: '40.09°', lon: '116.35°', alt: 2140, wx: '云开' },
    { lat: '39.91°', lon: '116.40°', alt: 1980, wx: '雾' },
    { lat: '41.20°', lon: '115.88°', alt: 2560, wx: '晴' },
    { lat: '40.55°', lon: '117.02°', alt: 2210, wx: '风' },
    { lat: '38.74°', lon: '118.11°', alt: 1760, wx: '阴' },
    { lat: '42.03°', lon: '114.76°', alt: 2890, wx: '晴' },
    { lat: '37.55°', lon: '119.30°', alt: 1520, wx: '雨' },
    { lat: '40.80°', lon: '116.99°', alt: 2330, wx: '霜' },
    { lat: '39.30°', lon: '113.45°', alt: 2010, wx: '晴' },
    { lat: '40.00°', lon: '116.00°', alt: 1200, wx: '起点' },
  ]
  return coords[index % coords.length]
}
</script>

<template>
  <div class="page">
      <section class="intro">
        <span class="eyebrow">field notebook · 卷一</span>
        <h1>盖一座博客的<br>登山手账</h1>
      </section>

      <main class="log">
        <div v-if="loading && posts.length === 0" class="empty" style="padding-left: clamp(4.4rem, 13vw, 10.5rem)">
          <span class="spinner"></span> 翻页中…
        </div>

        <div v-else-if="error && posts.length === 0" class="empty" style="padding-left: clamp(4.4rem, 13vw, 10.5rem); color: var(--blaze)">
          {{ error }}
        </div>

        <template v-else>
          <p v-if="error" style="padding-left: clamp(4.4rem, 13vw, 10.5rem); color: var(--blaze); font-size: .85rem;">{{ error }}</p>
          <RouterLink
            v-for="(post, i) in posts"
            :key="post.id"
            :to="`/posts/${post.slug}`"
            class="entry"
            :class="{ summit: i === 0 }"
          >
            <svg class="marker" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" v-html="markers[i % markers.length]"></svg>

            <span v-if="i === 0" class="flag">最新一站 ✎</span>

            <span class="stamp">
              <span class="n">N {{ coord(i).lat }} · E {{ coord(i).lon }}</span>
              · {{ coord(i).wx }}，↑{{ coord(i).alt }}m
            </span>

            <h2>{{ post.title }}</h2>
            <p class="dek">{{ post.title }}</p>
            <span class="date">{{ formatDate(post.published_at) }}</span>
          </RouterLink>
        </template>

        <div v-if="hasMore" style="padding: 2rem 0; padding-left: clamp(4.4rem, 13vw, 10.5rem)">
          <button
            class="load-more"
            style="border: 0; background: none; color: var(--pine); font-family: var(--mono); font-size: .7rem; letter-spacing: .1em; text-transform: uppercase; cursor: pointer;"
            :disabled="loading"
            @click="loadPosts(page + 1)"
          >
            {{ loading ? '加载中…' : '+ 加载更多' }}
          </button>
        </div>
      </main>
    </div>
</template>
