<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useAuthStore } from '../../../modules/auth'
import { APIRequestError } from '../../../shared/api/http'
import { grabCourses, listCourses, loginCourseRegistration } from '../api/tools'
import { useCourseGrabTasks } from '../composables/useCourseGrabTasks'
import type { CourseResult } from '../types'
import MISBindForm from '../components/MISBindForm.vue'

const authStore = useAuthStore()

// 勾选持久化：按 Cairn 用户隔离，存 localStorage，刷新后恢复上次勾选
function selectionStorageKey(): string {
  return `cairn:tool:course-selection:${authStore.user?.id ?? ''}`
}

function restoreSelection(): string[] {
  try {
    const raw = localStorage.getItem(selectionStorageKey())
    const parsed: unknown = raw ? JSON.parse(raw) : []
    if (!Array.isArray(parsed)) return []
    return parsed.filter((value): value is string => typeof value === 'string').slice(0, 50)
  } catch {
    return []
  }
}

// ---- 教务登录 ----
const tujianUsername = ref('')
const tujianPassword = ref('')
const loginLoading = ref(false)
const loginError = ref('')
/** null = 正在探测会话；true = 已建立 BJTU 会话；false = 未登录 */
const sessionReady = ref<boolean | null>(null)
const selectionOpen = ref<boolean | null>(null)

// ---- 课程列表 ----
const courses = ref<CourseResult[]>([])
const coursesLoading = ref(false)
const coursesError = ref('')
const filter = ref('')
const selected = ref<string[]>(restoreSelection())

watch(selected, (value) => {
  localStorage.setItem(selectionStorageKey(), JSON.stringify(value))
}, { deep: true })

type SortKey = 'name' | 'teacher' | 'remaining' | ''
type StatusFilter = 'all' | 'available' | 'selected' | 'unselected'

const statusFilter = ref<StatusFilter>('all')
const sortKey = ref<SortKey>('') // '' = 后端原始顺序
const sortDir = ref<'asc' | 'desc'>('asc')
const pageSize = ref(20)
const page = ref(1)
const PAGE_SIZES = [20, 50, 100]

// ---- 抢课 ----
const grabLoading = ref(false)
const grabError = ref('')

const {
  tasks,
  workerStatus,
  loginCount,
  lastLoginAt,
  loading: tasksLoading,
  error: tasksError,
  hasGrabbing,
  refresh: refreshTasks,
  pause,
  resume,
} = useCourseGrabTasks()

const STATUS_LABEL: Record<string, string> = {
  grabbing: '抢课中',
  paused: '已暂停',
  succeeded: '已抢到',
  not_found: '未找到',
  already_selected: '已选中',
  ambiguous: '标识不唯一',
  unconfirmed: '未确认',
}

const WORKER_STATUS_LABEL: Record<string, string> = {
  idle: '空闲',
  running: '运行中',
  auth_failed: '登录失效',
}

const MESSAGE_MAP: Record<string, string> = {
  'login required': '请先登录 Cairn',
  'BJTU login failed': '北交大账号或密码被拒绝，或登录需要额外验证，请检查后重试',
  'captcha recognition failed': '验证码识别失败，请检查图鉴账号是否可用',
  'saved course registration credentials not found': '尚未保存过教务凭据，请填写下方信息登录',
  'BJTU login session not found': '尚未登录教务系统',
  'BJTU academic system session expired': '教务会话已过期，请重新登录',
  'BJTU course selection is closed': '当前选课阶段未开放',
}

function friendlyError(error: unknown): string {
  if (error instanceof APIRequestError) {
    return MESSAGE_MAP[error.message] ?? error.message
  }
  return '网络异常，请稍后重试'
}

function isSessionMissing(error: unknown): boolean {
  return error instanceof APIRequestError && error.status === 401
}

function statusLabel(status: string): string {
  return STATUS_LABEL[status] ?? status
}

