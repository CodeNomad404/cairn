<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { APIRequestError } from '../../../shared/api/http'
import { grabCourses, listCourses, loginCourseRegistration } from '../api/tools'
import { useCourseGrabTasks } from '../composables/useCourseGrabTasks'
import type { CourseGrabItemResult, CourseResult } from '../types'

// ---- 教务登录 ----
const studentId = ref('')
const password = ref('')
const tujianUsername = ref('')
const tujianPassword = ref('')
const loginLoading = ref(false)
const loginError = ref('')
/** null = 正在探测会话；true = 已建立 BJTU 会话；false = 未登录 */
const sessionReady = ref<boolean | null>(null)
const showLoginForm = ref(false)

// ---- 课程列表 ----
const courses = ref<CourseResult[]>([])
const coursesLoading = ref(false)
const coursesError = ref('')
const filter = ref('')
const selected = ref<string[]>([])

// ---- 抢课 ----
const grabLoading = ref(false)
const grabError = ref('')
const grabResults = ref<CourseGrabItemResult[]>([])

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
  'BJTU login failed': '北交大账号、密码或验证码被拒绝，请检查后重试',
  'captcha recognition failed': '验证码识别失败，请检查图鉴账号是否可用',
  'saved course registration credentials not found': '尚未保存过教务凭据，请填写下方信息登录',
  'BJTU login session not found': '尚未登录教务系统',
  'BJTU academic system session expired': '教务会话已过期，请重新登录',
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
  } catch (e) {
    if (isSessionMissing(e)) {
      sessionReady.value = false
    } else {
      coursesError.value = friendlyError(e)
    }
  } finally {
    coursesLoading.value = false
  }
}

async function submitLogin() {
  loginLoading.value = true
  loginError.value = ''
  try {
    await loginCourseRegistration({
      student_id: studentId.value,
      password: password.value,
      tujian_username: tujianUsername.value,
      tujian_password: tujianPassword.value,
    })
    sessionReady.value = true
    showLoginForm.value = false
    password.value = ''
    tujianPassword.value = ''
    void loadCourses()
    void refreshTasks()
  } catch (e) {
    loginError.value = friendlyError(e)
  } finally {
    loginLoading.value = false
  }
}

