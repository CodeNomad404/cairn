import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchPosts, fetchPost, deletePost } from './api/posts'
import { fetchComments, createComment, deleteComment } from './api/comments'
import type { PostSummary, PostDetail, PostList, Comment, CommentWithReplies, CreateCommentInput } from './types'

function buildTree(flat: Comment[]): CommentWithReplies[] {
  const byID = new Map<number, CommentWithReplies>()
  const roots: CommentWithReplies[] = []

  for (const c of flat) {
    byID.set(c.id, { ...c, replies: [] })
  }
  for (const c of flat) {
    const node = byID.get(c.id)!
    if (c.parent_id && byID.has(c.parent_id)) {
      byID.get(c.parent_id)!.replies.push(node)
    } else {
      roots.push(node)
    }
  }
  return roots
}

export const useBlogStore = defineStore('blog', () => {
  const posts = ref<PostSummary[]>([])
  const currentPost = ref<PostDetail | null>(null)
  const comments = ref<Comment[]>([])
  const page = ref(1)
  const hasMore = ref(false)
  const loading = ref(false)

  const commentTree = computed(() => buildTree(comments.value))

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

  async function removePost(slug: string) {
    await deletePost(slug)
    currentPost.value = null
  }

  return { posts, currentPost, comments, commentTree, page, hasMore, loading, loadPosts, loadPost, loadComments, addComment, removeComment, removePost }
})