function statusClass(status: string): string {
  switch (status) {
    case 'succeeded':
    case 'already_selected':
      return 'ok'
    case 'grabbing':
      return 'warn'
    case 'paused':
      return 'dim'
    default:
      return 'err'
  }
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

async function loadCourses() {
  coursesLoading.value = true
  coursesError.value = ''
  try {
    const result = await listCourses()
    courses.value = result.courses
    sessionReady.value = true
    selectionOpen.value = result.selection_open
    // 列表非空时清理已不存在的勾选（如课程被教务移除），避免清空用户的正常选择
    if (result.courses.length > 0) {
      selected.value = selected.value.filter((name) =>
        result.courses.some((course) => course.name === name),
      )
    }
  } catch (e) {
    if (isSessionMissing(e)) {
      sessionReady.value = false
    } else {
      coursesError.value = friendlyError(e)
      loginError.value = `检查教务状态失败：${friendlyError(e)}`
      sessionReady.value = false
    }
    selectionOpen.value = null
  } finally {
    coursesLoading.value = false
  }
}

async function submitLogin() {
  loginLoading.value = true
  loginError.value = ''
  try {
    await loginCourseRegistration({
      tujian_username: tujianUsername.value,
      tujian_password: tujianPassword.value,
    })
    sessionReady.value = true
    tujianPassword.value = ''
    void loadCourses()
    void refreshTasks()
  } catch (e) {
    loginError.value = friendlyError(e)
  } finally {
    loginLoading.value = false
  }
}

function onMISBound() {
  sessionReady.value = true
  void loadCourses()
  void refreshTasks()
}

async function reloginWithSaved() {
  loginLoading.value = true
  loginError.value = ''
  try {
    await loginCourseRegistration({})
    sessionReady.value = true
    void loadCourses()
    void refreshTasks()
  } catch (e) {
    loginError.value = friendlyError(e)
  } finally {
    loginLoading.value = false
  }
}

const filteredCourses = computed(() => {
  const query = filter.value.trim().toLowerCase()
  let list = courses.value
  if (statusFilter.value === 'available') list = list.filter((course) => course.remaining > 0)
  else if (statusFilter.value === 'selected') list = list.filter((course) => course.selected)
  else if (statusFilter.value === 'unselected') list = list.filter((course) => !course.selected)
  if (!query) return list
  return list.filter((course) =>
    course.name.toLowerCase().includes(query) || course.teacher.toLowerCase().includes(query),
  )
})

const sortedCourses = computed(() => {
  const key = sortKey.value
  if (!key) return filteredCourses.value
  const dir = sortDir.value === 'asc' ? 1 : -1
  // 拷贝后再排序，避免污染「后端原始顺序」基线
  return [...filteredCourses.value].sort((a, b) => {
    if (key === 'remaining') return (a.remaining - b.remaining) * dir
    return a[key].localeCompare(b[key], 'zh-CN') * dir
  })
})

// 正在后台抢课中的课程名集合，用于列表里标记「在抢」
const grabbingNames = computed(() => {
  const names = new Set<string>()
  for (const task of tasks.value) {
    if (task.status === 'grabbing') names.add(task.course_name)
  }
  return names
})

// 已加入抢课任务（抢课中或已暂停）的课程名集合，这类课程不可再勾选
const taskedNames = computed(() => {
  const names = new Set<string>()
  for (const task of tasks.value) {
    if (task.status === 'grabbing' || task.status === 'paused') names.add(task.course_name)
  }
  return names
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredCourses.value.length / pageSize.value)))
const safePage = computed(() => Math.min(Math.max(1, page.value), totalPages.value))
const pagedCourses = computed(() => {
  const start = (safePage.value - 1) * pageSize.value
  return sortedCourses.value.slice(start, start + pageSize.value)
})

watch([filter, statusFilter, sortKey, sortDir, pageSize], () => {
  page.value = 1
})

function toggleCourse(course: CourseResult) {
  if (course.selected) return
  const index = selected.value.indexOf(course.name)
  if (index >= 0) {
    selected.value.splice(index, 1)
  } else if (selected.value.length < 50) {
    selected.value.push(course.name)
  }
}

// 点击表头循环切换：升序 → 降序 → 恢复后端原始顺序
function cycleSort(key: 'name' | 'teacher' | 'remaining') {
  if (sortKey.value !== key) {
    sortKey.value = key
    sortDir.value = 'asc'
  } else if (sortDir.value === 'asc') {
    sortDir.value = 'desc'
  } else {
    sortKey.value = ''
    sortDir.value = 'asc'
  }
}

async function submitGrab() {
  grabLoading.value = true
  grabError.value = ''
  try {
    await grabCourses({ courses: selected.value })
    // 已处理的课程全部移出已选列表：抢到的变「已选」，没余量的进抢课任务
    selected.value = []
    void loadCourses()
    void refreshTasks()
  } catch (e) {
    if (isSessionMissing(e)) {
      sessionReady.value = false
    }
    grabError.value = friendlyError(e)
  } finally {
    grabLoading.value = false
  }
}

