<script setup lang="ts">
const props = defineProps<{
  isLoggedIn: boolean
  submitting: boolean
  error: string
  replyToId: number | null
}>()

const emit = defineEmits<{
  submit: [content: string, parentId: number | null]
  'cancel-reply': []
}>()

const text = defineModel<string>('text', { required: true })

function handleSubmit() {
  if (!text.value.trim()) return
  emit('submit', text.value, props.replyToId)
}
</script>

<template>
  <div v-if="isLoggedIn" class="comment-form">
    <textarea
      v-model="text"
      :placeholder="replyToId ? '写下回复…' : '写下留言…'"
      maxlength="4000"
    />
    <div class="row">
      <span v-if="replyToId" class="hint">
        回复中…
        <button style="background:none;border:0;color:var(--blaze);font-family:var(--mono);font-size:.65rem;cursor:pointer;text-transform:uppercase;letter-spacing:.08em" @click="$emit('cancel-reply')">取消</button>
      </span>
      <span v-else class="hint">最多 4000 字</span>
      <button :disabled="submitting || !text.trim()" @click="handleSubmit">
        {{ submitting ? '提交中…' : '提交' }}
      </button>
    </div>
    <p v-if="error" class="comment-error">{{ error }}</p>
  </div>
  <p v-else style="color: var(--faded); font-size: .9rem">
    <RouterLink to="/login" style="color: var(--pine)">登录</RouterLink> 后即可留言。
  </p>
</template>
