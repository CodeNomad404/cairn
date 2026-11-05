<script setup lang="ts">
import { computed, ref } from 'vue'
import { bindMIS } from '../api/homework'

const props = defineProps<{ bound: boolean; currentStudentId?: string; disabled?: boolean }>()
const emit = defineEmits<{ bound: [] }>()
const studentId = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const success = ref('')
const switching = ref(false)
const savedStudentId = ref('')
const currentStudentId = computed(() => savedStudentId.value || props.currentStudentId)

function toggleSwitch() {
  switching.value = !switching.value
  password.value = ''
  error.value = ''
  success.value = ''
}

async function submit() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const result = await bindMIS({ student_id: studentId.value, password: password.value })
    savedStudentId.value = result.student_id
    switching.value = false
    success.value = `已绑定 ${result.student_id}`
    emit('bound')
  } catch (e) {
    const message = e instanceof Error ? e.message : '绑定失败，请稍后重试'
    error.value = message === 'BJTU login failed' ? 'MIS 账号或密码被拒绝，请检查后重试' : message
  } finally {
    password.value = ''
    loading.value = false
  }
}
</script>

<template>
  <section class="mis-binding">
    <div v-if="bound" class="mis-account">
      <span>{{ currentStudentId ? `学号：${currentStudentId}` : 'MIS 账号已绑定' }}</span>
      <button class="switch-account" type="button" :disabled="loading || disabled" :aria-expanded="switching" @click="toggleSwitch">{{ switching ? '取消切换' : '切换账号' }}</button>
    </div>
    <template v-if="!bound || switching">
      <p>使用你自己的北交大 MIS 账号，已绑定的信息可供课程作业和抢课工具共用。</p>
      <form @submit.prevent="submit">
        <label>学号<input v-model.trim="studentId" autocomplete="username" maxlength="64" :disabled="loading || disabled" required /></label>
        <label>MIS 密码<input v-model="password" type="password" autocomplete="current-password" maxlength="256" :disabled="loading || disabled" required /></label>
        <button type="submit" :disabled="loading || disabled">{{ loading ? '验证中…' : '验证并绑定' }}</button>
      </form>
    </template>
    <p v-if="error" class="mis-error" role="alert">{{ error }}</p>
    <p v-if="success" role="status">{{ success }}</p>
  </section>
</template>

<style scoped>
.mis-binding { margin: 1rem 0; }
.mis-account { display: flex; flex-wrap: wrap; align-items: center; gap: .75rem; color: var(--faded); font-size: .875rem; }
.switch-account { padding: .25rem .65rem; border: 1px solid var(--rule); border-radius: 4px; background: transparent; color: var(--ink); font: inherit; font-size: .8rem; line-height: 1.5; transition: background-color .15s, border-color .15s; }
.switch-account:hover:not(:disabled) { background: var(--paper-2); border-color: var(--binding); }
.switch-account:focus-visible { outline: 2px solid var(--pine); outline-offset: 3px; }
.switch-account:disabled { opacity: .5; cursor: not-allowed; }
p { margin: .8rem 0; color: var(--faded); }
form { display: flex; flex-wrap: wrap; align-items: end; gap: 1rem; }
label { display: grid; gap: .4rem; }
input { min-width: 12rem; padding: .6rem; border: 1px solid var(--rule); border-radius: 4px; background: var(--paper); color: inherit; }
button { padding: .6rem 1rem; cursor: pointer; }
.mis-error { color: #a33428; }
</style>
