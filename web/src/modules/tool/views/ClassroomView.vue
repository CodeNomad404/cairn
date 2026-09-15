<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listClassrooms } from '../api/tools'
import type {
  ClassroomListResult,
  ClassroomOccupationType,
  ClassroomRoom,
} from '../types'

const occupationOptions: Array<{ value: '' | ClassroomOccupationType; label: string }> = [
  { value: '', label: '全部状态' },
  { value: 'free', label: '空闲' },
  { value: 'scheduled', label: '排课占用' },
  { value: 'rescheduled', label: '调课占用' },
  { value: 'exam', label: '考试占用' },
  { value: 'laboratory', label: '实验占用' },
  { value: 'other', label: '其他安排' },
]

const occupationLabels: Record<ClassroomOccupationType, string> = {
  free: '空闲',
  scheduled: '排课占用',
  rescheduled: '调课占用',
  exam: '考试占用',
  laboratory: '实验占用',
  other: '其他安排',
}

const result = ref<ClassroomListResult | null>(null)
const loading = ref(false)
const error = ref('')
const selectedBuilding = ref('')
const selectedPeriod = ref(1)
const selectedOccupation = ref<'' | ClassroomOccupationType>('free')

const buildings = computed(() => {
  const names = new Set(result.value?.rooms.map((room) => room.building_name) ?? [])
  return [...names].sort((left, right) => left.localeCompare(right, 'zh-CN'))
})

const filteredRooms = computed<ClassroomRoom[]>(() => {
  const rooms = result.value?.rooms ?? []
  return rooms.filter((room) => {
    if (selectedBuilding.value && room.building_name !== selectedBuilding.value) return false
    if (room.period !== selectedPeriod.value) return false
    if (selectedOccupation.value && room.occupation_type !== selectedOccupation.value) return false
    return true
  })
})

const syncedAt = computed(() => {
  if (!result.value?.synced_at) return '暂无同步记录'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(result.value.synced_at))
})

async function loadClassrooms() {
  loading.value = true
  error.value = ''
  try {
    result.value = await listClassrooms()
  } catch (reason: unknown) {
    error.value = reason instanceof Error ? reason.message : '教室列表加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadClassrooms()
})
</script>

