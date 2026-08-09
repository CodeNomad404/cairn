<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useBlogStore } from '../store'
import { useAuthStore } from '../../auth/store'
import type { Comment } from '../types'
import SiteHeader from '../../../shared/components/SiteHeader.vue'
import SiteFooter from '../../../shared/components/SiteFooter.vue'

const route = useRoute()
const blog = useBlogStore()
const auth = useAuthStore()

const slug = route.params.slug as string
const commentText = ref('')
const replyTo = ref<number | null>(null)
const commentError = ref('')
const submitting = ref(false)

onMounted(async () => {
  await blog.loadPost(slug)
  await blog.loadComments(slug)
})

function formatDate(d: string | null) {
  if (!d) return ''
  return new Date(d).toISOString().slice(0, 10)
}

function pickMarker(slugStr: string) {
  const markers = [
    '<path d="M24 6 v36"/><path d="M24 12 h14 l4 4 -4 4 h-14 z"/><path d="M24 22 h-14 l-4 4 4 4 h14 z"/><path d="M17 42 h14"/>',
    '<path d="M14 12 q6 -5 12 0 q-6 4 -12 0z"/><path d="M11 20 q9 -6 18 0 q-9 5 -18 0z"/><path d="M8 29 q12 -7 24 0 q-12 6 -24 0z"/>',
    '<path d="M6 32 L17 12 L24 24 L28 18 L34 32 Z"/><path d="M17 12 v-6 l6 2 -6 2"/>',
    '<circle cx="20" cy="20" r="13"/><path d="M20 12 L24 22 L20 20 L16 22 Z"/>',
  ]
  let hash = 0
  for (let i = 0; i < slugStr.length; i++) hash = ((hash << 5) - hash) + slugStr.charCodeAt(i) | 0
  return markers[Math.abs(hash) % markers.length]
}

function replyToComment(id: number, username: string) {
  replyTo.value = id
  commentText.value = `@${username} `
  commentError.value = ''
}

function cancelReply() {
  replyTo.value = null
  commentText.value = ''
}

async function submitComment() {
  if (!commentText.value.trim()) return
  submitting.value = true
  commentError.value = ''
  try {
    await blog.addComment(slug, {
      content: commentText.value,
      parent_id: replyTo.value,
    })
    commentText.value = ''
    replyTo.value = null
  } catch (e: any) {
    commentError.value = e?.message ?? '提交失败，请稍后重试。'
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  if (!confirm('确定删除这条评论？')) return
  try {
    await blog.removeComment(id, slug)
  } catch (e: any) {
    alert(e?.message ?? '删除失败')
  }
}

function canDelete(comment: Comment) {
  if (!auth.user) return false
  return auth.user.role === 'admin' || auth.user.id === comment.user_id
}
</script>

<template>
  <div class="site">
    <SiteHeader />
    <div class="page">
      <div v-if="blog.loading && !blog.currentPost" class="reader">
        <span class="spinner"></span> 加载中…
      </div>

      <template v-else-if="blog.currentPost">
        <div class="reader">
          <RouterLink class="back" to="/">← 返回手账</RouterLink>

          <div class="fieldlog">
            <svg class="marker" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" v-html="pickMarker(slug)"></svg>
            <div class="meta">
              {{ formatDate(blog.currentPost.published_at) }}
            </div>
          </div>

          <article class="article">
            <h1>{{ blog.currentPost.title }}</h1>
            <div v-html="blog.currentPost.content_html" />
          </article>

          <!-- Comments section -->
          <section class="comments">
            <h2>留言</h2>

            <div v-if="auth.user" class="comment-form">
              <textarea
                v-model="commentText"
                :placeholder="replyTo ? '写下回复…' : '写下留言…'"
                maxlength="4000"
              />
              <div class="row">
                <span v-if="replyTo" class="hint">
                  回复中…
                  <button style="background:none;border:0;color:var(--blaze);font-family:var(--mono);font-size:.65rem;cursor:pointer;text-transform:uppercase;letter-spacing:.08em" @click="cancelReply">取消</button>
                </span>
                <span v-else class="hint">最多 4000 字</span>
                <button :disabled="submitting || !commentText.trim()" @click="submitComment">
                  {{ submitting ? '提交中…' : '提交' }}
                </button>
              </div>
              <p v-if="commentError" class="comment-error">{{ commentError }}</p>
            </div>
            <p v-else style="color: var(--faded); font-size: .9rem">
              <RouterLink to="/login" style="color: var(--pine)">登录</RouterLink> 后即可留言。
            </p>

            <div v-if="blog.comments.length === 0" class="empty">暂无留言</div>

            <div v-for="c in blog.comments" :key="c.id" class="comment">
              <div class="by">
                <span class="name">{{ c.username }}</span>
                · {{ formatDate(c.created_at) }}
              </div>
              <div class="body">{{ c.content }}</div>
              <div class="actions">
                <button v-if="auth.user" @click="replyToComment(c.id, c.username)">回复</button>
                <button v-if="canDelete(c)" @click="handleDelete(c.id)">删除</button>
              </div>
              <div v-if="c.replies && c.replies.length" class="replies">
                <div v-for="r in c.replies" :key="r.id" class="comment">
                  <div class="by">
                    <span class="name">{{ r.username }}</span>
                    · {{ formatDate(r.created_at) }}
                  </div>
                  <div class="body">{{ r.content }}</div>
                  <div class="actions">
                    <button v-if="canDelete(r)" @click="handleDelete(r.id)">删除</button>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </template>

      <div v-else class="reader">
        <RouterLink class="back" to="/">← 返回手账</RouterLink>
        <p style="color: var(--faded)">文章未找到。</p>
      </div>
    </div>
    <SiteFooter />
  </div>
</template>
