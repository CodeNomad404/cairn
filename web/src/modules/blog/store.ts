import { ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchPosts, fetchPost } from './api/posts'
import { fetchComments, createComment, deleteComment } from './api/comments'
import type { PostSummary, PostDetail, PostList, Comment, CreateCommentInput } from './types'

export const useBlogStore = defineStore('blog', () => {
  const posts = ref<PostSummary[]>([])
  const currentPost = ref<PostDetail | null>(null)
  const comments = ref<Comment[]>([])
  const page = ref(1)
  const hasMore = ref(false)
  const loading = ref(false)

  async function loadPosts(pageNum = 1) {
    loading.value = true
    try {
      const data: PostList = await fetchPosts(pageNum)
      if (pageNum === 1) {
        posts.value = data.items
      } else {
        posts.value.push(...data.items)
      }
      page.value = data.page
      hasMore.value = data.has_more
    } finally {
      loading.value = false
    }
  }

  async function loadPost(slug: string) {
    loading.value = true
    try {
      currentPost.value = await fetchPost(slug)
    } finally {
      loading.value = false
    }
  }

  async function loadComments(slug: string) {
    const data = await fetchComments(slug)
    comments.value = data.items
  }

  async function addComment(slug: string, input: CreateCommentInput) {
    await createComment(slug, input)
    await loadComments(slug)
  }

  async function removeComment(id: number, slug: string) {
    await deleteComment(id)
    await loadComments(slug)
  }

  return { posts, currentPost, comments, page, hasMore, loading, loadPosts, loadPost, loadComments, addComment, removeComment }
})