<template>
  <div class="page">
    <div class="tool-page classroom-page">
      <RouterLink class="back" to="/tools">← 返回工具</RouterLink>

      <header class="classroom-heading">
        <div>
          <span class="classroom-eyebrow">今日教室</span>
          <h1>找一间空教室</h1>
          <p class="desc">数据来自教务系统，每天同步一次；首次查询可能需要稍等片刻。</p>
        </div>
        <button class="classroom-refresh" type="button" :disabled="loading" @click="loadClassrooms">
          {{ loading ? '载入中…' : '刷新列表' }}
        </button>
      </header>

      <div class="classroom-meta" aria-live="polite">
        <span>{{ result?.date ?? '今天' }}</span>
        <span>更新于 {{ syncedAt }}</span>
        <strong>{{ filteredRooms.length }} 间</strong>
      </div>

      <section class="classroom-filters" aria-label="教室筛选">
        <label>
          <span>教学楼</span>
          <select v-model="selectedBuilding">
            <option value="">全部教学楼</option>
            <option v-for="building in buildings" :key="building" :value="building">
              {{ building }}
            </option>
          </select>
        </label>

        <label>
          <span>节次</span>
          <select v-model.number="selectedPeriod">
            <option v-for="period in 7" :key="period" :value="period">
              第 {{ period }} 节
            </option>
          </select>
        </label>

        <label>
          <span>占用状态</span>
          <select v-model="selectedOccupation">
            <option v-for="option in occupationOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>
      </section>

      <p v-if="error" class="classroom-message classroom-error" role="alert">{{ error }}</p>
      <p v-else-if="loading && !result" class="classroom-message">正在读取今天的教室安排…</p>
      <p v-else-if="!result?.synced_at" class="classroom-message">今天还没有可用的教室快照。</p>
      <p v-else-if="filteredRooms.length === 0" class="classroom-message">没有符合当前条件的教室。</p>

      <div v-else class="classroom-table-wrap">
        <table class="classroom-table">
          <thead>
            <tr>
              <th>教学楼</th>
              <th>教室</th>
              <th>节次</th>
              <th>容量</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="room in filteredRooms"
              :key="`${room.building_id}-${room.room_name}-${room.period}`"
            >
              <td>{{ room.building_name }}</td>
              <td class="classroom-room-name">{{ room.room_name }}</td>
              <td>第 {{ room.period }} 节</td>
              <td>{{ room.capacity > 0 ? room.capacity : '未知' }}</td>
              <td>
                <span class="classroom-status" :class="`is-${room.occupation_type}`">
                  {{ occupationLabels[room.occupation_type] }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.classroom-page { max-width: none; }
.classroom-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 2rem;
  margin-bottom: 1.5rem;
}
.classroom-heading .desc { margin-bottom: 0; }
.classroom-eyebrow {
  display: block;
  margin-bottom: .3rem;
  color: var(--blaze);
  font-family: var(--mono);
  font-size: .68rem;
  letter-spacing: .14em;
  text-transform: uppercase;
}
.classroom-refresh {
  flex: none;
  border: 1px solid var(--ink);
  border-radius: 4px;
  padding: .65rem 1rem;
  background: var(--ink);
  color: var(--paper);
  font-family: var(--mono);
  font-size: .7rem;
  letter-spacing: .06em;
  cursor: pointer;
}
.classroom-refresh:hover:not(:disabled) { background: var(--pine); border-color: var(--pine); }
.classroom-refresh:disabled { cursor: wait; opacity: .6; }
.classroom-meta {
  display: flex;
  flex-wrap: wrap;
  gap: .7rem 1.2rem;
  align-items: baseline;
  padding: .8rem 1rem;
  border-block: 1px solid var(--rule);
  color: var(--faded);
  font-family: var(--mono);
  font-size: .72rem;
}
.classroom-meta strong { margin-left: auto; color: var(--ink); font-size: .85rem; }
.classroom-filters {
  display: grid;
  grid-template-columns: minmax(12rem, 1.5fr) repeat(2, minmax(9rem, 1fr));
  gap: 1rem;
  margin: 1.4rem 0;
}
.classroom-filters label { display: grid; gap: .35rem; }
.classroom-filters label span {
  color: var(--faded);
  font-family: var(--mono);
  font-size: .68rem;
  letter-spacing: .08em;
}
.classroom-filters select {
  width: 100%;
  border: 1px solid var(--binding);
  border-radius: 4px;
  padding: .65rem .75rem;
  background: var(--paper);
  color: var(--ink);
  font: inherit;
}
.classroom-filters select:focus { outline: 2px solid var(--pine); outline-offset: 1px; }
.classroom-message {
  padding: 2rem 1rem;
  border: 1px dashed var(--binding);
  color: var(--faded);
  text-align: center;
}
.classroom-error { color: var(--blaze); }
.classroom-table-wrap { overflow-x: auto; border: 1px solid var(--rule); border-radius: 4px; }
.classroom-table { width: 100%; border-collapse: collapse; background: color-mix(in srgb, var(--paper) 88%, white); }
.classroom-table th,
.classroom-table td { padding: .72rem .85rem; border-bottom: 1px solid var(--rule); text-align: left; white-space: nowrap; }
.classroom-table th {
  color: var(--faded);
  font-family: var(--mono);
  font-size: .66rem;
  font-weight: 500;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.classroom-table td { font-size: .85rem; }
.classroom-table tbody tr:last-child td { border-bottom: 0; }
.classroom-table tbody tr:hover { background: color-mix(in srgb, var(--paper-2) 65%, transparent); }
.classroom-room-name { color: var(--pine); font-family: var(--mono); font-weight: 500; }
.classroom-status {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: .12rem .55rem;
  background: var(--paper-2);
  color: var(--faded);
  font-size: .72rem;
}
.classroom-status.is-free { background: color-mix(in srgb, var(--pine) 13%, transparent); color: var(--pine); }
.classroom-status.is-scheduled { background: color-mix(in srgb, var(--blaze) 14%, transparent); color: var(--blaze); }
.classroom-status.is-exam { background: rgb(57 78 214 / 12%); color: #394ed6; }
.classroom-status.is-laboratory { background: rgb(65 133 55 / 13%); color: #34712c; }
.classroom-status.is-rescheduled,
.classroom-status.is-other { background: rgb(133 122 102 / 14%); color: var(--faded); }

@media (max-width: 720px) {
  .classroom-heading { align-items: stretch; flex-direction: column; gap: 1rem; }
  .classroom-refresh { align-self: flex-start; }
  .classroom-filters { grid-template-columns: 1fr; }
  .classroom-meta strong { width: 100%; margin-left: 0; }
}
</style>
