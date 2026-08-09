<script setup lang="ts">
import { ref } from 'vue'
import { formatJSON } from '../api/tools'
import SiteHeader from '../../../shared/components/SiteHeader.vue'
import SiteFooter from '../../../shared/components/SiteFooter.vue'

const input = ref('')
const output = ref('')
const error = ref('')
const loading = ref(false)

async function handleFormat() {
  if (!input.value.trim()) return
  loading.value = true
  error.value = ''
  output.value = ''
  try {
    const result = await formatJSON(input.value)
    output.value = result.formatted
  } catch (e: any) {
    error.value = e?.message ?? '格式化失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="site">
    <SiteHeader />
    <div class="page">
      <div class="tool-page">
        <RouterLink class="back" to="/tools">← 返回工具</RouterLink>
        <h1>JSON 格式化</h1>
        <p class="desc">粘贴一行或压缩过的 JSON，输出缩进 2 空格的格式化结果。</p>

        <textarea v-model="input" placeholder='{"hello":"world"}' />

        <div class="row">
          <span v-if="error" class="tool-error">{{ error }}</span>
          <span v-else />
          <button :disabled="loading || !input.trim()" @click="handleFormat">
            {{ loading ? '格式化中…' : '格式化' }}
          </button>
        </div>

        <div v-if="output" class="output">
          <pre>{{ output }}</pre>
        </div>
      </div>
    </div>
    <SiteFooter />
  </div>
</template>
