<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
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
const selectedBuildings = ref<string[]>([])
const startPeriod = ref(1)
const endPeriod = ref(2)
const selectedOccupation = ref<'' | ClassroomOccupationType>('free')
const currentPage = ref(1)
const buildingPicker = ref<HTMLDetailsElement | null>(null)
const occupationPicker = ref<HTMLDetailsElement | null>(null)
const pageSize = 20

type ClassroomRangeRoom = Pick<
  ClassroomRoom,
  'building_id' | 'building_name' | 'room_name' | 'capacity'
> & {
  periods: ClassroomRoom[]
}

const buildings = computed(() => {
  const names = new Set(result.value?.rooms.map((room) => room.building_name) ?? [])
  return [...names].sort((left, right) => left.localeCompare(right, 'zh-CN'))
})

const selectedBuildingSet = computed(() => new Set(selectedBuildings.value))

const buildingSelectionLabel = computed(() => {
  const count = selectedBuildings.value.length
  if (count === 0 || count === buildings.value.length) return '全部教学楼'
  if (count === 1) return selectedBuildings.value[0]
  return `已选 ${count} 座教学楼`
})

const occupationSelectionLabel = computed(
  () => occupationOptions.find((option) => option.value === selectedOccupation.value)?.label,
)

const selectedPeriods = computed(() =>
  Array.from(
    { length: endPeriod.value - startPeriod.value + 1 },
    (_, index) => startPeriod.value + index,
  ),
)

const periodRangeLabel = computed(() => {
  if (startPeriod.value === endPeriod.value) return `第 ${startPeriod.value} 节`
  return `第 ${startPeriod.value}–${endPeriod.value} 节`
})

const timelineSelectionStyle = computed(() => ({
  left: `${((startPeriod.value - 1) / 6) * 100}%`,
  width: `${((endPeriod.value - startPeriod.value) / 6) * 100}%`,
}))

const filteredRooms = computed<ClassroomRangeRoom[]>(() => {
  const groupedRooms = new Map<string, Map<number, ClassroomRoom>>()
  for (const room of result.value?.rooms ?? []) {
    if (selectedBuildingSet.value.size > 0 && !selectedBuildingSet.value.has(room.building_name)) {
      continue
    }
    const key = `${room.building_id}\u0000${room.room_name}`
    const periods = groupedRooms.get(key) ?? new Map<number, ClassroomRoom>()
    periods.set(room.period, room)
    groupedRooms.set(key, periods)
  }

  const rooms: ClassroomRangeRoom[] = []
  for (const periods of groupedRooms.values()) {
    const selected = selectedPeriods.value.map((period) => periods.get(period))
    if (selected.some((room) => room === undefined)) continue

    const completeRange = selected as ClassroomRoom[]
    if (
      selectedOccupation.value &&
      completeRange.some((room) => room.occupation_type !== selectedOccupation.value)
    ) {
      continue
    }

    const first = completeRange[0]
    rooms.push({
      building_id: first.building_id,
      building_name: first.building_name,
      room_name: first.room_name,
      capacity: first.capacity,
      periods: completeRange,
    })
  }
  return rooms
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredRooms.value.length / pageSize)))

const paginatedRooms = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredRooms.value.slice(start, start + pageSize)
})

const pageRangeLabel = computed(() => {
  const start = (currentPage.value - 1) * pageSize + 1
  const end = Math.min(currentPage.value * pageSize, filteredRooms.value.length)
  return `第 ${start}–${end} 间，共 ${filteredRooms.value.length} 间`
})

watch(
  [startPeriod, endPeriod, selectedOccupation, () => selectedBuildings.value.join('\u0000')],
  () => {
    currentPage.value = 1
  },
)

watch(totalPages, (pages) => {
  currentPage.value = Math.min(currentPage.value, pages)
})

function updateStartPeriod(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  startPeriod.value = Math.min(value, endPeriod.value)
}

function updateEndPeriod(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  endPeriod.value = Math.max(value, startPeriod.value)
}

function selectAllBuildings() {
  selectedBuildings.value = [...buildings.value]
}