onMounted(() => {
  void loadCourses()
  void refreshTasks()
})
</script>

<template>
  <div class="page">
    <div class="tool-page course-page">
      <RouterLink class="back" to="/tools">← 返回工具</RouterLink>
      <h1>教务抢课</h1>
      <p class="desc">登录北交大教务系统，挑课程一键抢课。暂无余量的课程自动进入后台任务，每 2 秒尝试一次。</p>

      <section class="panel" aria-labelledby="section-login">
        <h2 id="section-login">教务登录</h2>

        <MISBindForm :bound="sessionReady !== false" @bound="onMISBound" />

        <p v-if="sessionReady === null" class="panel-hint">正在检查教务会话…</p>

        <template v-else-if="sessionReady">
          <p class="session-line">
            <span class="dot ok" aria-hidden="true" />已建立教务会话
            <button type="button" class="btn ghost small" :disabled="loginLoading" @click="reloginWithSaved">
              {{ loginLoading ? '登录中…' : '重新登录教务' }}
            </button>
          </p>
        </template>

        <template v-else>
          <p class="panel-hint">需要先登录北交大统一身份认证系统，抢课接口才能使用。</p>
        </template>

        <p class="panel-hint">图鉴仅用于抢课验证码识别，使用已绑定的 MIS 账号登录。</p>
        <form class="login-grid" @submit.prevent="submitLogin">
          <label>
            图鉴用户名
            <input v-model.trim="tujianUsername" name="tujian-username" autocomplete="username" required />
          </label>
          <label>
            图鉴密码
            <input v-model="tujianPassword" type="password" name="tujian-password" autocomplete="current-password" required />
          </label>
          <div class="form-actions">
            <button type="submit" class="btn" :disabled="loginLoading">
              {{ loginLoading ? '登录中…' : '保存图鉴配置并登录' }}
            </button>
            <button type="button" class="btn ghost" :disabled="loginLoading" @click="reloginWithSaved">
              使用已保存的凭据
            </button>
          </div>
          <p v-if="loginError" class="panel-error" role="alert">{{ loginError }}</p>
        </form>
      </section>

      <section class="panel" aria-labelledby="section-courses">
        <h2 id="section-courses">课程列表</h2>

        <template v-if="sessionReady === true">
          <p v-if="selectionOpen === false" class="panel-hint">
            当前选课阶段未开放。教务登录正常，开放后刷新课程列表即可。
          </p>

          <div v-if="selectionOpen" class="courses-toolbar">
            <input v-model="filter" type="search" placeholder="按课程名称或教师筛选…" aria-label="筛选课程" />
            <select v-model="statusFilter" aria-label="按状态筛选">
              <option value="all">全部</option>
              <option value="available">有余量</option>
              <option value="unselected">未选</option>
              <option value="selected">已选</option>
            </select>
            <button type="button" class="btn ghost small" :disabled="coursesLoading" @click="loadCourses">
              {{ coursesLoading ? '加载中…' : '刷新列表' }}
            </button>
          </div>

          <p v-if="coursesError && selectionOpen !== false" class="panel-error" role="alert">{{ coursesError }}</p>

          <div v-if="selectionOpen && courses.length > 0" class="course-list">
            <div class="course-row course-head">
              <span class="course-check" aria-hidden="true" />
              <button type="button" class="sort-btn course-name" @click="cycleSort('name')">
                课程<span v-if="sortKey === 'name'" aria-hidden="true">{{ sortDir === 'asc' ? '↑' : '↓' }}</span>
              </button>
              <button type="button" class="sort-btn course-teacher" @click="cycleSort('teacher')">
                教师<span v-if="sortKey === 'teacher'" aria-hidden="true">{{ sortDir === 'asc' ? '↑' : '↓' }}</span>
              </button>
              <button type="button" class="sort-btn course-remaining" @click="cycleSort('remaining')">
                余量<span v-if="sortKey === 'remaining'" aria-hidden="true">{{ sortDir === 'asc' ? '↑' : '↓' }}</span>
              </button>
            </div>
            <label v-for="course in pagedCourses" :key="course.id ?? course.name" class="course-row">
              <span class="course-check">
                <span v-if="taskedNames.has(course.name)" class="badge" :class="grabbingNames.has(course.name) ? 'warn' : 'err'">
                  {{ grabbingNames.has(course.name) ? '在抢' : '已暂停' }}
                </span>
                <span v-else-if="course.selected" class="badge ok">已选</span>
                <input
                  v-else
                  type="checkbox"
                  :checked="selected.includes(course.name)"
                  @change="toggleCourse(course)"
                />
              </span>
              <span class="course-name">{{ course.name }}</span>
              <span class="course-teacher">{{ course.teacher }}</span>
              <span class="course-remaining" :class="{ none: course.remaining === 0 }">
                {{ course.remaining }}
              </span>
            </label>
            <p v-if="filteredCourses.length === 0" class="list-empty">没有匹配的课程。</p>
          </div>
          <p v-else-if="selectionOpen && !coursesLoading && !coursesError" class="panel-hint">课程列表为空。</p>

          <div v-if="selectionOpen && filteredCourses.length > 0" class="pager">
            <p class="pager-info">第 {{ safePage }} / {{ totalPages }} 页 · 共 {{ filteredCourses.length }} 门</p>
            <div class="pager-controls">
              <button type="button" class="btn ghost small" :disabled="safePage <= 1" @click="page = safePage - 1">上一页</button>
              <button type="button" class="btn ghost small" :disabled="safePage >= totalPages" @click="page = safePage + 1">下一页</button>
              <label class="pager-size">每页
                <select v-model.number="pageSize">
                  <option v-for="n in PAGE_SIZES" :key="n" :value="n">{{ n }}</option>
                </select>门</label>
            </div>
          </div>

          <div v-if="selectionOpen" class="grab-bar">
            <div class="grab-box">
              <p class="grab-count">已选 {{ selected.length }} 门</p>
              <ul v-if="selected.length > 0" class="grab-list">
                <li v-for="name in selected" :key="name">{{ name }}</li>
              </ul>
              <p v-else class="grab-empty">—</p>
            </div>
            <button
              type="button"
              class="btn"
              :disabled="grabLoading || selected.length === 0"
              @click="submitGrab"
            >
              {{ grabLoading ? '抢课中…' : `抢课（${selected.length}）` }}
            </button>
          </div>

          <p v-if="selectionOpen && grabError" class="panel-error" role="alert">{{ grabError }}</p>
        </template>

        <p v-else class="panel-hint">登录教务系统后可查看课程。</p>
      </section>

      <section class="panel" aria-labelledby="section-tasks">
        <h2 id="section-tasks">抢课任务</h2>

        <p class="worker-row">
          <span>Worker：<span class="badge" :class="workerStatus === 'running' ? 'warn' : workerStatus === 'auth_failed' ? 'err' : 'dim'">{{ WORKER_STATUS_LABEL[workerStatus] }}</span></span>
          <span>教务登录 {{ loginCount }} 次</span>
          <span v-if="lastLoginAt">最近登录 {{ formatTime(lastLoginAt) }}</span>
          <span v-if="hasGrabbing" class="poll-hint">任务运行中，每 5 秒自动刷新</span>
        </p>

        <p v-if="workerStatus === 'auth_failed'" class="panel-error" role="alert">
          Worker 自动登录失败，重新登录教务系统后才会继续抢课。
        </p>

        <p v-if="tasksError" class="panel-error" role="alert">
          {{ MESSAGE_MAP[tasksError] ?? tasksError }}
          <button type="button" class="btn ghost small" @click="refreshTasks">重试</button>
        </p>

        <ul v-if="tasks.length > 0" class="task-list">
          <li v-for="task in tasks" :key="task.id" class="task-row">
            <div class="task-main">
              <span class="task-name">{{ task.course_name }}</span>
              <span class="badge" :class="statusClass(task.status)">{{ statusLabel(task.status) }}</span>
            </div>
            <p class="task-meta">
              <span>{{ task.teacher }}</span>
              <span>余量 {{ task.remaining }}</span>
              <span>已尝试 {{ task.attempt_count }} 次</span>
              <span>最近检查 {{ formatTime(task.last_checked_at) }}</span>
            </p>
            <p v-if="task.message" class="task-message">{{ task.message }}</p>
            <div v-if="task.status === 'grabbing' || task.status === 'paused'" class="task-actions">
              <button v-if="task.status === 'grabbing'" type="button" class="btn ghost small" @click="pause(task.id)">
                暂停
              </button>
              <button v-else type="button" class="btn ghost small" @click="resume(task.id)">
                恢复
              </button>
            </div>
          </li>
        </ul>

        <p v-else-if="!tasksLoading && !tasksError" class="panel-hint">
          暂无抢课任务。没有余量的课程会在抢课时自动加入任务。
        </p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.course-page {
  max-width: calc(var(--measure) + 24rem);
}

