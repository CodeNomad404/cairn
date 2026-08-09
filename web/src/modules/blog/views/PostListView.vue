<script setup lang="ts">
import { onMounted } from 'vue'
import { useBlogStore } from '../store'
import SiteHeader from '../../../shared/components/SiteHeader.vue'
import SiteFooter from '../../../shared/components/SiteFooter.vue'
import { markers } from '../../../shared/markers'

const blog = useBlogStore()

onMounted(() => blog.loadPosts(1))

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
  <div class="site">
    <SiteHeader />
    <div class="page">
      <section class="intro">
        <span class="eyebrow">field notebook · 卷一</span>
        <h1>盖一座博客的<br>登山手账</h1>
        <p class="lede">从空目录到 HTTPS 上线，一路把每块石头堆成路标。翻开这本，跟着走一遍。</p>
      </section>

      <main class="log">
        <div v-if="blog.loading && blog.posts.length === 0" class="empty" style="padding-left: clamp(4.4rem, 13vw, 10.5rem)">
          <span class="spinner"></span> 翻页中…
        </div>

        <template v-else>
          <RouterLink
            v-for="(post, i) in blog.posts"
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

        <div v-if="blog.hasMore" style="padding: 2rem 0; padding-left: clamp(4.4rem, 13vw, 10.5rem)">
          <button
            class="load-more"
            style="border: 0; background: none; color: var(--pine); font-family: var(--mono); font-size: .7rem; letter-spacing: .1em; text-transform: uppercase; cursor: pointer;"
            :disabled="blog.loading"
            @click="blog.loadPosts(blog.page + 1)"
          >
            {{ blog.loading ? '加载中…' : '+ 加载更多' }}
          </button>
        </div>
      </main>
    </div>
    <SiteFooter />
  </div>
</template>
