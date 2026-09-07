<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { diffTexts, diffStats } from '../diff'
import type { DiffRow } from '../diff'

const left = ref('')
const right = ref('')
const view = ref<'side' | 'unified'>('side')
const rows = ref<DiffRow[]>([])

const stats = computed(() => diffStats(rows.value))
const empty = computed(() => left.value === '' && right.value === '')

let timer: ReturnType<typeof setTimeout> | null = null

function runDiff() {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
  rows.value = diffTexts(left.value, right.value)
}

function scheduleDiff() {
  if (timer) clearTimeout(timer)
  timer = setTimeout(runDiff, 300)
}

watch([left, right], scheduleDiff)

function swap() {
  const t = left.value
  left.value = right.value
  right.value = t
  runDiff()
}

function clearAll() {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
  left.value = ''
  right.value = ''
  rows.value = []
}
</script>

<template>
  <div class="page">
      <div class="tool-page">
        <RouterLink class="back" to="/tools">← 返回工具</RouterLink>
        <h1>文本 Diff</h1>
        <p class="desc">粘贴两段文本，自动逐行比较。新增、删除、改动分别着色，可切换并排或合并视图。</p>

        <div class="diff-inputs">
          <textarea v-model="left" placeholder="旧文本…" aria-label="旧文本" />
          <textarea v-model="right" placeholder="新文本…" aria-label="新文本" />
        </div>

        <div class="controls">
          <span class="stats">
            <template v-if="!empty">
              <span class="st-add">新增 {{ stats.added }} 行</span>
              <span class="st-del">删除 {{ stats.removed }} 行</span>
              <span class="st-same">相同 {{ stats.same }} 行</span>
            </template>
            <span v-else>等待输入…</span>
          </span>
          <span class="actions">
            <button type="button" @click="swap">⇄ 交换</button>
            <button type="button" @click="clearAll">清空</button>
            <span class="view-toggle" role="group" aria-label="视图模式">
              <button type="button" :class="{ active: view === 'side' }" @click="view = 'side'">并排</button>
              <button type="button" :class="{ active: view === 'unified' }" @click="view = 'unified'">合并</button>
            </span>
          </span>
        </div>

        <div v-if="!empty" class="output">
          <div v-if="view === 'side'" class="diff-pane side">
            <div v-for="(row, idx) in rows" :key="idx" class="side-row">
              <span class="cell" :class="{ del: row.type === 'delete' || row.type === 'modify' }">
                <span class="ln">{{ row.leftNo ?? '' }}</span>
                <span class="text">
                  <template v-if="row.type === 'insert'">{{ ' ' }}</template>
                  <template v-else-if="row.type === 'modify'">
                    <span v-for="(s, i) in row.leftSpans" :key="i" :class="s.t">{{ s.text }}</span>
                  </template>
                  <template v-else>{{ row.left }}</template>
                </span>
              </span>
              <span class="cell" :class="{ add: row.type === 'insert' || row.type === 'modify' }">
                <span class="ln">{{ row.rightNo ?? '' }}</span>
                <span class="text">
                  <template v-if="row.type === 'delete'">{{ ' ' }}</template>
                  <template v-else-if="row.type === 'modify'">
                    <span v-for="(s, i) in row.rightSpans" :key="i" :class="s.t">{{ s.text }}</span>
                  </template>
                  <template v-else>{{ row.right }}</template>
                </span>
              </span>
            </div>
          </div>

          <div v-else class="diff-pane unified">
            <template v-for="(row, idx) in rows" :key="idx">
              <div v-if="row.type === 'equal'" class="u-line">
                <span class="u-p"> </span>
                <span class="u-n">{{ row.leftNo }}</span>
                <span class="u-t">{{ row.left }}</span>
              </div>
              <template v-else-if="row.type === 'modify'">
                <div class="u-line del">
                  <span class="u-p">-</span>
                  <span class="u-n">{{ row.leftNo }}</span>
                  <span class="u-t"><span v-for="(s, i) in row.leftSpans" :key="i" :class="s.t">{{ s.text }}</span></span>
                </div>
                <div class="u-line add">
                  <span class="u-p">+</span>
                  <span class="u-n">{{ row.rightNo }}</span>
                  <span class="u-t"><span v-for="(s, i) in row.rightSpans" :key="i" :class="s.t">{{ s.text }}</span></span>
                </div>
              </template>
              <div v-else-if="row.type === 'delete'" class="u-line del">
                <span class="u-p">-</span>
                <span class="u-n">{{ row.leftNo }}</span>
                <span class="u-t">{{ row.left }}</span>
              </div>
              <div v-else class="u-line add">
                <span class="u-p">+</span>
                <span class="u-n">{{ row.rightNo }}</span>
                <span class="u-t">{{ row.right }}</span>
              </div>
            </template>
          </div>
        </div>
      </div>
    </div>