function clearBuildingSelection() {
  selectedBuildings.value = []
}

function selectOccupation(value: '' | ClassroomOccupationType) {
  selectedOccupation.value = value
  occupationPicker.value?.removeAttribute('open')
}

function handlePickerToggle(event: Event, otherPicker: HTMLDetailsElement | null) {
  const picker = event.currentTarget as HTMLDetailsElement
  if (picker.open) otherPicker?.removeAttribute('open')
}

function closePickers(event: PointerEvent) {
  const target = event.target as Node
  if (!buildingPicker.value?.contains(target)) buildingPicker.value?.removeAttribute('open')
  if (!occupationPicker.value?.contains(target)) occupationPicker.value?.removeAttribute('open')
}

function rangeStatusLabel(room: ClassroomRangeRoom) {
  const statuses = new Set(room.periods.map((period) => period.occupation_type))
  if (statuses.size > 1) return '状态混合'
  const status = room.periods[0].occupation_type
  return status === 'free' ? '全程空闲' : `全程${occupationLabels[status]}`
}

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
  document.addEventListener('pointerdown', closePickers)
  void loadClassrooms()
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', closePickers)
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
        <strong>{{ periodRangeLabel }} · {{ filteredRooms.length }} 间</strong>
      </div>

      <section class="classroom-filters" aria-label="教室筛选">
        <div class="building-filter">
          <span>教学楼</span>
          <details
            ref="buildingPicker"
            class="filter-picker building-picker"
            @toggle="handlePickerToggle($event, occupationPicker)"
          >
            <summary>{{ buildingSelectionLabel }}</summary>
            <div class="filter-picker-panel building-picker-panel">
              <div class="building-picker-actions">
                <button type="button" @click="selectAllBuildings">全选</button>
                <button type="button" @click="clearBuildingSelection">不限</button>
              </div>
              <div class="building-options">
                <label v-for="building in buildings" :key="building">
                  <input v-model="selectedBuildings" type="checkbox" :value="building">
                  <span class="building-check" aria-hidden="true" />
                  <span class="building-name">{{ building }}</span>
                </label>
              </div>
            </div>
          </details>
        </div>

        <div class="occupation-filter">
          <span>占用状态</span>
          <details
            ref="occupationPicker"
            class="filter-picker occupation-picker"
            @toggle="handlePickerToggle($event, buildingPicker)"
          >
            <summary>
              <span
                v-if="selectedOccupation"
                class="occupation-dot"
                :class="`is-${selectedOccupation}`"
              />
              {{ occupationSelectionLabel }}
            </summary>
            <div class="filter-picker-panel occupation-options" role="radiogroup">
              <button
                v-for="option in occupationOptions"
                :key="option.value"
                type="button"
                role="radio"
                :aria-checked="selectedOccupation === option.value"
                :class="{ 'is-selected': selectedOccupation === option.value }"
                @click="selectOccupation(option.value)"
              >
                <span
                  class="occupation-dot"
                  :class="option.value ? `is-${option.value}` : 'is-all'"
                />
                <span>{{ option.label }}</span>
                <span class="occupation-check" aria-hidden="true">✓</span>
              </button>
            </div>
          </details>
        </div>
      </section>

      <section class="period-editor" aria-labelledby="period-editor-title">
        <div class="period-editor-heading">
          <div>
            <span id="period-editor-title">连续节次</span>
            <strong>{{ periodRangeLabel }}</strong>
          </div>
          <p>拖动左右节点，查找整个时间段内都符合筛选条件的教室</p>
        </div>

        <div class="period-timeline">
          <div class="period-track" aria-hidden="true">
            <span
              v-for="period in 7"
              :key="period"
              class="period-block"
              :class="{ 'is-selected': period >= startPeriod && period <= endPeriod }"
            />
            <span class="period-selection" :style="timelineSelectionStyle" />
          </div>
          <input
            class="period-range period-range-start"
            type="range"
            min="1"
            max="7"
            step="1"
            :value="startPeriod"
            :aria-label="`开始节次，第 ${startPeriod} 节`"
            :style="{ zIndex: startPeriod === endPeriod ? 5 : 3 }"
            @input="updateStartPeriod"
          >
          <input
            class="period-range period-range-end"
            type="range"
            min="1"
            max="7"
            step="1"
            :value="endPeriod"
            :aria-label="`结束节次，第 ${endPeriod} 节`"
            @input="updateEndPeriod"
          >
        </div>

        <div class="period-labels" aria-hidden="true">
          <span v-for="period in 7" :key="period">{{ period }}</span>
        </div>
      </section>

      <p v-if="error" class="classroom-message classroom-error" role="alert">{{ error }}</p>
      <p v-else-if="loading && !result" class="classroom-message">正在读取今天的教室安排…</p>
      <p v-else-if="!result?.synced_at" class="classroom-message">今天还没有可用的教室快照。</p>
      <p v-else-if="filteredRooms.length === 0" class="classroom-message">没有符合当前条件的教室。</p>

      <div v-else class="classroom-results">
        <div class="classroom-table-wrap">
          <table class="classroom-table">
            <thead>
              <tr>
                <th>教学楼</th>
                <th>教室</th>
                <th>连续节次</th>
                <th>容量</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="room in paginatedRooms"
                :key="`${room.building_id}-${room.room_name}`"
              >
                <td>{{ room.building_name }}</td>
                <td class="classroom-room-name">{{ room.room_name }}</td>
                <td>{{ periodRangeLabel }}</td>
                <td>{{ room.capacity > 0 ? room.capacity : '未知' }}</td>
                <td>
                  <div class="room-period-statuses">
                    <span
                      v-for="period in room.periods"
                      :key="period.period"
                      class="room-period-status"
                      :class="`is-${period.occupation_type}`"
                      :title="`第 ${period.period} 节：${occupationLabels[period.occupation_type]}`"
                    />
                    <span>{{ rangeStatusLabel(room) }}</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <nav class="classroom-pagination" aria-label="教室列表分页">
          <span>{{ pageRangeLabel }}</span>
          <div>
            <button
              type="button"
              :disabled="currentPage === 1"
              aria-label="上一页"
              @click="currentPage -= 1"
            >
              ← 上一页
            </button>
            <strong>第 {{ currentPage }} / {{ totalPages }} 页</strong>
            <button
              type="button"
              :disabled="currentPage === totalPages"
              aria-label="下一页"
              @click="currentPage += 1"
            >
              下一页 →
            </button>
          </div>
        </nav>
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
  grid-template-columns: minmax(12rem, 1.5fr) minmax(11rem, 1fr);
  gap: 1rem;
  margin: 1.4rem 0;
}
.building-filter,
.occupation-filter {
  position: relative;
  display: grid;
  gap: .35rem;
  min-width: 0;
}
.building-filter > span,
.occupation-filter > span {
  color: var(--faded);
  font-family: var(--mono);
  font-size: .68rem;
  letter-spacing: .08em;
}
.filter-picker { position: relative; min-width: 0; }
.filter-picker summary {
  display: flex;
  align-items: center;
  gap: .55rem;
  min-height: 2.8rem;
  border: 1px solid var(--binding);
  border-radius: 4px;
  padding: .65rem .75rem;
  background: var(--paper);
  color: var(--ink);
  font: inherit;
  cursor: pointer;
  list-style: none;
  user-select: none;
}
.filter-picker summary::-webkit-details-marker { display: none; }
.filter-picker summary::after {
  width: .45rem;
  height: .45rem;
  margin: -.2rem .15rem 0 auto;
  border-right: 1.5px solid var(--faded);
  border-bottom: 1.5px solid var(--faded);
  content: '';
  transform: rotate(45deg);
  transition: transform 120ms ease;
}
.filter-picker[open] summary {
  border-color: var(--pine);
  outline: 2px solid color-mix(in srgb, var(--pine) 18%, transparent);
  outline-offset: 1px;
}
.filter-picker[open] summary::after { margin-top: .2rem; transform: rotate(225deg); }
.filter-picker-panel {
  position: absolute;
  z-index: 20;
  top: calc(100% + .45rem);
  right: 0;
  left: 0;
  overflow: hidden;
  border: 1px solid var(--binding);
  border-radius: 6px;
  background: var(--paper);
  box-shadow: 0 .8rem 2rem rgb(47 43 35 / 16%);
}
.building-picker-panel { min-width: min(32rem, calc(100vw - 2rem)); }
.building-picker-actions {
  display: flex;
  justify-content: flex-end;
  gap: .45rem;
  padding: .65rem .7rem;
  border-bottom: 1px solid var(--rule);
}
.building-picker-actions button {
  border: 1px solid var(--binding);
  border-radius: 999px;
  padding: .25rem .65rem;
  background: transparent;
  color: var(--faded);
  font-family: var(--mono);
  font-size: .66rem;
  cursor: pointer;
}
.building-picker-actions button:hover { border-color: var(--pine); color: var(--pine); }
.building-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: .15rem;
  max-height: 18rem;
  overflow-y: auto;
  padding: .55rem;
}
.building-options label {
  position: relative;
  display: flex;
  align-items: center;
  gap: .55rem;
  min-width: 0;
  border-radius: 4px;
  padding: .45rem .5rem;
  cursor: pointer;
}
.building-options label:hover { background: var(--paper-2); }
.building-options input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
}
.building-check {
  display: grid;
  flex: none;
  width: 1rem;
  height: 1rem;
  place-items: center;
  border: 1px solid var(--binding);
  border-radius: 3px;
  background: color-mix(in srgb, var(--paper) 85%, white);
  color: var(--paper);
  font-size: .7rem;
}
.building-options input:checked + .building-check { border-color: var(--pine); background: var(--pine); }
.building-options input:checked + .building-check::after { content: '✓'; }
.building-options input:focus-visible + .building-check { outline: 2px solid var(--blaze); outline-offset: 2px; }
.building-name {
  overflow: hidden;
  color: var(--ink);
  font-size: .78rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.occupation-options { padding: .4rem; }
.occupation-options button {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: .6rem;
  align-items: center;
  width: 100%;
  border: 0;
  border-radius: 4px;
  padding: .55rem .6rem;
  background: transparent;
  color: var(--ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.occupation-options button:hover,
.occupation-options button.is-selected { background: var(--paper-2); }
.occupation-check { color: var(--pine); opacity: 0; }
.occupation-options button.is-selected .occupation-check { opacity: 1; }
.occupation-dot {
  display: inline-block;
  flex: none;
  width: .6rem;
  height: .6rem;
  border-radius: 2px;
  background: var(--binding);
}
.occupation-dot.is-free { background: var(--pine); }
.occupation-dot.is-scheduled { background: var(--blaze); }
.occupation-dot.is-exam { background: #394ed6; }
.occupation-dot.is-laboratory { background: #34712c; }
.occupation-dot.is-rescheduled,
.occupation-dot.is-other { background: #857a66; }
.occupation-dot.is-all {
  background: linear-gradient(135deg, var(--pine) 0 50%, var(--blaze) 50% 100%);
}
.period-editor {
  margin-bottom: 1.5rem;
  border: 1px solid var(--rule);
  border-radius: 6px;
  padding: 1.15rem 1.25rem 1rem;
  background: color-mix(in srgb, var(--paper-2) 48%, transparent);
}
.period-editor-heading {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}
.period-editor-heading > div { display: grid; gap: .2rem; }
.period-editor-heading span,
.period-editor-heading p {
  color: var(--faded);
  font-family: var(--mono);
  font-size: .68rem;
  letter-spacing: .06em;
}
.period-editor-heading strong { color: var(--ink); font-size: 1rem; }
.period-editor-heading p { margin: 0; letter-spacing: 0; }
.period-timeline {
  position: relative;
  height: 2.6rem;
  margin-inline: .55rem;
}
.period-track {
  position: absolute;
  inset: .55rem 0;
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 3px;
}
.period-block {
  border-radius: 3px;
  background: color-mix(in srgb, var(--binding) 55%, transparent);
  transition: background-color 120ms ease;
}
.period-block.is-selected { background: color-mix(in srgb, var(--pine) 45%, var(--paper)); }
.period-selection {
  position: absolute;
  z-index: 2;
  top: -.24rem;
  bottom: -.24rem;
  min-width: 2px;
  border-block: 2px solid var(--pine);
  pointer-events: none;
}
.period-range {
  position: absolute;
  z-index: 3;
  inset: 0;
  width: 100%;
  height: 100%;
  margin: 0;
  appearance: none;
  background: transparent;
  pointer-events: none;
}
.period-range-end { z-index: 4; }
.period-range::-webkit-slider-runnable-track { height: 100%; background: transparent; }
.period-range::-moz-range-track { height: 100%; background: transparent; }
.period-range::-webkit-slider-thumb {
  width: 1.05rem;
  height: 2.6rem;
  border: 2px solid var(--paper);
  border-radius: 4px;
  appearance: none;
  background: var(--pine);
  box-shadow: 0 1px 5px rgb(0 0 0 / 24%);
  cursor: ew-resize;
  pointer-events: auto;
}
.period-range::-moz-range-thumb {
  width: .8rem;
  height: 2.35rem;
  border: 2px solid var(--paper);
  border-radius: 4px;
  background: var(--pine);
  box-shadow: 0 1px 5px rgb(0 0 0 / 24%);
  cursor: ew-resize;
  pointer-events: auto;
}
.period-range:focus-visible::-webkit-slider-thumb { outline: 3px solid var(--blaze); outline-offset: 2px; }
.period-range:focus-visible::-moz-range-thumb { outline: 3px solid var(--blaze); outline-offset: 2px; }
.period-labels {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  margin-inline: .15rem;
  color: var(--faded);
  font-family: var(--mono);
  font-size: .68rem;
  text-align: center;
}
.classroom-message {
  padding: 2rem 1rem;
  border: 1px dashed var(--binding);
  color: var(--faded);
  text-align: center;
}
.classroom-error { color: var(--blaze); }
.classroom-results { display: grid; gap: .8rem; }
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
.room-period-statuses {
  display: flex;
  align-items: center;
  gap: .2rem;
  color: var(--faded);
  font-size: .72rem;
}
.room-period-status {
  width: .42rem;
  height: 1.05rem;
  border-radius: 2px;
  background: var(--binding);
}
.room-period-statuses > span:last-child { margin-left: .35rem; }
.room-period-status.is-free { background: var(--pine); }
.room-period-status.is-scheduled { background: var(--blaze); }
.room-period-status.is-exam { background: #394ed6; }
.room-period-status.is-laboratory { background: #34712c; }
.room-period-status.is-rescheduled,
.room-period-status.is-other { background: #857a66; }
.classroom-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  color: var(--faded);
  font-family: var(--mono);
  font-size: .68rem;
}
.classroom-pagination > div { display: flex; align-items: center; gap: .7rem; }
.classroom-pagination strong { color: var(--ink); font-weight: 500; white-space: nowrap; }
.classroom-pagination button {
  border: 1px solid var(--binding);
  border-radius: 4px;
  padding: .45rem .7rem;
  background: var(--paper);
  color: var(--ink);
  font: inherit;
  cursor: pointer;
}
.classroom-pagination button:hover:not(:disabled) { border-color: var(--pine); color: var(--pine); }
.classroom-pagination button:disabled { cursor: not-allowed; opacity: .4; }

@media (max-width: 720px) {
  .classroom-heading { align-items: stretch; flex-direction: column; gap: 1rem; }
  .classroom-refresh { align-self: flex-start; }
  .classroom-filters { grid-template-columns: 1fr; }
  .classroom-meta strong { width: 100%; margin-left: 0; }
  .building-picker-panel { right: auto; min-width: min(26rem, calc(100vw - 2rem)); }
  .building-options { grid-template-columns: 1fr; }
  .period-editor { padding-inline: .8rem; }
  .period-editor-heading { align-items: start; flex-direction: column; gap: .35rem; }
  .classroom-pagination { align-items: stretch; flex-direction: column; }
  .classroom-pagination > div { justify-content: space-between; }
}
</style>
