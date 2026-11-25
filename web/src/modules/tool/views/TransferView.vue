<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { APIRequestError } from '../../../shared/api/http'
import {
  createTransferChannel, createTransferText, deleteTransferChannel, deleteTransferItem,
  downloadTransferFile, getTransferItem, getTransferUsage, listTransferChannels,
  listTransferItems, renameTransferChannel, uploadTransferFile,
} from '../api/transfer'
import type { TransferChannel, TransferItem, TransferKind, TransferUsage } from '../api/transfer'

const channels = ref<TransferChannel[]>([])
const selectedID = ref<number | null>(null)
const selected = computed(() => channels.value.find((channel) => channel.id === selectedID.value))
const items = ref<TransferItem[]>([])
const usage = ref<TransferUsage | null>(null)
const page = ref(1)
const kind = ref<TransferKind>('')
const hasMore = ref(false)
const loading = ref(true)
const sending = ref(false)
const error = ref('')
const notice = ref('')
const paused = ref(false)
const sendMode = ref<'text' | 'file'>('text')
const drafts = reactive<Record<number, string>>({})
const draft = computed({
  get: () => selectedID.value === null ? '' : (drafts[selectedID.value] ?? ''),
  set: (value: string) => { if (selectedID.value !== null) drafts[selectedID.value] = value },
})
const textBytes = computed(() => new TextEncoder().encode(draft.value).length)
const fullText = reactive<Record<number, string>>({})
const expanded = reactive<Record<number, boolean>>({})
const working = reactive(new Set<number>())
const fileInput = ref<HTMLInputElement | null>(null)
const channelDialog = ref<HTMLDialogElement | null>(null)
const channelName = ref('')
const editingChannelID = ref<number | null>(null)
const channelSaving = ref(false)
const channelError = ref('')
type UploadEntry = { file: File; channelID: number; channelName: string; status: 'waiting' | 'uploading' | 'done' | 'failed'; error: string }
const uploads = ref<UploadEntry[]>([])
let processingUploads = false
let refreshVersion = 0
let polling = false
let disposed = false
let timer: number | undefined

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`
}
function formatTime(value: string) { return new Date(value).toLocaleString('zh-CN', { hour12: false }) }

function report(err: unknown) {
  error.value = err instanceof Error ? err.message : '请求失败，请稍后重试。'
  if (err instanceof APIRequestError && (err.status === 401 || err.status === 403)) {
    paused.value = true
    error.value = '登录已失效或管理员权限已变更，请重新登录后刷新。当前页面的文本草稿仍保留。'
  }
}

async function loadItems() {
  const version = ++refreshVersion
  const id = selectedID.value
  if (id === null) { items.value = []; loading.value = false; return }
  try {
    const result = await listTransferItems(id, page.value, kind.value)
    if (version !== refreshVersion || disposed) return
    items.value = result.items
    hasMore.value = result.has_more
    // Keep expanded full text across polling, but release removed/expired content.
    const visible = new Set(result.items.map((item) => item.id))
    for (const key of Object.keys(fullText)) {
      if (!visible.has(Number(key))) { delete fullText[Number(key)]; delete expanded[Number(key)] }
    }
    if (!items.value.length && page.value > 1) page.value--
  } catch (err) {
    if (version === refreshVersion) report(err)
  } finally {
    if (version === refreshVersion) loading.value = false
  }
}

async function refresh(manual = false) {
  if (polling || disposed || (!manual && paused.value)) return
  polling = true
  if (manual) { paused.value = false; error.value = '' }
  try {
    const [channelResult, usageResult] = await Promise.all([listTransferChannels(), getTransferUsage()])
    if (disposed) return
    channels.value = channelResult.items
    usage.value = usageResult
    if (!channels.value.some((channel) => channel.id === selectedID.value)) {
      if (selectedID.value !== null) notice.value = '原通道已被删除，已切换到可用通道。'
      selectedID.value = channels.value[0]?.id ?? null
      page.value = 1
    }
    await loadItems()
  } catch (err) { report(err) } finally {
    polling = false
    loading.value = false
    if (manual && !paused.value) void processUploads()
  }
}

function selectChannel(id: number) {
  if (id === selectedID.value) return
  refreshVersion++
  items.value = []
  selectedID.value = id
  page.value = 1
  loading.value = true
  void loadItems()
}
watch([page, kind], () => { loading.value = true; void loadItems() })

function openChannelForm(channel?: TransferChannel) {
  editingChannelID.value = channel?.id ?? null
  channelName.value = channel?.name ?? ''
  channelError.value = ''
  channelDialog.value?.showModal()
}
async function saveChannel() {
  if (channelSaving.value) return
  channelSaving.value = true
  channelError.value = ''
  try {
    const channel = editingChannelID.value === null
      ? await createTransferChannel(channelName.value)
      : await renameTransferChannel(editingChannelID.value, channelName.value)
    channelDialog.value?.close()
    selectedID.value = channel.id
    page.value = 1
    await refresh(true)
  } catch (err) {
    channelError.value = err instanceof Error ? err.message : '保存失败'
    if (err instanceof APIRequestError && (err.status === 401 || err.status === 403)) report(err)
  } finally { channelSaving.value = false }
}
async function removeChannel() {
  const channel = selected.value
  if (!channel || !window.confirm(`删除通道“${channel.name}”？其中 ${channel.file_count} 个文件和 ${channel.text_count} 条文本都将删除，且不可恢复。`)) return
  try {
    await deleteTransferChannel(channel.id)
    delete drafts[channel.id]
    notice.value = '通道及其中内容已删除。'
    await refresh(true)
  } catch (err) { report(err) }
}

async function sendText() {
  const id = selectedID.value
  const text = draft.value
  if (id === null || sending.value || paused.value || !text.trim() || textBytes.value > 100 * 1024) return
  sending.value = true
  error.value = ''
  try {
    await createTransferText(id, text)
    if (drafts[id] === text) drafts[id] = ''
    notice.value = '文本已发送，保存 15 天。'
    if (selectedID.value === id) page.value = 1
    await refresh()
  } catch (err) { report(err) } finally { sending.value = false }
}

function addFiles(files: FileList | File[]) {
  const channel = selected.value
  if (!channel || paused.value) return
  for (const file of Array.from(files)) {
    uploads.value.push({ file, channelID: channel.id, channelName: channel.name,
      status: file.size > 100 * 1024 * 1024 ? 'failed' : 'waiting',
      error: file.size > 100 * 1024 * 1024 ? '文件超过 100 MiB' : '' })
  }
  void processUploads()
}
function chooseFiles(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files) addFiles(input.files)
  input.value = ''
}
function dropFiles(event: DragEvent) { if (event.dataTransfer?.files) addFiles(event.dataTransfer.files) }
async function processUploads() {
  if (processingUploads) return
  processingUploads = true
  try {
    while (!paused.value && !disposed) {
      const entry = uploads.value.find((candidate) => candidate.status === 'waiting')
      if (!entry) break
      entry.status = 'uploading'
      try {
        await uploadTransferFile(entry.channelID, entry.file)
        entry.status = 'done'
        await refresh()
      } catch (err) {
        entry.status = 'failed'
        entry.error = err instanceof Error ? err.message : '上传失败'
        report(err)
      }
    }
  } finally { processingUploads = false }
}
function retryUpload(entry: UploadEntry) {
  if (entry.file.size > 100 * 1024 * 1024 || paused.value) return
  entry.status = 'waiting'
  entry.error = ''
  void processUploads()
}

async function expand(item: TransferItem) {
  if (expanded[item.id]) { expanded[item.id] = false; return }
  if (fullText[item.id] !== undefined) { expanded[item.id] = true; return }
  working.add(item.id)
  try {
    const detail = await getTransferItem(item.id)
    fullText[item.id] = detail.text ?? ''
    expanded[item.id] = true
    notice.value = '全文已加载，可点击复制。'
  } catch (err) { report(err); await refresh() } finally { working.delete(item.id) }
}
async function copy(item: TransferItem) {
  working.add(item.id)
  try {
    if (fullText[item.id] !== undefined) {
      await navigator.clipboard.writeText(fullText[item.id])
    } else {
      const text = getTransferItem(item.id).then((detail) => {
        fullText[item.id] = detail.text ?? ''
        expanded[item.id] = true
        return fullText[item.id]
      })
      void text.catch(() => {})
      // Start the clipboard operation during the click, before awaiting the API.
      if (typeof ClipboardItem !== 'undefined' && navigator.clipboard?.write) {
        await navigator.clipboard.write([new ClipboardItem({ 'text/plain': text.then((value) => new Blob([value], { type: 'text/plain' })) })])
      } else {
        await navigator.clipboard.writeText(await text)
      }
    }
    notice.value = '已复制完整原文。'
  } catch (err) {
    if (err instanceof APIRequestError) { report(err); await refresh() }
    else error.value = '浏览器未允许复制，请展开全文后手动选中复制。'
  } finally { working.delete(item.id) }
}
async function download(item: TransferItem) {
  working.add(item.id)
  try { await downloadTransferFile(item); notice.value = '文件已交给浏览器下载。' }
  catch (err) { report(err); await refresh() } finally { working.delete(item.id) }
}
async function removeItem(item: TransferItem) {
  if (!window.confirm(`删除${item.kind === 'file' ? `文件“${item.filename}”` : '这条文本'}？此操作不可恢复。`)) return
  working.add(item.id)
  try { await deleteTransferItem(item.id); notice.value = '内容已删除。'; await refresh() }
  catch (err) { report(err) } finally { working.delete(item.id) }
}

function visibleRefresh() { if (!document.hidden) void refresh() }
onMounted(() => {
  void refresh()
  timer = window.setInterval(visibleRefresh, 5000)
  document.addEventListener('visibilitychange', visibleRefresh)
})
onBeforeUnmount(() => {
  disposed = true
  refreshVersion++
  window.clearInterval(timer)
  document.removeEventListener('visibilitychange', visibleRefresh)
})
</script>

<template>
  <div class="page transfer-page">
    <header class="transfer-intro">
      <span class="eyebrow">管理员工具 · 同账号共享</span>
      <h1>传输通道</h1>
      <p class="lede">把文件和文字留在这里，在另一台电脑接着用。</p>
      <p class="transfer-hint">文件和文本保存 15 天后自动清理，通道长期保留。单文件最多 100 MiB，文本最多 100 KiB，不设总容量上限。</p>
      <p v-if="usage" class="transfer-hint">已保存内容 {{ formatBytes(usage.used_bytes) }}，含等待清理的内容。</p>
    </header>
    <p v-if="error" role="alert" class="transfer-error">{{ error }}</p>
    <p v-if="notice" role="status" class="transfer-notice">{{ notice }}</p>
    <div v-if="paused" class="transfer-actions">
      <a href="/login?redirect=/tools/transfer" target="_blank" rel="noopener">在新标签页登录</a>
      <button type="button" @click="refresh(true)">登录后重新连接</button>
    </div>

    <div class="transfer-layout">
      <aside class="transfer-sidebar" aria-label="通道列表">
        <button type="button" :disabled="paused" @click="openChannelForm()">＋ 新建通道</button>
        <p v-if="!channels.length" class="transfer-hint">创建一个通道，开始传输。</p>
        <button v-for="channel in channels" :key="channel.id" type="button" class="transfer-channel"
          :class="{ selected: selectedID === channel.id }" :aria-pressed="selectedID === channel.id" @click="selectChannel(channel.id)">
          <strong>{{ channel.name }}</strong>
          <small>{{ channel.file_count }} 个文件 · {{ channel.text_count }} 条文本</small>
        </button>
      </aside>

      <main class="transfer-main">
        <template v-if="selected">
          <div class="transfer-heading">
            <h2>{{ selected.name }}</h2>
            <div class="transfer-actions">
              <button type="button" @click="refresh(true)">刷新</button>
              <button type="button" :disabled="paused" @click="openChannelForm(selected)">重命名</button>
              <button type="button" class="danger" :disabled="paused" @click="removeChannel">删除通道</button>
            </div>
          </div>
          <section class="transfer-composer" aria-label="发送内容">
            <div class="transfer-actions">
              <button type="button" :aria-pressed="sendMode === 'text'" @click="sendMode = 'text'">文本</button>
              <button type="button" :aria-pressed="sendMode === 'file'" @click="sendMode = 'file'">文件</button>
            </div>
            <form v-if="sendMode === 'text'" @submit.prevent="sendText">
              <label for="transfer-text">命令或纯文本</label>
              <textarea id="transfer-text" v-model="draft" rows="6" spellcheck="false" placeholder="在这里粘贴命令、配置或文字，保留原始格式。"
                :disabled="paused" @keydown.ctrl.enter.prevent="sendText" />
              <div class="transfer-heading">
                <small :class="{ 'transfer-error': textBytes > 102400 }">{{ formatBytes(textBytes) }} / 100 KiB · Ctrl + Enter 发送</small>
                <button type="submit" :disabled="paused || sending || !draft.trim() || textBytes > 102400">{{ sending ? '发送中…' : '发送文本' }}</button>
              </div>
            </form>
            <div v-else class="transfer-drop" @dragover.prevent @drop.prevent="dropFiles">
              <p>拖入文件，或选择文件上传</p>
              <p class="transfer-hint">可多选，逐个上传；相同文件名不会覆盖。</p>
              <input ref="fileInput" type="file" multiple aria-label="选择上传文件" :disabled="paused" @change="chooseFiles">
            </div>
          </section>
          <div class="transfer-heading">
            <label>内容类型
              <select v-model="kind" @change="page = 1">
                <option value="">全部</option><option value="file">文件</option><option value="text">文本</option>
              </select>
            </label>
            <small class="transfer-hint">页面可见时每 5 秒刷新</small>
          </div>
          <p v-if="loading && !items.length" class="empty">正在加载…</p>
          <p v-else-if="!items.length" class="empty">这个通道还没有{{ kind === 'file' ? '文件' : kind === 'text' ? '文本' : '内容' }}。</p>
          <article v-for="item in items" :key="item.id" class="transfer-item">
            <div class="transfer-heading">
              <strong>{{ item.kind === 'file' ? item.filename : '纯文本' }}</strong>
              <small>{{ formatBytes(item.size_bytes) }}</small>
            </div>
            <small class="transfer-hint">{{ formatTime(item.created_at) }} · 到期 {{ formatTime(item.expires_at) }}</small>
            <pre v-if="item.kind === 'text'" :class="{ 'transfer-summary': !expanded[item.id] }">{{ expanded[item.id] ? fullText[item.id] : item.summary }}</pre>
            <div class="transfer-actions">
              <template v-if="item.kind === 'text'">
                <button type="button" :disabled="working.has(item.id) || paused" @click="expand(item)">{{ expanded[item.id] ? '收起全文' : '展开全文' }}</button>
                <button type="button" :disabled="working.has(item.id) || paused" @click="copy(item)">复制全文</button>
              </template>
              <button v-else type="button" :disabled="working.has(item.id) || paused" @click="download(item)">{{ working.has(item.id) ? '处理中…' : '下载文件' }}</button>
              <button type="button" class="danger" :disabled="working.has(item.id) || paused" @click="removeItem(item)">删除</button>
            </div>
          </article>
          <nav class="transfer-actions" aria-label="内容分页">
            <button type="button" :disabled="page <= 1 || loading" @click="page--">上一页</button>
            <span>第 {{ page }} 页</span>
            <button type="button" :disabled="!hasMore || loading" @click="page++">下一页</button>
          </nav>
        </template>
        <div v-else class="empty">选择或新建一个通道，在两台电脑之间传递内容。</div>
      </main>
    </div>

    <section v-if="uploads.length" class="transfer-queue" aria-label="上传队列">
      <div class="transfer-heading"><h2>上传队列</h2><button type="button" @click="uploads = uploads.filter((entry) => entry.status !== 'done')">移除成功记录</button></div>
      <div v-for="(entry, index) in uploads" :key="index" class="transfer-upload">
        <span>{{ entry.file.name }} → {{ entry.channelName }} · {{ formatBytes(entry.file.size) }}</span>
        <span role="status">{{ { waiting: '等待上传', uploading: '上传中…', done: '上传成功', failed: '上传失败' }[entry.status] }}</span>
        <small v-if="entry.error" class="transfer-error">{{ entry.error }}</small>
        <button v-if="entry.status === 'failed'" type="button" :disabled="paused || entry.file.size > 104857600" @click="retryUpload(entry)">重试</button>
      </div>
    </section>

    <dialog ref="channelDialog" class="transfer-dialog" @cancel="channelError = ''">
      <form @submit.prevent="saveChannel">
        <h2>{{ editingChannelID === null ? '新建通道' : '重命名通道' }}</h2>
        <label for="transfer-channel-name">通道名称</label>
        <input id="transfer-channel-name" v-model="channelName" maxlength="50" required autofocus autocomplete="off">
        <p v-if="channelError" role="alert" class="transfer-error">{{ channelError }}</p>
        <div class="transfer-actions">
          <button type="button" :disabled="channelSaving" @click="channelDialog?.close()">取消</button>
          <button type="submit" :disabled="channelSaving || !channelName.trim()">{{ channelSaving ? '保存中…' : '保存' }}</button>
        </div>
      </form>
    </dialog>
  </div>
</template>

<style scoped>
.transfer-page { max-width: 1200px; }
.transfer-page::before { display: none; }
.transfer-intro { margin-bottom: 1.5rem; }
.transfer-hint { color: var(--faded); font-size: .85rem; }
.transfer-layout { display: grid; grid-template-columns: 220px minmax(0, 1fr); gap: 2rem; }
.transfer-sidebar { border-right: 1px solid var(--rule); padding-right: 1rem; }
.transfer-channel { display: block; width: 100%; text-align: left; margin-top: .7rem; overflow-wrap: anywhere; }
.transfer-channel strong, .transfer-channel small { display: block; }
.transfer-channel.selected { border-color: var(--pine); background: var(--paper-2); }
.transfer-heading, .transfer-actions { display: flex; align-items: center; gap: .6rem; flex-wrap: wrap; }
.transfer-heading { justify-content: space-between; margin-bottom: .8rem; }
.transfer-heading h2 { margin: 0; overflow-wrap: anywhere; }
.transfer-main { min-width: 0; }
.transfer-page button, .transfer-page select { font: inherit; padding: .4rem .7rem; border: 1px solid var(--rule); border-radius: 5px; background: transparent; color: inherit; cursor: pointer; }
.transfer-page button:disabled { opacity: .5; cursor: default; }
.transfer-page button[aria-pressed="true"] { border-color: var(--pine); color: var(--pine); }
.transfer-page .danger, .transfer-error { color: #a5362b; }
.transfer-notice { color: var(--pine); }
.transfer-composer { padding: 1rem; border: 1px solid var(--rule); border-radius: 8px; margin-bottom: 1.5rem; }
.transfer-composer form { margin-top: .8rem; }
.transfer-composer label, .transfer-dialog label { display: block; margin-bottom: .4rem; }
.transfer-composer textarea { width: 100%; resize: vertical; min-height: 130px; padding: .7rem; font-family: ui-monospace, Consolas, monospace; font-size: .9rem; color: inherit; background: transparent; border: 1px solid var(--rule); border-radius: 4px; }
.transfer-drop { padding: 1.5rem .5rem; text-align: center; border: 1px dashed var(--rule); margin-top: .8rem; }
.transfer-drop input { max-width: 100%; }
.transfer-item { border-top: 1px solid var(--rule); padding: 1rem 0; }
.transfer-item strong { overflow-wrap: anywhere; }
.transfer-item pre { white-space: pre-wrap; overflow-wrap: anywhere; tab-size: 4; font-family: ui-monospace, Consolas, monospace; font-size: .85rem; max-height: 32rem; overflow-y: auto; user-select: text; }
.transfer-summary { max-height: 6rem !important; overflow: hidden !important; }
.transfer-item .transfer-actions { margin-top: .8rem; }
.transfer-queue { margin-top: 2rem; border-top: 1px solid var(--rule); padding-top: 1rem; }
.transfer-upload { display: flex; align-items: center; gap: .8rem; flex-wrap: wrap; border-top: 1px solid var(--rule); padding: .6rem 0; overflow-wrap: anywhere; }
.transfer-dialog { width: min(90vw, 420px); background: var(--paper, #faf8f3); color: inherit; border: 1px solid var(--rule); border-radius: 8px; padding: 1.5rem; }
.transfer-dialog::backdrop { background: #0006; }
.transfer-dialog input { width: 100%; padding: .5rem; margin-bottom: 1rem; font: inherit; }
@media (max-width: 700px) {
  .transfer-layout { grid-template-columns: minmax(0, 1fr); gap: 1rem; }
  .transfer-sidebar { border-right: 0; border-bottom: 1px solid var(--rule); padding: 0 0 1rem; }
}
</style>