</template>

<style scoped>
.diff-inputs { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }

.controls {
  display: flex; justify-content: space-between; align-items: center; gap: 1rem;
  margin-top: 1rem; flex-wrap: wrap;
}
.stats { font-family: var(--mono); font-size: .68rem; letter-spacing: .06em; color: var(--faded); }
.stats .st-add { color: var(--pine); margin-right: 1em; }
.stats .st-del { color: var(--blaze); margin-right: 1em; }
.actions { display: flex; gap: .5rem; align-items: center; flex-wrap: wrap; }
.actions button, .view-toggle button {
  background: transparent; border: 1px solid var(--binding); border-radius: 3px;
  color: var(--faded); font-family: var(--mono); font-size: .68rem;
  letter-spacing: .08em; text-transform: uppercase; padding: .35rem .85rem; cursor: pointer;
}
.actions button:hover, .view-toggle button:hover { border-color: var(--pine); color: var(--pine); }
.view-toggle { display: inline-flex; }
.view-toggle button.active { background: var(--ink); border-color: var(--ink); color: var(--paper); }

.diff-pane {
  border: 1px solid var(--binding); border-radius: 6px;
  background: var(--paper); padding: .5rem 0; overflow: hidden;
  font-family: var(--mono); font-size: .82rem; line-height: 1.75;
}

.side-row { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
.cell { display: flex; align-items: stretch; min-width: 0; }
.cell + .cell { border-left: 1px solid var(--rule); }
.ln {
  flex-shrink: 0; width: 3em; text-align: right; padding-right: .6em;
  color: var(--faded); font-size: .75em; user-select: none;
  display: flex; align-items: baseline; justify-content: flex-end;
  border-right: 1px solid var(--binding);
}
.text { flex: 1; padding-left: .6em; white-space: pre-wrap; overflow-wrap: anywhere; min-width: 0; }

.cell.del { background: color-mix(in srgb, var(--blaze) 12%, transparent); }
.cell.add { background: color-mix(in srgb, var(--pine) 12%, transparent); }
.cell.del .ln { color: var(--blaze); }
.cell.add .ln { color: var(--pine); }

.text span.del, .u-t span.del { background: color-mix(in srgb, var(--blaze) 26%, transparent); border-radius: 2px; }
.text span.add, .u-t span.add { background: color-mix(in srgb, var(--pine) 26%, transparent); border-radius: 2px; }

.u-line { display: flex; align-items: stretch; min-width: 0; }
.u-p { flex-shrink: 0; width: 1.6em; text-align: center; color: var(--faded); user-select: none; }
.u-n {
  flex-shrink: 0; width: 3em; text-align: right; padding-right: .6em;
  font-size: .75em; color: var(--faded); user-select: none;
  display: flex; align-items: baseline; justify-content: flex-end;
}
.u-t { flex: 1; padding-left: .6em; white-space: pre-wrap; overflow-wrap: anywhere; min-width: 0; }
.u-line.del { background: color-mix(in srgb, var(--blaze) 12%, transparent); }
.u-line.add { background: color-mix(in srgb, var(--pine) 12%, transparent); }
.u-line.del .u-p, .u-line.del .u-n { color: var(--blaze); }
.u-line.add .u-p, .u-line.add .u-n { color: var(--pine); }

@media (max-width: 720px) {
  .diff-inputs { grid-template-columns: 1fr; }
  .side-row { grid-template-columns: 1fr; }
  .side-row .cell + .cell { border-left: 0; border-top: 1px solid var(--rule); }
  .controls { flex-direction: column; align-items: flex-start; }
}
</style>
