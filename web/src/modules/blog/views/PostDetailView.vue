<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../auth'
import PostArticle from '../components/PostArticle.vue'
import CommentForm from '../components/CommentForm.vue'
import CommentItem from '../components/CommentItem.vue'
import { usePostDetail } from '../composables/usePostDetail'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const slug = computed(() => {
  const value = route.params.slug
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '')
})
const {
  post, postLoading, postError,
  commentTree, commentsLoading, commentsError,
  addComment, removeComment,
} = usePostDetail(() => slug.value)

const deletingPost = ref(false)
const commentSubmitting = ref(false)
const commentError = ref('')
const commentText = ref('')
const replyToId = ref<number | null>(null)

const isAdmin = computed(() => auth.user?.role === 'admin')
const currentUserId = computed(() => auth.user?.id ?? null)

function startReply(id: number, username: string) {
  replyToId.value = id
  commentText.value = `@${username} `
}

function cancelReply() {
  replyToId.value = null
  commentText.value = ''
}

async function handleSubmitComment(content: string, parentId: number | null) {
  commentSubmitting.value = true
  commentError.value = ''
  try {
    await addComment(slug.value, { content, parent_id: parentId })
    commentText.value = ''
    replyToId.value = null
    // Refresh may have set commentsError — surface it
    if (commentsError.value) {
      commentError.value = '评论已提交，但列表刷新失败。'
    }
  } catch (e: unknown) {
    commentError.value = e instanceof Error ? e.message : '提交失败，请稍后重试。'
  } finally {
    commentSubmitting.value = false
  }
}

async function handleDeleteComment(id: number) {
  if (!confirm('确定删除这条评论？')) return
  try {
    await removeComment(id, slug.value)
    if (commentsError.value) {
      alert('评论已删除，但列表刷新失败。')
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '删除失败'
    alert(msg)
  }
}

async function handleDeletePost() {
  if (!confirm('确定删除这篇文章？')) return
  deletingPost.value = true
  try {
    const { deletePost } = await import('../api/posts')
    await deletePost(slug.value)
    await router.replace('/')
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '删除失败'
    alert(msg)
  } finally {
    deletingPost.value = false
  }
}
</script>

<template>
  <div class="page">
    <div v-if="postLoading && !post" class="reader">
      <span class="spinner"></span> 加载中…
    </div>

    <div v-else-if="postError && !post" class="reader">
      <RouterLink class="back" to="/">← 返回手账</RouterLink>
      <p style="color: var(--blaze)">{{ postError }}</p>
    </div>

    <template v-else-if="post">
      <PostArticle
        :post="post"
        :is-admin="isAdmin"
        :deleting="deletingPost"
        @delete="handleDeletePost"
      />

      <!-- Comments section -->
      <section class="comments comment-section">
        <h2>留言</h2>

        <CommentForm
          v-model:text="commentText"
          :is-logged-in="!!auth.user"
          :submitting="commentSubmitting"
          :error="commentError"
          :reply-to-id="replyToId"
          @submit="handleSubmitComment"
          @cancel-reply="cancelReply"
        />

        <p v-if="commentsError" style="color: var(--blaze); font-size: .85rem; margin: .5rem 0;">{{ commentsError }}</p>

        <div v-if="commentsLoading && commentTree.length === 0" class="empty">
          <span class="spinner"></span> 加载中…
        </div>

        <div v-else-if="commentTree.length === 0 && !commentsError" class="empty">暂无留言</div>

        <CommentItem
          v-for="c in commentTree"
          :key="c.id"
          :comment="c"
          :current-user-id="currentUserId"
          :is-admin="isAdmin"
          :show-reply="true"
          @reply="startReply"
          @delete="handleDeleteComment"
        />
      </section>
    </template>

    <div v-else class="reader">
      <RouterLink class="back" to="/">← 返回手账</RouterLink>
      <p style="color: var(--faded)">文章未找到。</p>
    </div>
  </div>
</template>

<style scoped>
.comment-section {
  padding-left: clamp(4.4rem, 13vw, 10.5rem);
  max-width: calc(var(--measure) + 12rem);
}
@media (max-width: 720px) {
  .comment-section { padding-left: 3.4rem; }
}
</style>