/* ---------- panels ---------- */
.panel {
  border: 1px solid var(--rule);
  border-radius: 4px;
  padding: 1.4rem 1.6rem;
  margin-top: 2rem;
  background: color-mix(in srgb, var(--paper) 55%, transparent);
}
.panel h2 {
  font-family: var(--serif);
  font-weight: 500;
  font-size: 1.25rem;
  margin: 0 0 .8rem;
}
.panel-hint { color: var(--faded); font-size: .85rem; margin: 0; }
.panel-error { color: var(--blaze); font-size: .85rem; margin: .8rem 0 0; display: flex; align-items: center; gap: .8rem; flex-wrap: wrap; }

.session-line {
  display: flex; align-items: center; gap: .8rem;
  color: var(--faded); font-size: .9rem; margin: 0;
}
.dot {
  display: inline-block; width: .55rem; height: .55rem; border-radius: 50%;
  background: var(--faded);
}
.dot.ok { background: var(--pine); }

/* ---------- buttons ---------- */
.btn {
  border: 0; border-radius: 4px; padding: .6rem 1.4rem;
  background: var(--ink); color: var(--paper);
  font-family: var(--mono); font-size: .72rem; letter-spacing: .08em;
  text-transform: uppercase; cursor: pointer;
}
.btn:hover:not(:disabled) { background: var(--pine); }
.btn:disabled { opacity: .45; cursor: not-allowed; }
.btn.ghost {
  background: transparent; border: 1px solid var(--binding); color: var(--faded);
  padding: .55rem 1.3rem;
}
.btn.ghost:hover:not(:disabled) { border-color: var(--pine); color: var(--pine); }
.btn.small { padding: .3rem .7rem; font-size: .66rem; }

