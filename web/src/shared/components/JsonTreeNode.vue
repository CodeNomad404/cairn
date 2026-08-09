<script setup lang="ts">
import { ref } from 'vue'

const props = withDefaults(defineProps<{
  keyName?: string
  value: unknown
  depth: number
  maxDepth: number
}>(), {
  keyName: undefined,
})

const open = ref(props.depth < props.maxDepth)

function toggle() {
  open.value = !open.value
}

const type = typeof props.value
const isObject = type === 'object' && props.value !== null && !Array.isArray(props.value)
const isArray = Array.isArray(props.value)

function entries(): [string, unknown][] {
  if (isArray) return (props.value as unknown[]).map((v, i) => [String(i), v])
  if (isObject) return Object.entries(props.value as Record<string, unknown>)
  return []
}

function displayValue(val: unknown): string {
  if (val === null) return 'null'
  if (typeof val === 'string') return `"${val}"`
  return String(val)
}
</script>

<template>
  <div class="node">
    <span v-if="isObject || isArray" class="toggle" @click="toggle" :style="{ visibility: entries().length ? 'visible' : 'hidden' }">
      {{ open ? '▾' : '▸' }}
    </span>

    <span v-if="keyName !== undefined" class="key">{{ keyName }}<span class="sep">: </span></span>

    <!-- Object -->
    <template v-if="isObject">
      <span class="brace">{{ open && entries().length ? '{…}' : `{${entries().length}…}` }}</span>
      <template v-if="open">
        <div v-for="([k, v]) in entries()" :key="k" class="children">
          <JsonTreeNode :key-name="k" :value="v" :depth="depth + 1" :max-depth="maxDepth" />
        </div>
        <div class="closer">}</div>
      </template>
    </template>

    <!-- Array -->
    <template v-else-if="isArray">
      <span class="brace">{{ open && entries().length ? '[…]' : `[${entries().length}…]` }}</span>
      <template v-if="open">
        <div v-for="([, v], i) in entries()" :key="i" class="children">
          <JsonTreeNode :key-name="String(i)" :value="v" :depth="depth + 1" :max-depth="maxDepth" />
        </div>
        <div class="closer">]</div>
      </template>
    </template>

    <!-- Primitive -->
    <span v-else :class="['value', { null: props.value === null, bool: typeof props.value === 'boolean', string: typeof props.value === 'string', number: typeof props.value === 'number' }]">
      {{ displayValue(props.value) }}
    </span>
  </div>
</template>

<style scoped>
.node {
  font-family: var(--mono);
  font-size: 0.82rem;
  line-height: 1.7;
}
.toggle {
  display: inline-block;
  width: 1em;
  cursor: pointer;
  user-select: none;
  color: var(--faded);
  transition: transform .12s;
}
.toggle:hover { color: var(--ink); }
.key {
  color: #859900;
}
.sep { color: var(--faded); }
.brace {
  color: var(--faded);
  font-style: italic;
}
.children {
  padding-left: 1.5em;
}
.closer {
  color: var(--faded);
}
.value { color: #2aa198; font-weight: 500; }
.value.string { color: #268bd2; }
.value.number { color: #2aa198; }
.value.bool { color: #cb4b16; }
.value.null { color: #6c71c4; }
</style>
