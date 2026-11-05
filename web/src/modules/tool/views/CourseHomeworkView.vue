<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { getHomework, listHomework, syncHomework } from '../api/homework'
import type { HomeworkDetail, HomeworkFilter, HomeworkList, HomeworkStatus } from '../api/homework'
import MISBindForm from '../components/MISBindForm.vue'

const data = ref<HomeworkList | null>(null)
const course = ref('')
const filter = ref<HomeworkFilter>('todo')
const page = ref(1)
const loading = ref(false)
const syncing = ref(false)
const error = ref('')
const notice = ref('')
const expanded = ref<number | null>(null)
const detail = ref<HomeworkDetail | null>(null)
const detailLoading = ref(false)
const detailError = ref('')
let queryController: AbortController | undefined
let detailController: AbortController | undefined
const syncController = new AbortController()
let disposed = false
const labels: Record<HomeworkStatus, string> = { pending: '待处理', overdue: '已逾期', submitted: '已提交', upcoming: '未开始', unknown: '状态未知' }
const statuses: HomeworkStatus[] = ['pending', 'overdue', 'submitted', 'upcoming', 'unknown']
const totalCount = computed(() => statuses.reduce((total, status) => total + (data.value?.counts[status] ?? 0), 0))
const groups = computed(() => statuses.map(status => ({ status, items: data.value?.items.filter(item => item.status === status) ?? [] })).filter(group => group.items.length))
const pages = computed(() => Math.max(1, Math.ceil((data.value?.total ?? 0) / 20)))
const termName = computed(() => {
  const code = data.value?.term_code ?? ''
  return /^\d{8}0[12]$/.test(code) ? `${code.slice(0, 4)}–${code.slice(4, 8)} 学年第${code.endsWith('1') ? '一' : '二'}学期` : '当前学期'
})
function time(value: string | null | undefined) {
  return value ? new Date(value).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) : '—'
}
function message(e: unknown) { return e instanceof Error ? e.message : '请求失败，请稍后重试' }

async function load() {
  if (disposed) return
  queryController?.abort()
  const controller = new AbortController()
  queryController = controller
  loading.value = true
  error.value = ''
  try {
    const result = await listHomework({ course_id: course.value, status: filter.value, page: page.value }, controller.signal)
    if (!controller.signal.aborted) data.value = result
  } catch (e) {
    if (!controller.signal.aborted) error.value = message(e)
  } finally {
    if (queryController === controller) loading.value = false
  }
}
async function synchronize() {
  if (syncing.value || disposed) return
  syncing.value = true
  error.value = ''
  notice.value = ''
  expanded.value = null
  detailController?.abort()
  detail.value = null
  try {
    const result = await syncHomework(syncController.signal)
    notice.value = result.status === 'success' ? `已更新 ${result.succeeded} 门课程` : `更新成功 ${result.succeeded} 门，失败 ${result.failed} 门；失败课程保留上次结果`
  } catch (e) {
    if (!syncController.signal.aborted) notice.value = message(e)
  } finally {
    await load()
    syncing.value = false
  }
}
async function bound() {
  queryController?.abort()
  detailController?.abort()
  expanded.value = null
  detail.value = null
  data.value = null
  course.value = ''
  page.value = 1
  await load()
  await synchronize()
}
async function toggle(id: number) {
  detailController?.abort()
  detail.value = null
  detailError.value = ''
  if (expanded.value === id) { expanded.value = null; return }
  expanded.value = id
  const controller = new AbortController()
  detailController = controller
  detailLoading.value = true
  try {
    const result = await getHomework(id, controller.signal)
    if (!controller.signal.aborted) detail.value = result
  } catch (e) {
    if (!controller.signal.aborted) detailError.value = message(e)
  } finally {
    if (detailController === controller) detailLoading.value = false
  }
}
watch([course, filter], () => { page.value = 1 })
watch([course, filter, page], () => { expanded.value = null; detailController?.abort(); void load() })
onMounted(async () => { await load(); if (data.value?.bound && !data.value.has_snapshot) await synchronize() })
onBeforeUnmount(() => { disposed = true; queryController?.abort(); detailController?.abort(); syncController.abort() })
</script>

