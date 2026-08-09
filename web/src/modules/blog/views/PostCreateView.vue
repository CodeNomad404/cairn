<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../auth/store'
import { useBlogStore } from '../store'
import SiteHeader from '../../../shared/components/SiteHeader.vue'
import SiteFooter from '../../../shared/components/SiteFooter.vue'

const router = useRouter()
const auth = useAuthStore()
const blog = useBlogStore()

const slug = ref('')
const title = ref('')
const contentMD = ref('')
const errorMessage = ref('')
const submitting = ref(false)

if (auth.user?.role !== 'admin') {
  router.replace('/login')
}

async function submit() {
  errorMessage.value = ''
  submitting.value = true
  try {
    const post = await blog.addPost({
      slug: slug.value,
      title: title.value,
      content_md: contentMD.value,
    })
    await router.replace(`/posts/${post.slug}`)
  } catch (e: any) {
    errorMessage.value = e?.message ?? '发布失败，请稍后重试。'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="site">
    <SiteHeader />
    <div class="page">
      <div class="tool-page">
        <RouterLink class="back" to="/">← 返回手账</RouterLink>
        <h1>写文章</h1>
        <p class="desc">Markdown 格式，发布后自动渲染为 HTML。</p>

        <form @submit.prevent="submit" style="display: grid; gap: 1.2rem">
          <label style="display: grid; gap: .3rem; font-family: var(--mono); font-size: .75rem; letter-spacing: .06em; text-transform: uppercase; color: var(--faded);">
            Slug
            <input
              v-model.trim="slug"
              required
              placeholder="hello-world"
              style="width: 100%; border: 1px solid var(--binding); border-radius: 4px; padding: .6rem .8rem; font: inherit; background: var(--paper); color: var(--ink);"
            />
          </label>
          <label style="display: grid; gap: .3rem; font-family: var(--mono); font-size: .75rem; letter-spacing: .06em; text-transform: uppercase; color: var(--faded);">
            标题
            <input
              v-model.trim="title"
              required
              placeholder="文章标题"
              style="width: 100%; border: 1px solid var(--binding); border-radius: 4px; padding: .6rem .8rem; font: inherit; background: var(--paper); color: var(--ink);"
            />
          </label>
          <label style="display: grid; gap: .3rem; font-family: var(--mono); font-size: .75rem; letter-spacing: .06em; text-transform: uppercase; color: var(--faded);">
            内容 (Markdown)
            <textarea
              v-model="contentMD"
              required
              placeholder="# 标题&#10;&#10;开始写…"
              style="width: 100%; min-height: 20rem; border: 1px solid var(--binding); border-radius: 4px; padding: .8rem 1rem; font-family: var(--mono); font-size: .88rem; line-height: 1.7; resize: vertical; background: var(--paper); color: var(--ink);"
            />
          </label>

          <p v-if="errorMessage" style="color: var(--blaze); font-size: .85rem;">{{ errorMessage }}</p>

          <div style="display: flex; justify-content: flex-end;">
            <button
              type="submit"
              :disabled="submitting"
              style="border: 0; border-radius: 4px; padding: .7rem 2rem; background: var(--ink); color: var(--paper); font-family: var(--mono); font-size: .72rem; letter-spacing: .08em; text-transform: uppercase; cursor: pointer;"
            >
              {{ submitting ? '发布中…' : '发布' }}
            </button>
          </div>
        </form>
      </div>
    </div>
    <SiteFooter />
  </div>
</template>
