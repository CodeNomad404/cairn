import { computed, onUnmounted, ref } from 'vue'
import { APIRequestError } from '../../../shared/api/http'
import { listCourseGrabTasks, setCourseGrabTaskStatus } from '../api/tools'
import type { CourseGrabTask, CourseGrabTaskListResult } from '../types'

const POLL_INTERVAL_MS = 5000

/**
 * 抢课任务状态：拉取任务列表，存在抢课中的任务时每 5 秒自动刷新。
 * 后端 Worker 循环间隔为 2 秒，前端轮询不需要更频繁。
 */
export function useCourseGrabTasks() {
  const tasks = ref<CourseGrabTask[]>([])
  const workerStatus = ref<CourseGrabTaskListResult['worker_status']>('idle')
  const loginCount = ref(0)
  const lastLoginAt = ref<string>()
  const loading = ref(false)
  const error = ref('')

  let timer: ReturnType<typeof setInterval> | null = null

  const hasGrabbing = computed(() => tasks.value.some((task) => task.status === 'grabbing'))

  function syncPolling() {
    if (hasGrabbing.value && timer === null) {
      timer = setInterval(() => void refresh(), POLL_INTERVAL_MS)
    } else if (!hasGrabbing.value && timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  async function refresh() {
    if (loading.value) return
    loading.value = true
    error.value = ''
    try {
      const result = await listCourseGrabTasks()
      tasks.value = result.items
      workerStatus.value = result.worker_status
      loginCount.value = result.login_count
      lastLoginAt.value = result.last_login_at
      syncPolling()
    } catch (e) {
      error.value = e instanceof APIRequestError ? e.message : '加载抢课任务失败'
    } finally {
      loading.value = false
    }
  }

  async function changeStatus(taskID: number, status: 'grabbing' | 'paused') {
    try {
      await setCourseGrabTaskStatus(taskID, { status })
      await refresh()
    } catch (e) {
      error.value = e instanceof APIRequestError ? e.message : '操作失败，请重试'
    }
  }

  const pause = (taskID: number) => changeStatus(taskID, 'paused')
  const resume = (taskID: number) => changeStatus(taskID, 'grabbing')

  onUnmounted(() => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  })

  return {
    tasks,
    workerStatus,
    loginCount,
    lastLoginAt,
    loading,
    error,
    hasGrabbing,
    refresh,
    pause,
    resume,
  }
}