<template>
  <div class="homework-page">
    <header class="homework-heading">
      <div><span class="eyebrow">课程作业</span><h1>还有哪些作业要交？</h1></div>
      <button type="button" :disabled="syncing || !data?.bound" @click="synchronize">{{ syncing ? '正在同步…' : '刷新作业' }}</button>
    </header>
    <MISBindForm :bound="data?.bound ?? true" :current-student-id="data?.student_id" :disabled="syncing" @bound="bound" />
    <p v-if="error" role="alert" class="error">{{ error }} <button type="button" @click="load">重试</button></p>
    <p v-if="notice" role="status">{{ notice }}</p>
    <p v-if="syncing" role="status">正在读取课程平台。旧结果仍可查看，请稍候。</p>
    <template v-if="data?.bound">
      <p class="sync-time">{{ termName }} · 最近完整同步：{{ time(data.last_success_at) }}<span v-if="!data.last_success_at"> · 暂无完整同步记录</span></p>
      <ul v-if="data.courses.some(item => item.error)" class="failures">
        <li v-for="item in data.courses.filter(item => item.error)" :key="item.id">{{ item.name }}：{{ item.error }}；上次成功 {{ time(item.last_success_at) }}</li>
      </ul>
      <div class="filters">
        <label>课程<select v-model="course"><option value="">全部课程</option><option v-for="item in data.courses" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
        <label>状态<select v-model="filter"><option value="todo">待处理与逾期未交（{{ data.counts.pending + data.counts.overdue }}）</option><option v-for="status in statuses" :key="status" :value="status">{{ labels[status] }}（{{ data.counts[status] }}）</option><option value="all">全部作业（{{ totalCount }}）</option></select></label>
        <span v-if="loading" role="status">加载中…</span>
      </div>
      <section v-for="group in groups" :key="group.status" class="homework-group">
        <h2>{{ labels[group.status] }}</h2>
        <article v-for="item in group.items" :key="item.id" class="homework-card" :class="group.status">
          <p class="course-name">{{ item.course_name }}</p>
          <button type="button" class="title" :aria-expanded="expanded === item.id" @click="toggle(item.id)">{{ item.title }}</button>
          <p class="meta"><span>{{ item.deadline ? `截止：${time(item.deadline)}` : '无截止时间' }}</span><span>{{ item.submission || '提交状态未知' }}</span></p>
          <div v-if="expanded === item.id" class="detail">
            <p v-if="detailLoading" role="status">正在读取详情…</p>
            <p v-if="detailError" class="error" role="alert">{{ detailError }}</p>
            <template v-if="detail">
              <p v-if="detail.warning" role="status">{{ detail.warning }}</p>
              <p>开放：{{ time(item.opens_at) }} · 提交：{{ time(item.submitted_at) }}</p>
              <p>批改：{{ item.grading || '未公布' }}</p>
              <p class="content">{{ detail.content || '暂无正文' }}</p>
              <h3>附件</h3><ul v-if="detail.attachments.length"><li v-for="(attachment, index) in detail.attachments" :key="index">{{ attachment.name }}</li></ul><p v-else>{{ detail.warning ? '附件信息暂不可用' : '无附件' }}</p>
            </template>
          </div>
        </article>
      </section>
      <p v-if="!loading && !syncing && !data.items.length">{{ data.has_snapshot ? '当前筛选下没有作业。' : '暂无作业快照，请刷新获取。' }}</p>
      <nav v-if="data.total > 20" class="pager" aria-label="作业分页"><button type="button" :disabled="page <= 1" @click="page--">上一页</button><span>{{ page }} / {{ pages }}</span><button type="button" :disabled="page >= pages" @click="page++">下一页</button></nav>
    </template>
    <p v-else-if="data">绑定自己的 MIS 账号后即可同步课程作业，无需图鉴账号。</p>
  </div>
</template>

<style scoped>
.homework-page { max-width: 70rem; }
.homework-heading { display: flex; align-items: center; justify-content: space-between; gap: 1rem; flex-wrap: wrap; }
.homework-heading h1 { margin: .6rem 0; }
button, select { padding: .6rem .8rem; border: 1px solid var(--rule); border-radius: 4px; background: var(--paper); color: inherit; }
button { cursor: pointer; } button:disabled { opacity: .5; cursor: default; }
.sync-time, .meta, .course-name { color: var(--faded); font-size: .9rem; }
.filters, .pager { display: flex; gap: 1rem; flex-wrap: wrap; align-items: center; margin: 1.5rem 0; }
.filters label { display: grid; gap: .4rem; }
.homework-group { margin: 2rem 0; }.homework-group h2 { font-size: 1.2rem; }
.homework-card { padding: 1rem 1.2rem; border: 1px solid var(--rule); border-radius: 4px; margin: .8rem 0; }
.homework-card.overdue { border-left: 3px solid #ac6043; }
.course-name { margin: 0 0 .4rem; }.title { font-size: 1.1rem; font-weight: 600; text-align: left; border: 0; padding: .2rem 0; }
.meta { display: flex; gap: 1rem; flex-wrap: wrap; }.detail { margin-top: 1rem; border-top: 1px solid var(--rule); padding-top: .6rem; }
.content { white-space: pre-wrap; overflow-wrap: anywhere; line-height: 1.8; }.failures,.error { color: #a33428; }.failures { line-height: 1.8; }
@media (max-width: 600px) { .homework-heading h1 { font-size: 1.6rem; }.filters label,.filters select { width: 100%; } }
</style>
