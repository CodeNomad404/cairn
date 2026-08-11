<script setup lang="ts">
import { computed, ref } from 'vue'

type Line =
  | { _t: 'open'; depth: number; key?: string; count: number; isArray: boolean; path: string }
  | { _t: 'close'; depth: number; isArray: boolean; path: string }
  | { _t: 'leaf'; depth: number; key?: string; value: unknown; path: string }

const props = withDefaults(defineProps<{
  data: unknown
  maxDepth?: number
}>(), {
  maxDepth: 2,
})

const toggled = ref(new Set<string>())

function isExpanded(path: string, depth: number): boolean {
  return toggled.value.has(path) ? depth >= props.maxDepth : depth < props.maxDepth
}

function walk(val: unknown, depth: number, key: string | undefined, path: string): Line[] {
  if (val === null || typeof val !== 'object') {
    return [{ _t: 'leaf', depth, key, value: val, path }]
  }
  const isArray = Array.isArray(val)
  const entries: [string, unknown][] = isArray
    ? (val as unknown[]).map((v, i) => [String(i), v])
    : Object.entries(val as Record<string, unknown>)
  const count = entries.length

  const open: Line = { _t: 'open', depth, key, count, isArray, path }
  const close: Line = { _t: 'close', depth, isArray, path }
  const children: Line[] = []

  if (isExpanded(path, depth)) {
    for (const [k, v] of entries) {
      children.push(...walk(v, depth + 1, k, path ? `${path}.${k}` : k))
    }
  }

  return [open, ...children, close]
}

const lines = computed<Line[]>(() => {
  toggled.value
  return walk(props.data, 0, undefined, '')
})

function toggle(path: string) {
  const next = new Set(toggled.value)
  if (next.has(path)) next.delete(path)
  else next.add(path)
  toggled.value = next
}

function displayValue(val: unknown): string {
  if (val === null) return 'null'
  if (typeof val === 'string') return `"${val}"`
  return String(val)
}

function valueClass(val: unknown): string {
  if (val === null) return 'value null'
  const t = typeof val
  if (t === 'boolean') return 'value bool'
  if (t === 'string') return 'value string'
  if (t === 'number') return 'value number'
  return 'value'
}

</script>

<template>
  <div class="tree">
    <div
      v-for="(line, idx) in lines"
      :key="idx"
      class="line"
    >
      <!-- line number gutter -->
      <span class="ln">{{ idx + 1 }}</span>

      <!-- indent guides: vertical lines via border-left -->
      <span
        v-for="g in line.depth"
        :key="g"
        class="guide"
      />

      <!-- open line -->
      <template v-if="line._t === 'open'">
        <span
          v-if="line.count"
          class="toggle"
          @click="toggle(line.path)"
        >{{ isExpanded(line.path, line.depth) ? '▾' : '▸' }}</span>
        <span v-else class="toggle-gap" />
        <span v-if="line.key !== undefined" class="key">{{ line.key }}<span class="sep">: </span></span>
        <span class="brace">{{ line.isArray ? '[' + line.count + ']' : '{' + line.count + '}' }}</span>
      </template>

      <!-- close line -->
      <template v-else-if="line._t === 'close'">
        <span class="toggle-gap" />
        <span class="brace closer-brace">{{ line.isArray ? ']' : '}' }}</span>
      </template>

      <!-- leaf line -->
      <template v-else>
        <span class="toggle-gap" />
        <span v-if="line.key !== undefined" class="key">{{ line.key }}<span class="sep">: </span></span>
        <span :class="valueClass(line.value)">{{ displayValue(line.value) }}</span>
      </template>
    </div>
  </div>
</template>

<style scoped>
.tree {
  font-family: var(--mono);
  font-size: 0.82rem;
  line-height: 1.75;
  overflow-x: auto;
}
.line {
  display: flex;
  align-items: stretch;
  height: 1.75em;
}
/* ---- line number ---- */
.ln {
  flex-shrink: 0;
  width: 3em;
  text-align: right;
  padding-right: 0.6em;
  color: var(--faded);
  font-size: 0.75em;
  user-select: none;
  display: flex;
  align-items: baseline;
  justify-content: flex-end;
  border-right: 1px solid var(--binding);
}
/* ---- indent guide ---- */
.guide {
  flex-shrink: 0;
  width: 1.5em;
  position: relative;
}
.guide::after {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  border-left: 1px solid var(--binding);
}
/* ---- content ---- */
.toggle {
  flex-shrink: 0;
  width: 1em;
  cursor: pointer;
  user-select: none;
  color: var(--faded);
  display: flex;
  align-items: baseline;
}
.toggle:hover { color: var(--ink); }
.toggle-gap {
  flex-shrink: 0;
  width: 1em;
}
.key { color: #859900; white-space: nowrap; }
.sep { color: var(--faded); }
.brace { color: var(--faded); }
.closer-brace { color: var(--faded); }
.value { color: #2aa198; font-weight: 500; }
.value.string { color: #268bd2; }
.value.number { color: #2aa198; }
.value.bool { color: #cb4b16; }
.value.null { color: #6c71c4; }
</style>