/* ---------- badges ---------- */
.badge {
  display: inline-block;
  font-family: var(--mono); font-size: .66rem; letter-spacing: .05em;
  padding: .12rem .5rem; border-radius: 2px;
  border: 1px solid var(--binding); color: var(--faded);
  white-space: nowrap;
}
.badge.ok { color: var(--pine); border-color: var(--pine); }
.badge.warn { background: var(--blaze); border-color: var(--blaze); color: var(--paper); }
.badge.err { color: var(--blaze); border-color: var(--blaze); background: color-mix(in srgb, var(--blaze) 10%, transparent); }

/* ---------- login form ---------- */
.login-grid { display: grid; grid-template-columns: 1fr 1fr; gap: .9rem 1.2rem; margin-top: 1.1rem; }
.login-grid label {
  display: flex; flex-direction: column; gap: .35rem;
  font-size: .78rem; color: var(--faded);
}
.login-grid input {
  border: 1px solid var(--binding); border-radius: 3px;
  background: var(--paper); color: var(--ink);
  padding: .5rem .7rem; font: inherit; font-size: .9rem;
}
.login-grid input:focus { outline: 2px solid var(--pine); border-color: var(--pine); }
.form-actions { grid-column: 1 / -1; display: flex; gap: .8rem; align-items: center; flex-wrap: wrap; }
.form-actions .panel-error { margin: 0; }

/* ---------- courses ---------- */
.courses-toolbar { display: flex; gap: .8rem; margin-bottom: 1rem; }
.courses-toolbar input,
.courses-toolbar select {
  flex: 1; min-width: 0;
  border: 1px solid var(--binding); border-radius: 3px;
  background: var(--paper); color: var(--ink);
  padding: .5rem .7rem; font: inherit; font-size: .9rem;
}
.courses-toolbar select { flex: 0 0 auto; cursor: pointer; }
.courses-toolbar input:focus,
.courses-toolbar select:focus { outline: 2px solid var(--pine); border-color: var(--pine); }

