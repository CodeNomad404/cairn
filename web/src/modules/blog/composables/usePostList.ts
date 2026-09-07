import { ref } from 'vue'
import { fetchPosts } from '../api/posts'
import type { PostSummaryResult } from '../types'

export function usePostList() {
  const posts = ref<PostSummaryResult[]>([])
  const page = ref(1)
  const hasMore = ref(false)
  const loading = ref(false)
  const error = ref('')

  async function loadPosts(pageNum = 1) {
    loading.value = true
    error.value = ''
    try {
      const data = await fetchPosts(pageNum)
      if (pageNum === 1) {
        posts.value = data.items
      } else {
        posts.value.push(...data.items)
      }
      page.value = data.page
      hasMore.value = data.has_more
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : '加载文章失败'
    } finally {
      loading.value = false
    }
  }

  return { posts, page, hasMore, loading, error, loadPosts }
}