async function reloginWithSaved() {
  loginLoading.value = true
  loginError.value = ''
  try {
    await loginCourseRegistration({})
    sessionReady.value = true
    showLoginForm.value = false
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
  if (!query) return courses.value
  return courses.value.filter((course) =>
    course.name.toLowerCase().includes(query) || course.teacher.toLowerCase().includes(query),
  )
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

async function submitGrab() {
  grabLoading.value = true
  grabError.value = ''
  try {
    const result = await grabCourses({ courses: selected.value })
    grabResults.value = result.results
    // 已抢到或已选中的课程不再留在待抢列表
    selected.value = selected.value.filter((name) => {
      const item = result.results.find((r) => r.target === name)
      return item === undefined || (item.status !== 'succeeded' && item.status !== 'already_selected')
    })
    void loadCourses()
    void refreshTasks()
  } catch (e) {
    if (isSessionMissing(e)) {
      sessionReady.value = false
      showLoginForm.value = true
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

        <p v-if="sessionReady === null" class="panel-hint">正在检查教务会话…</p>

        <template v-else-if="sessionReady">
          <p class="session-line">
            <span class="dot ok" aria-hidden="true" />已建立教务会话
            <button type="button" class="btn ghost small" @click="showLoginForm = !showLoginForm">
              {{ showLoginForm ? '收起' : '重新登录' }}
            </button>
          </p>
        </template>

        <template v-else>
          <p class="panel-hint">需要先登录北交大统一身份认证系统，抢课接口才能使用。</p>
        </template>

        <form v-if="showLoginForm || sessionReady === false" class="login-grid" @submit.prevent="submitLogin">
          <label>
            学号
            <input v-model.trim="studentId" name="student-id" autocomplete="username" required />
          </label>
          <label>
            教务密码
            <input v-model="password" type="password" name="bjtu-password" autocomplete="current-password" required />
          </label>
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
              {{ loginLoading ? '登录中…' : '登录' }}
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
          <div class="courses-toolbar">
            <input v-model="filter" type="search" placeholder="按课程名称或教师筛选…" aria-label="筛选课程" />
            <button type="button" class="btn ghost small" :disabled="coursesLoading" @click="loadCourses">
              {{ coursesLoading ? '加载中…' : '刷新列表' }}
            </button>
          </div>

          <p v-if="coursesError" class="panel-error" role="alert">{{ coursesError }}</p>

          <div v-if="courses.length > 0" class="course-list">
            <div class="course-row course-head" aria-hidden="true">
              <span class="course-check" />
              <span class="course-name">课程</span>
              <span class="course-teacher">教师</span>
              <span class="course-remaining">余量</span>
            </div>
            <label v-for="course in filteredCourses" :key="course.id ?? course.name" class="course-row">
              <span class="course-check">
                <input
                  v-if="!course.selected"
                  type="checkbox"
                  :checked="selected.includes(course.name)"
                  @change="toggleCourse(course)"
                />
                <span v-else class="badge ok">已选</span>
              </span>
              <span class="course-name">{{ course.name }}</span>
              <span class="course-teacher">{{ course.teacher }}</span>
              <span class="course-remaining" :class="{ none: course.remaining === 0 }">
                {{ course.remaining }}
              </span>
            </label>
            <p v-if="filteredCourses.length === 0" class="list-empty">没有匹配的课程。</p>
          </div>
          <p v-else-if="!coursesLoading && !coursesError" class="panel-hint">课程列表为空。</p>

          <div class="grab-bar">
            <p class="grab-selection">已选 {{ selected.length }} 门：{{ selected.join('、') || '—' }}</p>
            <button
              type="button"
              class="btn"
              :disabled="grabLoading || selected.length === 0"
              @click="submitGrab"
            >
              {{ grabLoading ? '抢课中…' : `抢课（${selected.length}）` }}
            </button>
          </div>

          <p v-if="grabError" class="panel-error" role="alert">{{ grabError }}</p>

          <ul v-if="grabResults.length > 0" class="grab-results" aria-label="抢课结果">
            <li v-for="item in grabResults" :key="item.target">
              <span class="badge" :class="statusClass(item.status)">{{ statusLabel(item.status) }}</span>
              <span class="grab-target">{{ item.target }}</span>
              <span class="grab-message">{{ item.message }}</span>
            </li>
          </ul>
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
.courses-toolbar input {
  flex: 1; min-width: 0;
  border: 1px solid var(--binding); border-radius: 3px;
  background: var(--paper); color: var(--ink);
  padding: .5rem .7rem; font: inherit; font-size: .9rem;
}
.courses-toolbar input:focus { outline: 2px solid var(--pine); border-color: var(--pine); }

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
.course-check { width: 3em; flex-shrink: 0; display: flex; }
.course-check input { accent-color: var(--pine); cursor: pointer; }
.course-name { flex: 1; min-width: 0; overflow-wrap: anywhere; }
.course-teacher { width: 7em; flex-shrink: 0; color: var(--faded); }
.course-remaining {
  width: 3.5em; flex-shrink: 0; text-align: right;
  font-family: var(--mono); font-size: .8rem;
}
.course-remaining.none { color: var(--blaze); }
.list-empty { color: var(--faded); font-size: .85rem; padding: .8rem .9rem; }

.grab-bar {
  display: flex; justify-content: space-between; align-items: center;
  gap: 1rem; flex-wrap: wrap;
}
.grab-selection { margin: 0; color: var(--faded); font-size: .85rem; min-width: 0; overflow-wrap: anywhere; }

.grab-results { list-style: none; margin: 1.2rem 0 0; padding: 0; }
.grab-results li {
  display: flex; gap: .8rem; align-items: baseline; flex-wrap: wrap;
  padding: .5rem 0; border-top: 1px dashed var(--rule);
  font-size: .85rem;
}
.grab-target { font-family: var(--mono); font-size: .8rem; }
.grab-message { color: var(--faded); }

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
}
</style>