.course-list {
  border: 1px solid var(--binding); border-radius: 4px;
  margin: 0 0 1.2rem; max-height: 24rem; overflow-y: auto;
  background: var(--paper);
}
.course-row {
  display: flex; align-items: center; gap: .8rem;
  padding: .55rem .9rem; font-size: .85rem; cursor: pointer;
}
.course-row + .course-row { border-top: 1px solid var(--rule); }
.course-row:hover { background: color-mix(in srgb, var(--paper-2) 55%, transparent); }
.course-head {
  cursor: default; position: sticky; top: 0; z-index: 1;
  background: var(--paper-2);
  font-family: var(--mono); font-size: .66rem; letter-spacing: .08em;
  text-transform: uppercase; color: var(--faded);
}
.course-head:hover { background: var(--paper-2); }
.sort-btn {
  font: inherit; color: inherit; background: none; border: 0; padding: 0;
  letter-spacing: inherit; text-transform: inherit; cursor: pointer;
  text-align: left; display: flex; align-items: center; gap: .25rem;
}
.course-remaining.sort-btn { text-align: right; justify-content: flex-end; }
.sort-btn:hover { color: var(--pine); }
.sort-btn:focus-visible { outline: 2px solid var(--pine); }
.course-check { min-width: 3em; flex-shrink: 0; display: flex; }
.course-check input { accent-color: var(--pine); cursor: pointer; }
.course-check .badge { padding: .12rem .4rem; }
.course-name { flex: 1; min-width: 0; overflow-wrap: anywhere; }
.course-teacher { width: 7em; flex-shrink: 0; color: var(--faded); }
.course-remaining {
  width: 3.5em; flex-shrink: 0; text-align: right;
  font-family: var(--mono); font-size: .8rem;
}
.course-remaining.none { color: var(--blaze); }
.list-empty { color: var(--faded); font-size: .85rem; padding: .8rem .9rem; }

.grab-bar {
  display: flex; justify-content: space-between; align-items: flex-start;
  gap: 1rem;
}
.grab-bar .btn { flex-shrink: 0; }
.grab-box {
  flex: 1; min-width: 0;
  border: 1px solid var(--binding); border-radius: 4px;
  background: var(--paper);
  padding: .7rem .9rem;
  max-height: 11rem; overflow-y: auto;
}
.grab-count { margin: 0; color: var(--faded); font-size: .85rem; }
.grab-list { list-style: none; margin: .5rem 0 0; padding: 0; }
.grab-list li {
  font-size: .85rem; overflow-wrap: anywhere;
  padding: .15rem 0;
}
.grab-list li + li { border-top: 1px dashed var(--rule); }
.grab-empty { margin: .5rem 0 0; color: var(--faded); font-size: .85rem; }

.pager {
  display: flex; justify-content: space-between; align-items: center;
  gap: 1rem; flex-wrap: wrap; margin: 0 0 1.2rem;
}
.pager-info { margin: 0; color: var(--faded); font-size: .85rem; }
.pager-controls { display: flex; align-items: center; gap: .8rem; flex-wrap: wrap; }
.pager-size { color: var(--faded); font-size: .85rem; display: flex; align-items: center; gap: .4rem; }
.pager-size select {
  border: 1px solid var(--binding); border-radius: 3px;
  background: var(--paper); color: var(--ink);
  padding: .2rem .4rem; font: inherit; font-size: .85rem; cursor: pointer;
}

/* ---------- tasks ---------- */
.worker-row {
  display: flex; gap: 1.4rem; align-items: center; flex-wrap: wrap;
  margin: 0; color: var(--faded); font-size: .85rem;
}
.poll-hint { font-family: var(--mono); font-size: .68rem; letter-spacing: .05em; color: var(--blaze); }

.task-list { list-style: none; margin: 1.1rem 0 0; padding: 0; }
.task-row {
  border: 1px solid var(--binding); border-radius: 4px;
  padding: .9rem 1rem; margin-bottom: .8rem; background: var(--paper);
}
.task-main { display: flex; gap: .8rem; align-items: center; flex-wrap: wrap; margin-bottom: .5rem; }
.task-name { font-size: .9rem; overflow-wrap: anywhere; }
.task-meta {
  display: flex; gap: 1.2rem; flex-wrap: wrap; margin: 0;
  font-family: var(--mono); font-size: .68rem; color: var(--faded);
}
.task-message { margin: .5rem 0 0; font-size: .85rem; color: var(--faded); }
.task-actions { margin-top: .7rem; display: flex; gap: .6rem; }

/* ---------- responsive ---------- */
@media (max-width: 720px) {
  .login-grid { grid-template-columns: 1fr; }
  .course-teacher { width: auto; }
  .grab-bar { flex-direction: column; align-items: stretch; }
  .grab-bar .btn { align-self: flex-end; }
}
</style>
