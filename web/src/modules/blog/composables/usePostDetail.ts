import { ref, watch, onUnmounted } from 'vue'
import { fetchPost } from '../api/posts'
import { fetchComments, createComment, deleteComment } from '../api/comments'
import type { CommentResult, CreateCommentRequest, PostDetailResult } from '../types'

export function usePostDetail(slug: () => string) {
  const post = ref<PostDetailResult | null>(null)
  const postLoading = ref(false)
  const postError = ref('')

  const commentTree = ref<CommentResult[]>([])
  const commentsLoading = ref(false)
  const commentsError = ref('')

  let postController: AbortController | null = null
  let commentsController: AbortController | null = null

  async function loadPost(s: string) {
    // Don't start loading for a slug that's already stale
    if (s !== slug()) return

    postController?.abort()

    const controller = new AbortController()
    postController = controller

    postLoading.value = true
    postError.value = ''

    try {
      const result = await fetchPost(s, controller.signal)
      if (postController !== controller || s !== slug()) return
      post.value = result
    } catch (e: unknown) {
      if (postController !== controller || s !== slug()) return
      if (e instanceof DOMException && e.name === 'AbortError') return
      postError.value = e instanceof Error ? e.message : '加载文章失败'
    } finally {
      if (postController === controller) {
        postLoading.value = false
      }
    }
  }

  async function loadComments(s: string) {
    // Don't start loading for a slug that's already stale
    if (s !== slug()) return

    commentsController?.abort()

    const controller = new AbortController()
    commentsController = controller

    commentsLoading.value = true
    commentsError.value = ''

    try {
      const data = await fetchComments(s, controller.signal)
      if (commentsController !== controller || s !== slug()) return
      commentTree.value = data.items
    } catch (e: unknown) {
      if (commentsController !== controller || s !== slug()) return
      if (e instanceof DOMException && e.name === 'AbortError') return
      commentsError.value = e instanceof Error ? e.message : '加载评论失败'
    } finally {
      if (commentsController === controller) {
        commentsLoading.value = false
      }
    }
  }

  async function addComment(s: string, input: CreateCommentRequest) {
    await createComment(s, input)
    if (s !== slug()) return
    await loadComments(s)
  }

  async function removeComment(id: number, s: string) {
    await deleteComment(id)
    if (s !== slug()) return
    await loadComments(s)
  }

  // Reload when slug changes
  watch(
    slug,
    (newSlug, oldSlug) => {
      if (newSlug !== oldSlug) {
        post.value = null
        commentTree.value = []
      }
      loadPost(newSlug)
      loadComments(newSlug)
    },
    { immediate: true },
  )

  onUnmounted(() => {
    postController?.abort()
    commentsController?.abort()
  })

  return {
    post, postLoading, postError,
    commentTree, commentsLoading, commentsError,
    loadPost, loadComments, addComment, removeComment,
  }
}
