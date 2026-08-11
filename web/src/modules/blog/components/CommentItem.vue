<script setup lang="ts">
import { computed } from 'vue'
import type { CommentResult } from '../types'

const props = defineProps<{
  comment: CommentResult
  currentUserId: number | null
  isAdmin: boolean
  showReply: boolean
}>()

defineEmits<{
  reply: [id: number, username: string]
  delete: [id: number]
}>()

const canDelete = computed(() => {
  if (!props.currentUserId) return false
  return props.isAdmin || props.currentUserId === props.comment.user_id
})

function formatDate(d: string | null) {
  if (!d) return ''
  return new Date(d).toISOString().slice(0, 10)
}
</script>

<template>
  <div class="comment">
    <div class="by">
      <span class="name">{{ comment.username }}</span>
      · {{ formatDate(comment.created_at) }}
    </div>
    <div class="body">{{ comment.content }}</div>
    <div class="actions">
      <button v-if="showReply" @click="$emit('reply', comment.id, comment.username)">回复</button>
      <button v-if="canDelete" @click="$emit('delete', comment.id)">删除</button>
    </div>
    <div v-if="comment.replies && comment.replies.length" class="replies">
      <CommentItem
        v-for="r in comment.replies"
        :key="r.id"
        :comment="r"
        :current-user-id="currentUserId"
        :is-admin="isAdmin"
        :show-reply="false"
        @delete="(id: number) => $emit('delete', id)"
        @reply="(id: number, username: string) => $emit('reply', id, username)"
      />
    </div>
  </div>
</template>
