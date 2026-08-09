<script setup lang="ts">
import { computed, ref } from 'vue'
import { formatJSON } from '../api/tools'
import JsonTreeNode from '../../../shared/components/JsonTreeNode.vue'
import SiteHeader from '../../../shared/components/SiteHeader.vue'
import SiteFooter from '../../../shared/components/SiteFooter.vue'

const input = ref('')
const output = ref('')
const error = ref('')
const loading = ref(false)
const maxDepth = ref(2)

const parsedOutput = computed(() => {
  if (!output.value) return null
  try {
    return JSON.parse(output.value)
  } catch {
    return null
  }
})

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
        <p class="desc">粘贴一行或压缩过的 JSON，输出可展开折叠的树形结构。</p>

        <textarea v-model="input" placeholder='{"hello":"world"}' />

        <div class="row">
          <label class="depth-label">
            展开层级
            <select v-model.number="maxDepth">
              <option :value="1">1</option>
              <option :value="2">2</option>
              <option :value="3">3</option>
              <option :value="4">4</option>
              <option :value="99">全部</option>
            </select>
          </label>
          <span v-if="error" class="tool-error">{{ error }}</span>
          <span v-else />
          <button :disabled="loading || !input.trim()" @click="handleFormat">
            {{ loading ? '格式化中…' : '格式化' }}
          </button>
        </div>

        <div v-if="parsedOutput" class="output tree">
          <JsonTreeNode
            key-name="root"
            :value="parsedOutput"
            :depth="0"
            :max-depth="maxDepth"
          />
        </div>
      </div>
    </div>
    <SiteFooter />
  </div>
</template>

<style scoped>
.depth-label {
  display: flex; align-items: center; gap: .4rem;
  font-family: var(--mono); font-size: .68rem; color: var(--faded); cursor: pointer;
}
.depth-label select {
  border: 1px solid var(--binding); border-radius: 3px;
  padding: .15rem .4rem; font: inherit;
  background: var(--paper); color: var(--ink); cursor: pointer;
}
.tree {
  padding: 1rem 1.2rem;
  overflow-x: auto;
}
</style>
